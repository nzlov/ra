package plugins

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/nzlov/ra/internal/pluginbundle"
)

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
	PluginID     string `json:"pluginId"`
	CapabilityID string `json:"capabilityId,omitempty"`
	UI           string `json:"ui,omitempty"`
	Query        string `json:"query,omitempty"`
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
	trimmed := strings.TrimSpace(query)
	tokens := strings.Fields(strings.ToLower(trimmed))
	var results []Result
	for _, plugin := range r.Plugins {
		if plugin.Disabled {
			continue
		}
		for _, capability := range plugin.Capabilities {
			if capability.Disabled {
				continue
			}
			text := strings.ToLower(plugin.Name + " " + capability.Title + " " + strings.Join(capability.Keywords, " "))
			if len(tokens) > 0 && !matchesAnyToken(text, tokens) {
				continue
			}
			results = append(results, Result{
				ID:       "capability:" + plugin.ID + ":" + capability.ID,
				Title:    capability.Title,
				Subtitle: plugin.Name,
				Kind:     "capability",
				Action: Action{
					Type:         "capability.open",
					PluginID:     plugin.ID,
					CapabilityID: capability.ID,
					UI:           capability.UI,
					Query:        trimmed,
				},
			})
		}
	}
	if limit > 0 && len(results) > limit {
		return results[:limit]
	}
	return results
}

func matchesAnyToken(text string, tokens []string) bool {
	words := strings.Fields(text)
	for _, token := range tokens {
		if strings.Contains(text, token) {
			return true
		}
		for _, word := range words {
			if word != "" && strings.Contains(token, word) {
				return true
			}
		}
	}
	return false
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
	bundle, err := pluginbundle.Read(raw)
	if err != nil {
		return Plugin{}, err
	}
	return Plugin{
		ID:           bundle.Manifest.ID,
		Name:         bundle.Manifest.Name,
		Version:      bundle.Manifest.Version,
		Permissions:  append([]string(nil), bundle.Manifest.Permissions...),
		Capabilities: capabilitiesFromBundle(bundle.Capabilities),
		Assets:       cloneAssets(bundle.Assets),
		Source:       source,
		Path:         sourcePath,
	}, nil
}

func capabilitiesFromBundle(items []pluginbundle.Capability) []Capability {
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
