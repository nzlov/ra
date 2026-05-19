package plugins

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadRegistryReadsValidWebviewManifest(t *testing.T) {
	root := t.TempDir()
	writePlugin(t, root, "example", `{
  "id": "example-webview",
  "name": "Example Webview",
  "type": "webview",
  "entry": "index.html",
  "commands": [
    {"id": "open", "title": "Open Example", "subtitle": "Show demo page"}
  ]
}`)
	if err := os.WriteFile(filepath.Join(root, "example", "index.html"), []byte("<main></main>"), 0o644); err != nil {
		t.Fatal(err)
	}

	registry, err := LoadRegistry(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(registry.Plugins) != 1 {
		t.Fatalf("len(Plugins) = %d", len(registry.Plugins))
	}
	plugin := registry.Plugins[0]
	if plugin.ID != "example-webview" {
		t.Fatalf("ID = %q", plugin.ID)
	}
	if plugin.EntryPath != filepath.Join(root, "example", "index.html") {
		t.Fatalf("EntryPath = %q", plugin.EntryPath)
	}
}

func TestLoadRegistryRejectsInvalidIDAndMissingEntry(t *testing.T) {
	root := t.TempDir()
	writePlugin(t, root, "bad-id", `{"id":"Bad ID","name":"Bad","type":"webview","entry":"index.html"}`)
	writePlugin(t, root, "missing-entry", `{"id":"missing-entry","name":"Missing","type":"webview","entry":"index.html"}`)

	registry, err := LoadRegistry(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(registry.Plugins) != 0 {
		t.Fatalf("len(Plugins) = %d", len(registry.Plugins))
	}
	if len(registry.Errors) != 2 {
		t.Fatalf("len(Errors) = %d", len(registry.Errors))
	}
}

func TestSearchReturnsCommandResults(t *testing.T) {
	registry := Registry{Plugins: []Plugin{{
		ID:   "example-webview",
		Name: "Example Webview",
		Type: "webview",
		Commands: []Command{{
			ID:       "open",
			Title:    "Open Example",
			Subtitle: "Show demo page",
		}},
	}}}

	results := registry.Search("example", 10)
	if len(results) != 1 {
		t.Fatalf("len(results) = %d", len(results))
	}
	if results[0].Action.Type != "plugin.open" {
		t.Fatalf("Action.Type = %q", results[0].Action.Type)
	}
	if results[0].Action.PluginID != "example-webview" {
		t.Fatalf("PluginID = %q", results[0].Action.PluginID)
	}
}

func TestSearchReturnsRunActionForCommandPlugin(t *testing.T) {
	registry := Registry{Plugins: []Plugin{{
		ID:        "answer-command",
		Name:      "Answer Command",
		Type:      "command",
		EntryPath: "/tmp/answer.wasm",
		Commands: []Command{{
			ID:       "answer",
			Title:    "Calculate Answer",
			Subtitle: "Run WASM export",
			Export:   "answer",
		}},
	}}}

	results := registry.Search("answer", 10)
	if len(results) != 1 {
		t.Fatalf("len(results) = %d", len(results))
	}
	if results[0].Action.Type != "plugin.run" {
		t.Fatalf("Action.Type = %q", results[0].Action.Type)
	}
	if results[0].Action.Export != "answer" {
		t.Fatalf("Export = %q", results[0].Action.Export)
	}
}

func writePlugin(t *testing.T, root, dir, manifest string) {
	t.Helper()
	pluginDir := filepath.Join(root, dir)
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "manifest.json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
}
