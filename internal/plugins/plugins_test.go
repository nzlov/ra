package plugins

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nzlov/ra/internal/pluginbundle"
)

func TestLoadRegistryReadsWASMPlugins(t *testing.T) {
	root := t.TempDir()
	writeBundle(t, filepath.Join(root, "codec-tools.wasm"), pluginbundle.Manifest{
		ID:          "codec-tools",
		Name:        "Codec Tools",
		Version:     "0.1.0",
		Permissions: []string{"clipboard:write"},
	}, []pluginbundle.Capability{{
		ID:       "base64",
		Title:    "Base64 Convert",
		UI:       "/base64/index.html",
		Keywords: []string{"base64", "b64"},
	}})

	registry, err := LoadRegistry(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(registry.Plugins) != 1 {
		t.Fatalf("len(Plugins) = %d", len(registry.Plugins))
	}
	plugin := registry.Plugins[0]
	if plugin.ID != "codec-tools" {
		t.Fatalf("ID = %q", plugin.ID)
	}
	if plugin.Path != filepath.Join(root, "codec-tools.wasm") {
		t.Fatalf("Path = %q", plugin.Path)
	}
	if len(plugin.Capabilities) != 1 || plugin.Capabilities[0].ID != "base64" {
		t.Fatalf("Capabilities = %#v", plugin.Capabilities)
	}
	if plugin.Permissions[0] != "clipboard:write" {
		t.Fatalf("Permissions = %#v", plugin.Permissions)
	}
}

func TestLoadRegistryRejectsInvalidWASMAndIDConflicts(t *testing.T) {
	root := t.TempDir()
	writeBundle(t, filepath.Join(root, "first.wasm"), pluginbundle.Manifest{ID: "shared", Name: "Shared", Version: "0.1.0"}, nil)
	writeBundle(t, filepath.Join(root, "second.wasm"), pluginbundle.Manifest{ID: "shared", Name: "Shared Again", Version: "0.1.0"}, nil)
	if err := os.WriteFile(filepath.Join(root, "bad.wasm"), []byte("bad"), 0o644); err != nil {
		t.Fatal(err)
	}

	registry, err := LoadRegistry(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(registry.Plugins) != 1 {
		t.Fatalf("len(Plugins) = %d", len(registry.Plugins))
	}
	if len(registry.Errors) != 2 {
		t.Fatalf("len(Errors) = %d: %#v", len(registry.Errors), registry.Errors)
	}
}

func TestLoadRegistriesIncludesBuiltinPlugins(t *testing.T) {
	userRoot := t.TempDir()
	builtin := BuiltinPlugin{
		Raw: mustBundle(t, pluginbundle.Manifest{ID: "ra-app-launcher", Name: "RA App Launcher", Version: "0.1.0"}, []pluginbundle.Capability{{
			ID:       "apps",
			Title:    "Apps",
			UI:       "/apps/index.html",
			Keywords: []string{"app"},
		}}),
	}

	registry, err := LoadRegistriesWithSources([]Root{{Path: userRoot, Source: "user"}}, []BuiltinPlugin{builtin})
	if err != nil {
		t.Fatal(err)
	}
	if len(registry.Plugins) != 1 {
		t.Fatalf("len(Plugins) = %d", len(registry.Plugins))
	}
	if registry.Plugins[0].Source != "builtin" {
		t.Fatalf("Source = %q", registry.Plugins[0].Source)
	}
}

func TestSearchReturnsCapabilityResults(t *testing.T) {
	registry := Registry{Plugins: []Plugin{{
		ID:   "codec-tools",
		Name: "Codec Tools",
		Capabilities: []Capability{{
			ID:       "base64",
			Title:    "Base64 Convert",
			UI:       "/base64/index.html",
			Keywords: []string{"base64", "b64"},
		}},
	}}}

	results := registry.Search("b64 hello", 10)
	if len(results) != 1 {
		t.Fatalf("len(results) = %d", len(results))
	}
	if results[0].Action.Type != "capability.open" {
		t.Fatalf("Action.Type = %q", results[0].Action.Type)
	}
	if results[0].Action.PluginID != "codec-tools" || results[0].Action.CapabilityID != "base64" {
		t.Fatalf("Action = %#v", results[0].Action)
	}
	if results[0].Action.Query != "b64 hello" {
		t.Fatalf("Query = %q", results[0].Action.Query)
	}
}

func writeBundle(t *testing.T, path string, manifest pluginbundle.Manifest, capabilities []pluginbundle.Capability) {
	t.Helper()
	raw := mustBundle(t, manifest, capabilities)
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
}

func mustBundle(t *testing.T, manifest pluginbundle.Manifest, capabilities []pluginbundle.Capability) []byte {
	t.Helper()
	raw, err := pluginbundle.Build(manifest, capabilities, bundleAssets(capabilities))
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func bundleAssets(capabilities []pluginbundle.Capability) map[string][]byte {
	assets := map[string][]byte{
		"/index.html": []byte("<main></main>"),
	}
	for _, capability := range capabilities {
		assets[capability.UI] = []byte("<main>" + capability.ID + "</main>")
		if capability.Icon != "" {
			assets[capability.Icon] = []byte("<svg></svg>")
		}
	}
	return assets
}
