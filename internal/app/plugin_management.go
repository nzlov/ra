package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/nzlov/ra/internal/plugins"
)

const (
	pluginManagerID     = "ra-plugin-manager"
	appLauncherPluginID = "ra-app-launcher"
)

type ManagedPlugin struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Type          string `json:"type"`
	Source        string `json:"source"`
	Dir           string `json:"dir"`
	EntryPath     string `json:"entryPath"`
	Enabled       bool   `json:"enabled"`
	Protected     bool   `json:"protected"`
	Uninstallable bool   `json:"uninstallable"`
}

type ManagedLoadError struct {
	Path  string `json:"path"`
	Error string `json:"error"`
}

type PluginManagerState struct {
	Plugins          []ManagedPlugin    `json:"plugins"`
	LoadErrors       []ManagedLoadError `json:"loadErrors"`
	PluginRoots      []string           `json:"pluginRoots"`
	UserPluginRoot   string             `json:"userPluginRoot"`
	PluginConfigPath string             `json:"pluginConfigPath"`
}

type InstallPluginResult struct {
	PluginID      string             `json:"pluginId"`
	InstalledPath string             `json:"installedPath"`
	State         PluginManagerState `json:"state"`
}

type PluginConfig struct {
	Disabled []string `json:"disabled"`
}

func (s *LauncherService) PluginManagerState() PluginManagerState {
	state := PluginManagerState{
		PluginRoots:      append([]string(nil), s.config.PluginRoots...),
		UserPluginRoot:   s.config.UserPluginRoot,
		PluginConfigPath: s.config.PluginConfigPath,
	}
	for _, plugin := range s.pluginRegistry.Plugins {
		state.Plugins = append(state.Plugins, managedPlugin(plugin))
	}
	for _, loadError := range s.pluginRegistry.Errors {
		state.LoadErrors = append(state.LoadErrors, ManagedLoadError{
			Path:  loadError.Path,
			Error: loadError.Error,
		})
	}
	sort.SliceStable(state.Plugins, func(i, j int) bool {
		if state.Plugins[i].Source != state.Plugins[j].Source {
			return state.Plugins[i].Source < state.Plugins[j].Source
		}
		return state.Plugins[i].Name < state.Plugins[j].Name
	})
	return state
}

func (s *LauncherService) SetPluginEnabled(id string, enabled bool) (PluginManagerState, error) {
	if id == pluginManagerID && !enabled {
		return s.PluginManagerState(), errors.New("plugin manager cannot be disabled")
	}
	if _, ok := s.findPlugin(id); !ok {
		return s.PluginManagerState(), fmt.Errorf("plugin %q is not loaded", id)
	}

	config := s.pluginConfig
	if enabled {
		config.Disabled = removeString(config.Disabled, id)
	} else if !containsString(config.Disabled, id) {
		config.Disabled = append(config.Disabled, id)
	}
	sort.Strings(config.Disabled)
	if err := writePluginConfig(s.config.PluginConfigPath, config); err != nil {
		return s.PluginManagerState(), err
	}
	s.pluginConfig = config
	if err := s.RefreshPlugins(); err != nil {
		return s.PluginManagerState(), err
	}
	return s.PluginManagerState(), nil
}

func (s *LauncherService) InstallPlugin(sourceDir string) (InstallPluginResult, error) {
	sourceDir = filepath.Clean(sourceDir)
	plugin, err := plugins.LoadPluginPackage(sourceDir)
	if err != nil {
		return InstallPluginResult{State: s.PluginManagerState()}, fmt.Errorf("read plugin package: %w", err)
	}
	if isBuiltinPluginID(plugin.ID) {
		return InstallPluginResult{State: s.PluginManagerState()}, fmt.Errorf("built-in plugin %q cannot be installed from a user package", plugin.ID)
	}
	if _, ok := s.findPlugin(plugin.ID); ok {
		return InstallPluginResult{State: s.PluginManagerState()}, fmt.Errorf("plugin id conflict: %q already exists", plugin.ID)
	}

	targetDir := filepath.Join(s.config.UserPluginRoot, plugin.ID)
	if _, err := os.Stat(targetDir); err == nil {
		return InstallPluginResult{State: s.PluginManagerState()}, fmt.Errorf("plugin id conflict: %q already exists", plugin.ID)
	} else if !os.IsNotExist(err) {
		return InstallPluginResult{State: s.PluginManagerState()}, err
	}
	if err := copyDir(sourceDir, targetDir); err != nil {
		return InstallPluginResult{State: s.PluginManagerState()}, fmt.Errorf("copy plugin: %w", err)
	}
	if err := s.RefreshPlugins(); err != nil {
		return InstallPluginResult{State: s.PluginManagerState()}, err
	}
	return InstallPluginResult{
		PluginID:      plugin.ID,
		InstalledPath: targetDir,
		State:         s.PluginManagerState(),
	}, nil
}

func (s *LauncherService) UninstallPlugin(id string) (PluginManagerState, error) {
	if id == pluginManagerID {
		return s.PluginManagerState(), errors.New("plugin manager cannot be uninstalled")
	}
	plugin, ok := s.findPlugin(id)
	if !ok {
		return s.PluginManagerState(), fmt.Errorf("plugin %q is not loaded", id)
	}
	if plugin.Source != "user" {
		return s.PluginManagerState(), fmt.Errorf("plugin %q is not a user plugin", id)
	}
	if !pathInside(s.config.UserPluginRoot, plugin.Dir) {
		return s.PluginManagerState(), fmt.Errorf("plugin %q is outside the user plugin root", id)
	}
	if err := os.RemoveAll(plugin.Dir); err != nil {
		return s.PluginManagerState(), err
	}

	s.pluginConfig.Disabled = removeString(s.pluginConfig.Disabled, id)
	if err := writePluginConfig(s.config.PluginConfigPath, s.pluginConfig); err != nil {
		return s.PluginManagerState(), err
	}
	if err := s.RefreshPlugins(); err != nil {
		return s.PluginManagerState(), err
	}
	return s.PluginManagerState(), nil
}

func (s *LauncherService) findPlugin(id string) (plugins.Plugin, bool) {
	for _, plugin := range s.pluginRegistry.Plugins {
		if plugin.ID == id {
			return plugin, true
		}
	}
	return plugins.Plugin{}, false
}

func managedPlugin(plugin plugins.Plugin) ManagedPlugin {
	protected := plugin.ID == pluginManagerID
	return ManagedPlugin{
		ID:            plugin.ID,
		Name:          plugin.Name,
		Type:          plugin.Type,
		Source:        plugin.Source,
		Dir:           plugin.Dir,
		EntryPath:     plugin.EntryPath,
		Enabled:       !plugin.Disabled,
		Protected:     protected,
		Uninstallable: plugin.Source == "user" && !protected,
	}
}

func readPluginConfig(path string) (PluginConfig, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return PluginConfig{}, nil
		}
		return PluginConfig{}, err
	}
	var config PluginConfig
	if err := json.Unmarshal(raw, &config); err != nil {
		return PluginConfig{}, err
	}
	config.Disabled = uniqueStrings(config.Disabled)
	return config, nil
}

func writePluginConfig(path string, config PluginConfig) error {
	config.Disabled = uniqueStrings(config.Disabled)
	raw, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, append(raw, '\n'), 0o644)
}

func copyDir(sourceDir string, targetDir string) error {
	info, err := os.Stat(sourceDir)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", sourceDir)
	}
	return filepath.WalkDir(sourceDir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		target := filepath.Join(targetDir, rel)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		return copyFile(path, target, info.Mode().Perm())
	})
}

func copyFile(source string, target string, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}

func pathInside(root string, path string) bool {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return false
	}
	pathAbs, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(rootAbs, pathAbs)
	if err != nil {
		return false
	}
	return rel != "." && rel != ".." && !filepath.IsAbs(rel) && len(rel) > 0 && rel != ".." && !startsWithDotDot(rel)
}

func startsWithDotDot(path string) bool {
	return path == ".." || len(path) > 3 && path[:3] == ".."+string(filepath.Separator)
}

func containsString(items []string, needle string) bool {
	for _, item := range items {
		if item == needle {
			return true
		}
	}
	return false
}

func removeString(items []string, needle string) []string {
	var out []string
	for _, item := range items {
		if item != needle {
			out = append(out, item)
		}
	}
	return out
}

func uniqueStrings(items []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, item := range items {
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	sort.Strings(out)
	return out
}
