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
