package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPluginManagerSearchResultOpensManager(t *testing.T) {
	root := t.TempDir()
	builtin := filepath.Join(root, "builtin")
	user := filepath.Join(root, "user")
	configPath := filepath.Join(root, "config", "plugins.json")

	service := NewLauncherService(Config{
		PluginRoots:      []string{builtin, user},
		UserPluginRoot:   user,
		PluginConfigPath: configPath,
	})
	if err := service.RefreshPlugins(); err != nil {
		t.Fatal(err)
	}

	results := service.Search("plugin manager")
	if len(results) != 1 {
		t.Fatalf("len(results) = %d", len(results))
	}
	if results[0].Action.PluginID != "ra-plugin-manager" {
		t.Fatalf("PluginID = %q", results[0].Action.PluginID)
	}
	if results[0].Action.Type != "plugin.manage" {
		t.Fatalf("Action.Type = %q", results[0].Action.Type)
	}
}

func TestSetPluginEnabledWritesDisabledConfigAndKeepsManagerEnabled(t *testing.T) {
	root := t.TempDir()
	builtin := filepath.Join(root, "builtin")
	user := filepath.Join(root, "user")
	configPath := filepath.Join(root, "config", "plugins.json")
	writeManagedWebPlugin(t, builtin, "builtin-web", "Builtin Web")

	service := NewLauncherService(Config{
		PluginRoots:      []string{builtin, user},
		UserPluginRoot:   user,
		PluginConfigPath: configPath,
	})
	if err := service.RefreshPlugins(); err != nil {
		t.Fatal(err)
	}

	if _, err := service.SetPluginEnabled("builtin-web", false); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"builtin-web"`) {
		t.Fatalf("config = %s", raw)
	}

	results := service.Search("builtin")
	if len(results) != 0 {
		t.Fatalf("disabled plugin search results = %#v", results)
	}
	state := service.PluginManagerState()
	if got := findManagedPlugin(t, state, "builtin-web"); got.Enabled {
		t.Fatalf("builtin-web enabled = true")
	}

	if _, err := service.SetPluginEnabled("ra-plugin-manager", false); err == nil {
		t.Fatal("expected disabling plugin manager to fail")
	}
	if got := findManagedPlugin(t, service.PluginManagerState(), "ra-plugin-manager"); !got.Enabled {
		t.Fatal("plugin manager should remain enabled")
	}
}

func TestInstallPluginCopiesLocalDirectoryAndRejectsIDConflicts(t *testing.T) {
	root := t.TempDir()
	builtin := filepath.Join(root, "builtin")
	user := filepath.Join(root, "user")
	configPath := filepath.Join(root, "config", "plugins.json")
	source := filepath.Join(root, "source", "user-web")
	conflict := filepath.Join(root, "conflict", "builtin-web")
	writeManagedWebPlugin(t, builtin, "builtin-web", "Builtin Web")
	writeManagedWebPlugin(t, filepath.Dir(source), "user-web", "User Web")
	writeManagedWebPlugin(t, filepath.Dir(conflict), "builtin-web", "Conflict")

	service := NewLauncherService(Config{
		PluginRoots:      []string{builtin, user},
		UserPluginRoot:   user,
		PluginConfigPath: configPath,
	})
	if err := service.RefreshPlugins(); err != nil {
		t.Fatal(err)
	}

	result, err := service.InstallPlugin(source)
	if err != nil {
		t.Fatal(err)
	}
	if result.PluginID != "user-web" {
		t.Fatalf("PluginID = %q", result.PluginID)
	}
	if _, err := os.Stat(filepath.Join(user, "user-web", "manifest.json")); err != nil {
		t.Fatal(err)
	}
	if got := findManagedPlugin(t, service.PluginManagerState(), "user-web"); got.Source != "user" {
		t.Fatalf("source = %q", got.Source)
	}

	if _, err := service.InstallPlugin(conflict); err == nil {
		t.Fatal("expected ID conflict")
	}
}

func TestInstallPluginRejectsExternalManagerType(t *testing.T) {
	root := t.TempDir()
	builtin := filepath.Join(root, "builtin")
	user := filepath.Join(root, "user")
	configPath := filepath.Join(root, "config", "plugins.json")
	source := filepath.Join(root, "source", "fake-manager")
	writeBrokenManagedPlugin(t, filepath.Dir(source), "fake-manager", `{
  "id":"fake-manager",
  "name":"Fake Manager",
  "type":"manager",
  "entry":"index.html",
  "commands":[{"id":"open","title":"Open Fake Manager"}]
}`)
	if err := os.WriteFile(filepath.Join(source, "index.html"), []byte("<main></main>"), 0o644); err != nil {
		t.Fatal(err)
	}

	service := NewLauncherService(Config{
		PluginRoots:      []string{builtin, user},
		UserPluginRoot:   user,
		PluginConfigPath: configPath,
	})
	if err := service.RefreshPlugins(); err != nil {
		t.Fatal(err)
	}

	if _, err := service.InstallPlugin(source); err == nil {
		t.Fatal("expected manager type install to fail")
	}
}

func TestInstallPluginRejectsBuiltinAppLauncherID(t *testing.T) {
	root := t.TempDir()
	user := filepath.Join(root, "user")
	source := filepath.Join(root, "source", "ra-app-launcher")
	writeManagedWebPlugin(t, filepath.Dir(source), "ra-app-launcher", "Fake App Launcher")

	service := NewLauncherService(Config{UserPluginRoot: user})

	if _, err := service.InstallPlugin(source); err == nil {
		t.Fatal("expected built-in app launcher ID install to fail")
	}
	if _, err := os.Stat(filepath.Join(user, "ra-app-launcher")); !os.IsNotExist(err) {
		t.Fatalf("reserved plugin was copied: %v", err)
	}
}

func TestUserPluginSourceUsesConfiguredUserRoot(t *testing.T) {
	root := t.TempDir()
	builtin := filepath.Join(root, "builtin")
	user := filepath.Join(root, "user")
	extra := filepath.Join(root, "extra")
	configPath := filepath.Join(root, "config", "plugins.json")
	writeManagedWebPlugin(t, extra, "extra-web", "Extra Web")
	writeManagedWebPlugin(t, user, "user-web", "User Web")

	service := NewLauncherService(Config{
		PluginRoots:      []string{builtin, extra, user},
		UserPluginRoot:   user,
		PluginConfigPath: configPath,
	})
	if err := service.RefreshPlugins(); err != nil {
		t.Fatal(err)
	}

	if got := findManagedPlugin(t, service.PluginManagerState(), "extra-web"); got.Source != "builtin" {
		t.Fatalf("extra source = %q", got.Source)
	}
	if got := findManagedPlugin(t, service.PluginManagerState(), "user-web"); got.Source != "user" {
		t.Fatalf("user source = %q", got.Source)
	}
}

func TestUninstallPluginOnlyAllowsUserPlugins(t *testing.T) {
	root := t.TempDir()
	builtin := filepath.Join(root, "builtin")
	user := filepath.Join(root, "user")
	configPath := filepath.Join(root, "config", "plugins.json")
	writeManagedWebPlugin(t, builtin, "builtin-web", "Builtin Web")
	writeManagedWebPlugin(t, user, "user-web", "User Web")

	service := NewLauncherService(Config{
		PluginRoots:      []string{builtin, user},
		UserPluginRoot:   user,
		PluginConfigPath: configPath,
	})
	if err := service.RefreshPlugins(); err != nil {
		t.Fatal(err)
	}

	if _, err := service.UninstallPlugin("builtin-web"); err == nil {
		t.Fatal("expected builtin uninstall to fail")
	}
	if _, err := service.UninstallPlugin("ra-plugin-manager"); err == nil {
		t.Fatal("expected plugin manager uninstall to fail")
	}

	if _, err := service.SetPluginEnabled("user-web", false); err != nil {
		t.Fatal(err)
	}
	if _, err := service.UninstallPlugin("user-web"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(user, "user-web")); !os.IsNotExist(err) {
		t.Fatalf("user plugin still exists: %v", err)
	}
	if got := service.PluginManagerState(); hasManagedPlugin(got, "user-web") {
		t.Fatalf("uninstalled plugin still listed: %#v", got.Plugins)
	}
	raw, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "user-web") {
		t.Fatalf("disabled config still references user-web: %s", raw)
	}
}

func TestPluginManagerStateIncludesLoadErrorsAndIDConflicts(t *testing.T) {
	root := t.TempDir()
	builtin := filepath.Join(root, "builtin")
	user := filepath.Join(root, "user")
	configPath := filepath.Join(root, "config", "plugins.json")
	writeManagedWebPlugin(t, builtin, "shared-web", "Shared Web")
	writeManagedWebPlugin(t, user, "shared-web", "Conflicting Web")
	writeBrokenManagedPlugin(t, user, "missing-entry", `{"id":"missing-entry","name":"Missing","type":"webview","entry":"index.html"}`)

	service := NewLauncherService(Config{
		PluginRoots:      []string{builtin, user},
		UserPluginRoot:   user,
		PluginConfigPath: configPath,
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
	if !hasLoadError(state, "entry") {
		t.Fatalf("missing entry error: %#v", state.LoadErrors)
	}
}

func TestExternalPluginCannotUsePluginManagerID(t *testing.T) {
	root := t.TempDir()
	builtin := filepath.Join(root, "builtin")
	user := filepath.Join(root, "user")
	configPath := filepath.Join(root, "config", "plugins.json")
	writeManagedWebPlugin(t, user, "ra-plugin-manager", "Fake Manager")

	service := NewLauncherService(Config{
		PluginRoots:      []string{builtin, user},
		UserPluginRoot:   user,
		PluginConfigPath: configPath,
	})
	if err := service.RefreshPlugins(); err != nil {
		t.Fatal(err)
	}

	state := service.PluginManagerState()
	if len(state.Plugins) != 2 {
		t.Fatalf("len(Plugins) = %d: %#v", len(state.Plugins), state.Plugins)
	}
	manager := findManagedPlugin(t, state, "ra-plugin-manager")
	if manager.Type != "manager" {
		t.Fatalf("manager plugin = %#v", manager)
	}
	if !hasLoadError(state, "reserved plugin id") {
		t.Fatalf("missing reserved id error: %#v", state.LoadErrors)
	}
}

func TestExternalPluginCannotUseAppLauncherID(t *testing.T) {
	root := t.TempDir()
	builtin := filepath.Join(root, "builtin")
	user := filepath.Join(root, "user")
	configPath := filepath.Join(root, "config", "plugins.json")
	writeManagedWebPlugin(t, user, "ra-app-launcher", "Fake App Launcher")

	service := NewLauncherService(Config{
		PluginRoots:      []string{builtin, user},
		UserPluginRoot:   user,
		PluginConfigPath: configPath,
	})
	if err := service.RefreshPlugins(); err != nil {
		t.Fatal(err)
	}

	state := service.PluginManagerState()
	appLauncher := findManagedPlugin(t, state, "ra-app-launcher")
	if appLauncher.Type != "app" || appLauncher.Source != "builtin" {
		t.Fatalf("app launcher plugin = %#v", appLauncher)
	}
	if appLauncher.Protected {
		t.Fatalf("ra-app-launcher protected = true")
	}
	if !hasLoadError(state, "reserved plugin id") {
		t.Fatalf("missing reserved id error: %#v", state.LoadErrors)
	}
}

func writeManagedWebPlugin(t *testing.T, root string, id string, name string) {
	t.Helper()
	dir := filepath.Join(root, id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `{"id":"` + id + `","name":"` + name + `","type":"webview","entry":"index.html","commands":[{"id":"open","title":"Open ` + name + `"}]}`
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("<main></main>"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeBrokenManagedPlugin(t *testing.T, root string, id string, manifest string) {
	t.Helper()
	dir := filepath.Join(root, id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
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
