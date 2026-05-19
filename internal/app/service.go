package app

import (
	"bytes"
	"errors"
	"fmt"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/nzlov/ra/internal/desktop"
	"github.com/nzlov/ra/internal/plugins"
	builtinplugins "github.com/nzlov/ra/plugins"
)

type Config struct {
	PluginRoot       string
	PluginRoots      []string
	UserPluginRoot   string
	PluginConfigPath string
	DesktopDirs      []string
	Limit            int
	BuiltinPlugins   []plugins.BuiltinPlugin
}

type LauncherService struct {
	config         Config
	desktopEntries []desktop.Entry
	pluginRegistry plugins.Registry
	pluginConfig   PluginConfig
	actions        ActionExecutor
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
	Command      string `json:"command,omitempty"`
	Text         string `json:"text,omitempty"`
	PluginID     string `json:"pluginId,omitempty"`
	CapabilityID string `json:"capabilityId,omitempty"`
	UI           string `json:"ui,omitempty"`
	Query        string `json:"query,omitempty"`
}

type InvokeResult struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

type PluginActionRequest struct {
	PluginID     string `json:"pluginId"`
	CapabilityID string `json:"capabilityId"`
	Action       Action `json:"action"`
}

type Status struct {
	AppCount         int      `json:"appCount"`
	PluginCount      int      `json:"pluginCount"`
	PluginErrorCount int      `json:"pluginErrorCount"`
	PluginRoots      []string `json:"pluginRoots"`
}

func NewLauncherService(config Config) *LauncherService {
	if len(config.PluginRoots) == 0 {
		if config.PluginRoot != "" {
			config.PluginRoots = []string{config.PluginRoot}
		} else {
			home, _ := os.UserHomeDir()
			config.PluginRoots = defaultPluginRoots(home)
		}
	}
	if config.UserPluginRoot == "" {
		home, _ := os.UserHomeDir()
		config.UserPluginRoot = defaultUserPluginRoot(home)
	}
	if config.PluginConfigPath == "" {
		home, _ := os.UserHomeDir()
		config.PluginConfigPath = defaultPluginConfigPath(home)
	}
	if config.PluginRoot == "" && len(config.PluginRoots) > 0 {
		config.PluginRoot = config.PluginRoots[0]
	}
	if config.Limit == 0 {
		config.Limit = 20
	}
	if len(config.DesktopDirs) == 0 {
		home, _ := os.UserHomeDir()
		config.DesktopDirs = desktop.DefaultDirs(home)
	}
	if config.BuiltinPlugins == nil {
		config.BuiltinPlugins = builtinplugins.List()
	}
	return &LauncherService{config: config, actions: NewActionExecutor()}
}

func NewDefaultLauncherService() *LauncherService {
	service := NewLauncherService(Config{})
	_ = service.Refresh()
	return service
}

func (s *LauncherService) Refresh() error {
	entries, err := desktop.LoadDirs(s.config.DesktopDirs)
	if err != nil {
		return err
	}
	s.desktopEntries = entries
	return s.RefreshPlugins()
}

func (s *LauncherService) RefreshPlugins() error {
	config, err := readPluginConfig(s.config.PluginConfigPath)
	if err != nil {
		return err
	}
	registry, err := plugins.LoadRegistriesWithSources(pluginRootSources(s.config.PluginRoots, s.config.UserPluginRoot), s.config.BuiltinPlugins)
	if err != nil {
		return err
	}
	registry.Plugins = rejectReservedUserPlugins(registry.Plugins, &registry)
	for i := range registry.Plugins {
		registry.Plugins[i].Disabled = containsString(config.Disabled, registry.Plugins[i].ID)
		if registry.Plugins[i].ID == pluginManagerID {
			registry.Plugins[i].Disabled = false
		}
		for j := range registry.Plugins[i].Capabilities {
			registry.Plugins[i].Capabilities[j].Disabled = containsString(config.DisabledCapabilities, capabilityKey(registry.Plugins[i].ID, registry.Plugins[i].Capabilities[j].ID))
			if registry.Plugins[i].ID == pluginManagerID && registry.Plugins[i].Capabilities[j].ID == "manage" {
				registry.Plugins[i].Capabilities[j].Disabled = false
			}
		}
	}
	s.pluginConfig = config
	s.pluginRegistry = registry
	return nil
}

func (s *LauncherService) setDesktopEntries(entries []desktop.Entry) {
	s.desktopEntries = entries
}

func (s *LauncherService) Search(query string) []Result {
	results := make([]Result, 0, s.config.Limit)
	if s.capabilityEnabled(appLauncherPluginID, "apps") {
		for _, entry := range desktop.Search(s.desktopEntries, query, s.config.Limit) {
			results = append(results, Result{
				ID:       "app:" + entry.ID,
				Title:    entry.Name,
				Subtitle: entry.Comment,
				Kind:     "app",
				Action: Action{
					Type:         "app.launch",
					AppID:        entry.ID,
					Command:      entry.LaunchCommand(),
					PluginID:     appLauncherPluginID,
					CapabilityID: "apps",
				},
			})
		}
	}

	remaining := s.config.Limit - len(results)
	if remaining <= 0 {
		return results
	}
	for _, result := range s.searchCapabilities(query, remaining) {
		results = append(results, Result{
			ID:       result.ID,
			Title:    result.Title,
			Subtitle: result.Subtitle,
			Kind:     result.Kind,
			Action: Action{
				Type:         actionType(result),
				PluginID:     result.Action.PluginID,
				CapabilityID: result.Action.CapabilityID,
				UI:           result.Action.UI,
				Query:        result.Action.Query,
			},
		})
	}
	return results
}

func (s *LauncherService) searchCapabilities(query string, limit int) []plugins.Result {
	registry := s.pluginRegistry
	filtered := make([]plugins.Plugin, 0, len(registry.Plugins))
	for _, plugin := range registry.Plugins {
		if plugin.ID == appLauncherPluginID {
			continue
		}
		filtered = append(filtered, plugin)
	}
	registry.Plugins = filtered
	return registry.Search(query, limit)
}

func (s *LauncherService) Invoke(action Action) (InvokeResult, error) {
	if action.Type == "app.launch" {
		entry, ok := s.findDesktopEntry(action.AppID)
		if !ok {
			return InvokeResult{}, fmt.Errorf("app %q is not loaded", action.AppID)
		}
		if !s.capabilityEnabled(appLauncherPluginID, "apps") {
			return InvokeResult{}, fmt.Errorf("capability %q.%q is not loaded", appLauncherPluginID, "apps")
		}
		if action.Command != entry.LaunchCommand() {
			return InvokeResult{}, errors.New("launch command does not match loaded app")
		}
	}
	if action.Type == "clipboard.write" {
		return InvokeResult{}, errors.New("clipboard.write is only available to plugin capabilities")
	}
	return s.actions.Invoke(action)
}

func (s *LauncherService) InvokePluginAction(request PluginActionRequest) (InvokeResult, error) {
	plugin, ok := s.findPlugin(request.PluginID)
	if !ok {
		return InvokeResult{}, errPluginNotLoaded(request.PluginID)
	}
	if !s.capabilityEnabled(request.PluginID, request.CapabilityID) {
		return InvokeResult{}, errCapabilityNotLoaded(request.PluginID, request.CapabilityID)
	}
	permission, ok := permissionForAction(request.Action.Type)
	if !ok {
		return InvokeResult{}, errUnsupportedPluginAction(request.Action.Type)
	}
	if !containsString(plugin.Permissions, permission) {
		return InvokeResult{}, errMissingPluginPermission(plugin.ID, permission)
	}
	return s.actions.Invoke(request.Action)
}

func (s *LauncherService) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	pluginID, capabilityID, requestAssetPath, ok := parseCapabilityAssetPath(req.URL.Path)
	if !ok || !s.capabilityEnabled(pluginID, capabilityID) {
		http.NotFound(rw, req)
		return
	}
	plugin, ok := s.findPlugin(pluginID)
	if !ok {
		http.NotFound(rw, req)
		return
	}
	assetPath, ok := pluginAssetPath(plugin.Assets, requestAssetPath)
	if !ok || !capabilityOwnsAsset(plugin.Capabilities, capabilityID, assetPath) {
		http.NotFound(rw, req)
		return
	}
	raw, ok := plugin.Assets[assetPath]
	if !ok {
		http.NotFound(rw, req)
		return
	}
	if contentType := mime.TypeByExtension(path.Ext(assetPath)); contentType != "" {
		rw.Header().Set("Content-Type", contentType)
	}
	setPluginSecurityHeaders(rw)
	if isHTMLAsset(assetPath, raw) {
		raw = injectPluginBridge(raw)
	}
	_, _ = rw.Write(raw)
}

func (s *LauncherService) Status() Status {
	return Status{
		AppCount:         len(s.desktopEntries),
		PluginCount:      len(s.pluginRegistry.Plugins),
		PluginErrorCount: len(s.pluginRegistry.Errors),
		PluginRoots:      append([]string(nil), s.config.PluginRoots...),
	}
}

func defaultPluginRoots(home string) []string {
	roots := []string{"plugins"}
	if home != "" {
		roots = append(roots, defaultUserPluginRoot(home))
	}
	return roots
}

func defaultUserPluginRoot(home string) string {
	if home == "" {
		return filepath.Join(".local", "share", "ra", "plugins")
	}
	return filepath.Join(home, ".local", "share", "ra", "plugins")
}

func defaultPluginConfigPath(home string) string {
	if home == "" {
		return filepath.Join(".config", "ra", "plugins.json")
	}
	return filepath.Join(home, ".config", "ra", "plugins.json")
}

func pluginRootSources(roots []string, userPluginRoot string) []plugins.Root {
	items := make([]plugins.Root, 0, len(roots))
	for _, root := range roots {
		source := "builtin"
		if samePath(root, userPluginRoot) {
			source = "user"
		}
		items = append(items, plugins.Root{Path: root, Source: source})
	}
	return items
}

func samePath(a string, b string) bool {
	if a == "" || b == "" {
		return false
	}
	absA, err := filepath.Abs(a)
	if err != nil {
		return false
	}
	absB, err := filepath.Abs(b)
	if err != nil {
		return false
	}
	return filepath.Clean(absA) == filepath.Clean(absB)
}

func parseCapabilityAssetPath(requestPath string) (string, string, string, bool) {
	parts := strings.Split(strings.TrimPrefix(path.Clean("/"+requestPath), "/"), "/")
	if len(parts) > 0 && parts[0] == "plugins" {
		parts = parts[1:]
	}
	if len(parts) < 3 {
		return "", "", "", false
	}
	return parts[0], parts[1], "/" + strings.Join(parts[2:], "/"), true
}

func capabilityOwnsAsset(capabilities []plugins.Capability, capabilityID string, assetPath string) bool {
	for _, capability := range capabilities {
		if capability.ID != capabilityID {
			continue
		}
		if assetPath == capability.UI || assetPath == capability.Icon {
			return true
		}
		if strings.HasPrefix(assetPath, capabilityAssetPrefix(capability.UI)) {
			return true
		}
		return path.Ext(assetPath) != ".html"
	}
	return false
}

func capabilityAssetPrefix(uiPath string) string {
	dir := path.Dir(uiPath)
	if dir == "/" {
		return "/"
	}
	return dir + "/"
}

func pluginAssetPath(assets map[string][]byte, requestAssetPath string) (string, bool) {
	if _, ok := assets[requestAssetPath]; ok {
		return requestAssetPath, true
	}
	return "", false
}

func isHTMLAsset(assetPath string, raw []byte) bool {
	return path.Ext(assetPath) == ".html" || bytes.Contains(raw[:min(len(raw), 256)], []byte("<html"))
}

func setPluginSecurityHeaders(rw http.ResponseWriter) {
	rw.Header().Set("X-Content-Type-Options", "nosniff")
	rw.Header().Set("Content-Security-Policy", "default-src 'none'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self' data:; connect-src 'none'; form-action 'none'; base-uri 'none'")
}

func injectPluginBridge(raw []byte) []byte {
	script := []byte(`<script>
window.ra = {
  invoke(action) {
    return new Promise((resolve, reject) => {
      const id = Math.random().toString(36).slice(2);
      function onMessage(event) {
        const message = event.data || {};
        if (message.ra !== 'response' || message.id !== id) return;
        window.removeEventListener('message', onMessage);
        if (message.error) reject(new Error(message.error));
        else resolve(message.result);
      }
      window.addEventListener('message', onMessage);
      window.parent.postMessage({ra: 'invoke', id, action}, '*');
    });
  }
};
</script>`)
	if bytes.Contains(raw, []byte("</head>")) {
		return bytes.Replace(raw, []byte("</head>"), append(script, []byte("</head>")...), 1)
	}
	return append(script, raw...)
}

func permissionForAction(actionType string) (string, bool) {
	switch actionType {
	case "clipboard.write":
		return "clipboard:write", true
	default:
		return "", false
	}
}

func errPluginNotLoaded(pluginID string) error {
	return fmt.Errorf("plugin %q is not loaded", pluginID)
}

func errCapabilityNotLoaded(pluginID string, capabilityID string) error {
	return fmt.Errorf("capability %q.%q is not loaded", pluginID, capabilityID)
}

func errMissingPluginPermission(pluginID string, permission string) error {
	return fmt.Errorf("plugin %q does not declare permission %q", pluginID, permission)
}

func errUnsupportedPluginAction(actionType string) error {
	if actionType == "" {
		return errors.New("missing plugin action type")
	}
	return fmt.Errorf("plugin action %q is not supported", actionType)
}

func (s *LauncherService) pluginEnabled(id string) bool {
	plugin, ok := s.findPlugin(id)
	return ok && !plugin.Disabled
}

func (s *LauncherService) capabilityEnabled(pluginID string, capabilityID string) bool {
	if !s.pluginEnabled(pluginID) {
		return false
	}
	capability, ok := s.findCapability(pluginID, capabilityID)
	return ok && !capability.Disabled
}

func rejectReservedUserPlugins(items []plugins.Plugin, registry *plugins.Registry) []plugins.Plugin {
	filtered := items[:0]
	for _, plugin := range items {
		if plugin.Source != "builtin" && isBuiltinPluginID(plugin.ID) {
			registry.Errors = append(registry.Errors, plugins.LoadError{
				Path:  plugin.Path,
				Error: "id conflict for reserved plugin id \"" + plugin.ID + "\"",
			})
			continue
		}
		filtered = append(filtered, plugin)
	}
	return filtered
}

func isBuiltinPluginID(id string) bool {
	return id == pluginManagerID || id == appLauncherPluginID || id == calculatorPluginID
}

func actionType(result plugins.Result) string {
	if result.Action.PluginID == pluginManagerID && result.Action.CapabilityID == "manage" {
		return "plugin.manage"
	}
	return result.Action.Type
}
