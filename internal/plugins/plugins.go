package plugins

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"

	"github.com/nzlov/ra/internal/pluginruntime"
	"github.com/nzlov/ra/pkg/raplugin"
)

var idPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-_.]*$`)
var runPluginSearch = pluginruntime.Search

type Registry struct {
	Root    string
	Roots   []string
	Plugins []Plugin
	Errors  []LoadError
}

type Root struct {
	Path   string
	Source string
}

type BuiltinPlugin struct {
	Name string
	Raw  []byte
}

type Plugin struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Version      string            `json:"version"`
	Permissions  []string          `json:"permissions,omitempty"`
	Capabilities []Capability      `json:"capabilities,omitempty"`
	Assets       map[string][]byte `json:"-"`
	Raw          []byte            `json:"-"`
	Source       string            `json:"source,omitempty"`
	Path         string            `json:"-"`
	Disabled     bool              `json:"-"`
}

type Capability struct {
	ID       string   `json:"id"`
	Title    string   `json:"title"`
	Icon     string   `json:"icon,omitempty"`
	UI       string   `json:"ui"`
	Keywords []string `json:"keywords,omitempty"`
	Disabled bool     `json:"-"`
}

type LoadError struct {
	Path  string
	Error string
}

type Result struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Subtitle string `json:"subtitle"`
	Kind     string `json:"kind"`
	Action   Action `json:"action"`
}

type Action struct {
	Type         string `json:"type"`
	AppID        string `json:"appId,omitempty"`
	Text         string `json:"text,omitempty"`
	PluginID     string `json:"pluginId"`
	CapabilityID string `json:"capabilityId,omitempty"`
	UI           string `json:"ui,omitempty"`
	Query        string `json:"query,omitempty"`
}

type SearchRequest struct {
	Query   string
	Limit   int
	HostAPI HostAPI
}

type HostAPI struct {
	Apps []raplugin.App
}

func LoadRegistry(root string) (Registry, error) {
	return LoadRegistries([]string{root})
}

func LoadRegistries(roots []string) (Registry, error) {
	items := make([]Root, 0, len(roots))
	for i, root := range roots {
		source := "builtin"
		if i > 0 {
			source = "user"
		}
		items = append(items, Root{Path: root, Source: source})
	}
	return LoadRegistriesWithSources(items, nil)
}

func LoadRegistriesWithSources(roots []Root, builtins []BuiltinPlugin) (Registry, error) {
	rootPaths := make([]string, 0, len(roots))
	for _, root := range roots {
		rootPaths = append(rootPaths, root.Path)
	}
	registry := Registry{Roots: rootPaths}
	if len(roots) == 1 {
		registry.Root = roots[0].Path
	}
	seenRoots := map[string]struct{}{}
	seenPlugins := map[string]string{}
	for _, builtin := range builtins {
		plugin, err := loadWASMBytes(builtin.Raw, "builtin", builtin.Name)
		if err != nil {
			registry.Errors = append(registry.Errors, LoadError{Path: builtin.Name, Error: err.Error()})
			continue
		}
		addPlugin(plugin, &registry, seenPlugins)
	}
	for _, root := range roots {
		if root.Path == "" {
			continue
		}
		if _, ok := seenRoots[root.Path]; ok {
			continue
		}
		seenRoots[root.Path] = struct{}{}
		source := root.Source
		if source == "" {
			source = "builtin"
		}
		if err := loadRoot(root.Path, source, &registry, seenPlugins); err != nil {
			return registry, err
		}
	}

	sort.SliceStable(registry.Plugins, func(i, j int) bool {
		return registry.Plugins[i].Name < registry.Plugins[j].Name
	})
	return registry, nil
}

func loadRoot(root string, source string, registry *Registry, seenPlugins map[string]string) error {
	items, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, item := range items {
		if item.IsDir() || filepath.Ext(item.Name()) != ".wasm" {
			continue
		}
		pluginPath := filepath.Join(root, item.Name())
		plugin, err := loadWASMFile(pluginPath, source)
		if err != nil {
			registry.Errors = append(registry.Errors, LoadError{Path: pluginPath, Error: err.Error()})
			continue
		}
		addPlugin(plugin, registry, seenPlugins)
	}
	return nil
}

func addPlugin(plugin Plugin, registry *Registry, seenPlugins map[string]string) {
	if firstPath, ok := seenPlugins[plugin.ID]; ok {
		registry.Errors = append(registry.Errors, LoadError{
			Path:  plugin.Path,
			Error: fmt.Sprintf("id conflict for %q: already loaded from %s", plugin.ID, firstPath),
		})
		return
	}
	seenPlugins[plugin.ID] = plugin.Path
	registry.Plugins = append(registry.Plugins, plugin)
}

func (r Registry) Search(query string, limit int) []Result {
	return r.SearchWithContext(SearchRequest{Query: query, Limit: limit})
}

func (r Registry) SearchWithContext(request SearchRequest) []Result {
	trimmed := strings.TrimSpace(request.Query)
	type searchResultSet struct {
		index   int
		plugin  Plugin
		results []raplugin.SearchResult
	}
	items := make([]searchResultSet, 0, len(r.Plugins))
	sem := make(chan struct{}, searchConcurrencyLimit(len(r.Plugins)))
	var wg sync.WaitGroup
	var mu sync.Mutex
	for i, plugin := range r.Plugins {
		if plugin.Disabled {
			continue
		}
		wg.Add(1)
		go func(index int, plugin Plugin) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			rawResults, err := runPluginSearch(plugin.Raw, raplugin.SearchRequest{
				Query: trimmed,
				Limit: request.Limit,
			}, pluginruntime.HostAPI{
				Permissions: append([]string(nil), plugin.Permissions...),
				Apps:        append([]raplugin.App(nil), request.HostAPI.Apps...),
			})
			if err != nil {
				return
			}
			mu.Lock()
			items = append(items, searchResultSet{
				index:   index,
				plugin:  plugin,
				results: rawResults,
			})
			mu.Unlock()
		}(i, plugin)
	}
	wg.Wait()
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].index < items[j].index
	})

	var results []Result
	for _, item := range items {
		for _, result := range item.results {
			if !pluginCapabilityEnabled(item.plugin, result.Action.CapabilityID) {
				continue
			}
			results = append(results, resultFromPlugin(item.plugin, result, trimmed))
		}
	}
	if request.Limit > 0 && len(results) > request.Limit {
		return results[:request.Limit]
	}
	return results
}

func searchConcurrencyLimit(pluginCount int) int {
	if pluginCount < 1 {
		return 1
	}
	limit := runtime.GOMAXPROCS(0)
	if limit < 1 {
		return 1
	}
	if pluginCount < limit {
		return pluginCount
	}
	return limit
}

func pluginCapabilityEnabled(plugin Plugin, capabilityID string) bool {
	if capabilityID == "" {
		return false
	}
	for _, capability := range plugin.Capabilities {
		if capability.ID == capabilityID {
			return !capability.Disabled
		}
	}
	return false
}

func resultFromPlugin(plugin Plugin, result raplugin.SearchResult, query string) Result {
	action := Action{
		Type:         result.Action.Type,
		AppID:        result.Action.AppID,
		Text:         result.Action.Text,
		PluginID:     plugin.ID,
		CapabilityID: result.Action.CapabilityID,
		Query:        result.Action.Query,
	}
	if action.Query == "" {
		action.Query = query
	}
	if action.Type == "" {
		action.Type = "capability.open"
	}
	if capability, ok := findCapability(plugin.Capabilities, action.CapabilityID); ok {
		action.UI = capability.UI
	}
	id := result.ID
	if id == "" {
		id = "capability:" + plugin.ID + ":" + action.CapabilityID
	}
	kind := result.Kind
	if kind == "" {
		kind = "capability"
	}
	return Result{
		ID:       id,
		Title:    result.Title,
		Subtitle: result.Subtitle,
		Kind:     kind,
		Action:   action,
	}
}

func findCapability(capabilities []Capability, id string) (Capability, bool) {
	for _, capability := range capabilities {
		if capability.ID == id {
			return capability, true
		}
	}
	return Capability{}, false
}

func loadWASMFile(path string, source string) (Plugin, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Plugin{}, err
	}
	return loadWASMBytes(raw, source, path)
}

func LoadPluginFile(path string) (Plugin, error) {
	return loadWASMFile(path, "user")
}

func loadWASMBytes(raw []byte, source string, sourcePath string) (Plugin, error) {
	bundle, err := pluginruntime.Load(raw)
	if err != nil {
		return Plugin{}, err
	}
	if err := validateBundle(bundle); err != nil {
		return Plugin{}, err
	}
	return Plugin{
		ID:           bundle.Manifest.ID,
		Name:         bundle.Manifest.Name,
		Version:      bundle.Manifest.Version,
		Permissions:  append([]string(nil), bundle.Manifest.Permissions...),
		Capabilities: capabilitiesFromBundle(bundle.Capabilities),
		Assets:       cloneAssets(bundle.Assets),
		Raw:          append([]byte(nil), raw...),
		Source:       source,
		Path:         sourcePath,
	}, nil
}

func validateBundle(bundle pluginruntime.Plugin) error {
	if !validID(bundle.Manifest.ID) {
		return fmt.Errorf("invalid plugin id %q", bundle.Manifest.ID)
	}
	if strings.TrimSpace(bundle.Manifest.Name) == "" {
		return errors.New("missing plugin name")
	}
	seenCapabilities := map[string]struct{}{}
	for _, capability := range bundle.Capabilities {
		if !validID(capability.ID) {
			return fmt.Errorf("invalid capability id %q", capability.ID)
		}
		if _, ok := seenCapabilities[capability.ID]; ok {
			return fmt.Errorf("duplicate capability id %q", capability.ID)
		}
		seenCapabilities[capability.ID] = struct{}{}
		if !validAssetPath(capability.UI) || path.Ext(capability.UI) != ".html" {
			return fmt.Errorf("invalid capability UI path %q", capability.UI)
		}
		if _, ok := bundle.Assets[capability.UI]; !ok {
			return fmt.Errorf("missing capability UI asset %q", capability.UI)
		}
		if capability.Icon != "" && !validAssetPath(capability.Icon) {
			return fmt.Errorf("invalid capability icon path %q", capability.Icon)
		}
	}
	for assetPath := range bundle.Assets {
		if !validAssetPath(assetPath) {
			return fmt.Errorf("invalid asset path %q", assetPath)
		}
	}
	return nil
}

func validID(id string) bool {
	return idPattern.MatchString(id)
}

func validAssetPath(assetPath string) bool {
	return strings.HasPrefix(assetPath, "/") && path.Clean(assetPath) == assetPath && !strings.Contains(assetPath, "\x00")
}

func capabilitiesFromBundle(items []raplugin.Capability) []Capability {
	capabilities := make([]Capability, 0, len(items))
	for _, item := range items {
		capabilities = append(capabilities, Capability{
			ID:       item.ID,
			Title:    item.Title,
			Icon:     item.Icon,
			UI:       item.UI,
			Keywords: append([]string(nil), item.Keywords...),
		})
	}
	return capabilities
}

func cloneAssets(assets map[string][]byte) map[string][]byte {
	out := make(map[string][]byte, len(assets))
	for path, data := range assets {
		out[path] = append([]byte(nil), data...)
	}
	return out
}
