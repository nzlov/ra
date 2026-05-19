package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nzlov/ra/internal/pluginbundle"
)

func TestPluginManagerSearchResultOpensManagerCapability(t *testing.T) {
	root := t.TempDir()
	user := filepath.Join(root, "user")
	configPath := filepath.Join(root, "config", "plugins.json")

	service := NewLauncherService(Config{
		PluginRoots:      []string{user},
		UserPluginRoot:   user,
		PluginConfigPath: configPath,
		BuiltinPlugins:   builtinTestPlugins(t),
	})
	if err := service.RefreshPlugins(); err != nil {
		t.Fatal(err)
	}

	results := service.Search("plugin manager")
	if len(results) != 1 {
		t.Fatalf("len(results) = %d", len(results))
	}
	if results[0].Action.PluginID != "ra-plugin-manager" || results[0].Action.CapabilityID != "manage" {
		t.Fatalf("Action = %#v", results[0].Action)
	}
	if results[0].Action.Type != "plugin.manage" {
		t.Fatalf("Action.Type = %q", results[0].Action.Type)
	}
}

func TestRefreshPluginsKeepsManagerCapabilityEnabledFromManualConfig(t *testing.T) {
	root := t.TempDir()
	user := filepath.Join(root, "user")
	configPath := filepath.Join(root, "config", "plugins.json")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, []byte(`{"disabled":["ra-plugin-manager"],"disabledCapabilities":["ra-plugin-manager.manage"]}`), 0o644); err != nil {
		t.Fatal(err)
	}

	service := NewLauncherService(Config{
		PluginRoots:      []string{user},
		UserPluginRoot:   user,
		PluginConfigPath: configPath,
		BuiltinPlugins:   builtinTestPlugins(t),
	})
	if err := service.RefreshPlugins(); err != nil {
		t.Fatal(err)
	}

	manager := findManagedPlugin(t, service.PluginManagerState(), "ra-plugin-manager")
	if !manager.Enabled {
		t.Fatal("plugin manager should ignore manual disabled config")
	}
	if got := findManagedCapability(t, manager, "manage"); !got.Enabled {
		t.Fatal("plugin manager capability should ignore manual disabled config")
	}
	results := service.Search("plugin manager")
	if len(results) != 1 || results[0].Action.Type != "plugin.manage" {
		t.Fatalf("manager search results = %#v", results)
	}
}

func TestSetPluginEnabledWritesDisabledConfigAndKeepsManagerEnabled(t *testing.T) {
	root := t.TempDir()
	user := filepath.Join(root, "user")
	configPath := filepath.Join(root, "config", "plugins.json")
	writeWASMPlugin(t, user, pluginbundle.Manifest{ID: "codec-tools", Name: "Codec Tools", Version: "0.1.0"}, []pluginbundle.Capability{{
		ID:       "base64",
		Title:    "Base64 Convert",
		UI:       "/base64/index.html",
		Keywords: []string{"base64"},
	}})

	service := NewLauncherService(Config{
		PluginRoots:      []string{user},
		UserPluginRoot:   user,
		PluginConfigPath: configPath,
		BuiltinPlugins:   builtinTestPlugins(t),
	})
	if err := service.RefreshPlugins(); err != nil {
		t.Fatal(err)
	}

	if _, err := service.SetPluginEnabled("codec-tools", false); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"codec-tools"`) {
		t.Fatalf("config = %s", raw)
	}

	results := service.Search("base64")
	if len(results) != 0 {
		t.Fatalf("disabled plugin search results = %#v", results)
	}
	state := service.PluginManagerState()
	if got := findManagedPlugin(t, state, "codec-tools"); got.Enabled {
		t.Fatalf("codec-tools enabled = true")
	}

	if _, err := service.SetPluginEnabled("ra-plugin-manager", false); err == nil {
		t.Fatal("expected disabling plugin manager to fail")
	}
	if got := findManagedPlugin(t, service.PluginManagerState(), "ra-plugin-manager"); !got.Enabled {
		t.Fatal("plugin manager should remain enabled")
	}

	if _, err := service.SetCapabilityEnabled("ra-plugin-manager", "manage", false); err == nil {
		t.Fatal("expected disabling plugin manager capability to fail")
	}
	if got := findManagedCapability(t, findManagedPlugin(t, service.PluginManagerState(), "ra-plugin-manager"), "manage"); !got.Enabled {
		t.Fatal("plugin manager capability should remain enabled")
	}
}

func TestInstallPluginCopiesWASMFileAndRejectsIDConflicts(t *testing.T) {
	root := t.TempDir()
	user := filepath.Join(root, "user")
	configPath := filepath.Join(root, "config", "plugins.json")
	source := filepath.Join(root, "source", "codec-tools.wasm")
	conflict := filepath.Join(root, "source", "ra-app-launcher.wasm")
	writeWASMFile(t, source, pluginbundle.Manifest{ID: "codec-tools", Name: "Codec Tools", Version: "0.1.0"}, nil)
	writeWASMFile(t, conflict, pluginbundle.Manifest{ID: "ra-app-launcher", Name: "Fake Apps", Version: "0.1.0"}, nil)

	service := NewLauncherService(Config{
		PluginRoots:      []string{user},
		UserPluginRoot:   user,
		PluginConfigPath: configPath,
		BuiltinPlugins:   builtinTestPlugins(t),
	})
	if err := service.RefreshPlugins(); err != nil {
		t.Fatal(err)
	}

	result, err := service.InstallPlugin(source)
	if err != nil {
		t.Fatal(err)
	}
	if result.PluginID != "codec-tools" {
		t.Fatalf("PluginID = %q", result.PluginID)
	}
	if _, err := os.Stat(filepath.Join(user, "codec-tools.wasm")); err != nil {
		t.Fatal(err)
	}
	if got := findManagedPlugin(t, service.PluginManagerState(), "codec-tools"); got.Source != "user" {
		t.Fatalf("source = %q", got.Source)
	}

	if _, err := service.InstallPlugin(conflict); err == nil {
		t.Fatal("expected ID conflict")
	}
}

func TestInstallPluginRejectsDirectoriesAndInvalidWASM(t *testing.T) {
	root := t.TempDir()
	user := filepath.Join(root, "user")
	sourceDir := filepath.Join(root, "source-dir")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	invalid := filepath.Join(root, "bad.wasm")
	if err := os.WriteFile(invalid, []byte("bad"), 0o644); err != nil {
		t.Fatal(err)
	}
	service := NewLauncherService(Config{UserPluginRoot: user, BuiltinPlugins: builtinTestPlugins(t)})

	if _, err := service.InstallPlugin(sourceDir); err == nil {
		t.Fatal("expected directory install to fail")
	}
	if _, err := service.InstallPlugin(invalid); err == nil {
		t.Fatal("expected invalid wasm install to fail")
	}
}

func TestUninstallPluginOnlyAllowsUserWASMPlugins(t *testing.T) {
	root := t.TempDir()
	user := filepath.Join(root, "user")
	configPath := filepath.Join(root, "config", "plugins.json")
	writeWASMPlugin(t, user, pluginbundle.Manifest{ID: "codec-tools", Name: "Codec Tools", Version: "0.1.0"}, nil)

	service := NewLauncherService(Config{
		PluginRoots:      []string{user},
		UserPluginRoot:   user,
		PluginConfigPath: configPath,
		BuiltinPlugins:   builtinTestPlugins(t),
	})
	if err := service.RefreshPlugins(); err != nil {
		t.Fatal(err)
	}

	if _, err := service.UninstallPlugin("ra-app-launcher"); err == nil {
		t.Fatal("expected builtin uninstall to fail")
	}
	if _, err := service.UninstallPlugin("ra-plugin-manager"); err == nil {
		t.Fatal("expected plugin manager uninstall to fail")
	}

	if _, err := service.SetPluginEnabled("codec-tools", false); err != nil {
		t.Fatal(err)
	}
	if _, err := service.UninstallPlugin("codec-tools"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(user, "codec-tools.wasm")); !os.IsNotExist(err) {
		t.Fatalf("user plugin still exists: %v", err)
	}
	if got := service.PluginManagerState(); hasManagedPlugin(got, "codec-tools") {
		t.Fatalf("uninstalled plugin still listed: %#v", got.Plugins)
	}
	raw, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "codec-tools") {
		t.Fatalf("disabled config still references codec-tools: %s", raw)
	}
}

func TestPluginManagerStateIncludesLoadErrorsAndIDConflicts(t *testing.T) {
	root := t.TempDir()
	user := filepath.Join(root, "user")
	configPath := filepath.Join(root, "config", "plugins.json")
	writeWASMPlugin(t, user, pluginbundle.Manifest{ID: "ra-app-launcher", Name: "Fake Apps", Version: "0.1.0"}, nil)
	if err := os.WriteFile(filepath.Join(user, "bad.wasm"), []byte("bad"), 0o644); err != nil {
		t.Fatal(err)
	}

	service := NewLauncherService(Config{
		PluginRoots:      []string{user},
		UserPluginRoot:   user,
		PluginConfigPath: configPath,
		BuiltinPlugins:   builtinTestPlugins(t),
	})
	if err := service.RefreshPlugins(); err != nil {
		t.Fatal(err)
	}

	state := service.PluginManagerState()
	if len(state.LoadErrors) != 2 {
		t.Fatalf("len(LoadErrors) = %d: %#v", len(state.LoadErrors), state.LoadErrors)
	}
	if !hasLoadError(state, "id conflict") {
		t.Fatalf("missing conflict error: %#v", state.LoadErrors)
	}
	if !hasLoadError(state, "invalid wasm") {
		t.Fatalf("missing wasm error: %#v", state.LoadErrors)
	}
}

func findManagedPlugin(t *testing.T, state PluginManagerState, id string) ManagedPlugin {
	t.Helper()
	for _, plugin := range state.Plugins {
		if plugin.ID == id {
			return plugin
		}
	}
	t.Fatalf("missing plugin %q in %#v", id, state.Plugins)
	return ManagedPlugin{}
}

func findManagedCapability(t *testing.T, plugin ManagedPlugin, id string) ManagedCapability {
	t.Helper()
	for _, capability := range plugin.Capabilities {
		if capability.ID == id {
			return capability
		}
	}
	t.Fatalf("missing capability %q in %#v", id, plugin.Capabilities)
	return ManagedCapability{}
}

func hasManagedPlugin(state PluginManagerState, id string) bool {
	for _, plugin := range state.Plugins {
		if plugin.ID == id {
			return true
		}
	}
	return false
}

func hasLoadError(state PluginManagerState, needle string) bool {
	for _, loadError := range state.LoadErrors {
		if strings.Contains(loadError.Error, needle) {
			return true
		}
	}
	return false
}

func writeWASMFile(t *testing.T, path string, manifest pluginbundle.Manifest, capabilities []pluginbundle.Capability) {
	t.Helper()
	raw := mustBundle(t, manifest, capabilities)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
}
