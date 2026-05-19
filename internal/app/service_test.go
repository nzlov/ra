package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nzlov/ra/internal/desktop"
)

func TestSearchMergesCalculatorAppsAndPlugins(t *testing.T) {
	root := t.TempDir()
	pluginDir := filepath.Join(root, "plugins", "example")
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "manifest.json"), []byte(`{
  "id": "example-webview",
  "name": "Example Webview",
  "type": "webview",
  "entry": "index.html",
  "commands": [{"id": "open", "title": "Open Example", "subtitle": "Show demo page"}]
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "index.html"), []byte("<main></main>"), 0o644); err != nil {
		t.Fatal(err)
	}

	service := NewLauncherService(Config{PluginRoot: filepath.Join(root, "plugins")})
	service.SetDesktopEntries([]desktop.Entry{{ID: "firefox", Name: "Firefox", Exec: "firefox %U"}})
	if err := service.RefreshPlugins(); err != nil {
		t.Fatal(err)
	}

	calc := service.Search("=6*7")
	if len(calc) != 1 || calc[0].Kind != "calculator" || calc[0].Title != "42" {
		t.Fatalf("calculator results = %#v", calc)
	}

	apps := service.Search("fire")
	if len(apps) != 1 || apps[0].Kind != "app" || apps[0].Title != "Firefox" {
		t.Fatalf("app results = %#v", apps)
	}
	if apps[0].Action.PluginID != "ra-app-launcher" {
		t.Fatalf("app PluginID = %q", apps[0].Action.PluginID)
	}

	pluginResults := service.Search("example")
	if len(pluginResults) != 1 || pluginResults[0].Kind != "plugin" {
		t.Fatalf("plugin results = %#v", pluginResults)
	}
}

func TestInvokeReturnsPluginOpenPayload(t *testing.T) {
	service := NewLauncherService(Config{})
	result, err := service.Invoke(Action{Type: "plugin.open", PluginID: "example", EntryPath: "/tmp/example/index.html"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Type != "plugin.open" {
		t.Fatalf("Type = %q", result.Type)
	}
	if result.EntryPath != "/tmp/example/index.html" {
		t.Fatalf("EntryPath = %q", result.EntryPath)
	}
}

func TestRefreshPluginsLoadsMultiplePluginRootsAndStatus(t *testing.T) {
	root := t.TempDir()
	builtin := filepath.Join(root, "builtin")
	user := filepath.Join(root, "user")
	writeWebPlugin(t, builtin, "builtin-web", "Builtin Web")
	writeWebPlugin(t, user, "user-web", "User Web")

	service := NewLauncherService(Config{PluginRoots: []string{builtin, user}})
	if err := service.RefreshPlugins(); err != nil {
		t.Fatal(err)
	}
	status := service.Status()
	if status.PluginCount != 4 {
		t.Fatalf("PluginCount = %d", status.PluginCount)
	}
	if status.PluginErrorCount != 0 {
		t.Fatalf("PluginErrorCount = %d", status.PluginErrorCount)
	}
}

func TestDisablingBuiltinAppLauncherRemovesAppSearchResults(t *testing.T) {
	root := t.TempDir()
	builtin := filepath.Join(root, "builtin")
	user := filepath.Join(root, "user")
	configPath := filepath.Join(root, "config", "plugins.json")

	service := NewLauncherService(Config{
		PluginRoots:      []string{builtin, user},
		UserPluginRoot:   user,
		PluginConfigPath: configPath,
	})
	service.SetDesktopEntries([]desktop.Entry{{ID: "firefox", Name: "Firefox", Exec: "firefox %U"}})
	if err := service.RefreshPlugins(); err != nil {
		t.Fatal(err)
	}

	state, err := service.SetPluginEnabled("ra-app-launcher", false)
	if err != nil {
		t.Fatal(err)
	}
	appLauncher := findManagedPlugin(t, state, "ra-app-launcher")
	if appLauncher.Enabled {
		t.Fatalf("ra-app-launcher enabled = true")
	}
	if appLauncher.Protected {
		t.Fatalf("ra-app-launcher protected = true")
	}
	if appLauncher.Uninstallable {
		t.Fatalf("ra-app-launcher uninstallable = true")
	}

	results := service.Search("fire")
	if len(results) != 0 {
		t.Fatalf("disabled app launcher results = %#v", results)
	}
}

func writeWebPlugin(t *testing.T, root string, id string, name string) {
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
