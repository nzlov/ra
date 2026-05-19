package plugins

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/nzlov/ra/pkg/raplugin"
)

func TestLoadRegistryReadsWASMPlugins(t *testing.T) {
	root := t.TempDir()
	writeTestPlugin(t, root, "codec-tools", "./internal/plugins/testdata/codecplugin")

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
	writeTestPlugin(t, root, "first", "./internal/plugins/testdata/sharedone")
	writeTestPlugin(t, root, "second", "./internal/plugins/testdata/sharedtwo")
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
		Name: "ra-app-launcher",
		Raw:  buildTestPlugin(t, "./internal/plugins/testdata/appplugin"),
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

func TestSearchCallsPluginSearch(t *testing.T) {
	raw := buildTestPlugin(t, "./internal/plugins/testdata/appplugin")
	plugin, err := loadWASMBytes(raw, "builtin", "ra-app-launcher")
	if err != nil {
		t.Fatal(err)
	}
	registry := Registry{Plugins: []Plugin{plugin}}

	results := registry.SearchWithContext(SearchRequest{
		Query: "fire",
		Limit: 10,
		Apps: []raplugin.App{{
			ID:      "firefox",
			Name:    "Firefox",
			Comment: "Browser",
			Command: "firefox",
		}},
	})
	if len(results) != 1 {
		t.Fatalf("len(results) = %d", len(results))
	}
	if results[0].Action.Type != "app.launch" {
		t.Fatalf("Action.Type = %q", results[0].Action.Type)
	}
	if results[0].Action.PluginID != "ra-app-launcher" || results[0].Action.CapabilityID != "apps" {
		t.Fatalf("Action = %#v", results[0].Action)
	}
}

func writeTestPlugin(t *testing.T, root string, id string, pkg string) {
	t.Helper()
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(root, id+".wasm")
	buildTestPluginTo(t, output, pkg)
}

func buildTestPlugin(t *testing.T, pkg string) []byte {
	t.Helper()
	output := filepath.Join(t.TempDir(), "plugin.wasm")
	buildTestPluginTo(t, output, pkg)
	raw, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func buildTestPluginTo(t *testing.T, output string, pkg string) {
	t.Helper()
	cmd := exec.Command("go", "build", "-buildvcs=false", "-buildmode=c-shared", "-o", output, pkg)
	cmd.Dir = repoRootForTest(t)
	cmd.Env = append(os.Environ(),
		"GOOS=wasip1",
		"GOARCH=wasm",
		"GOCACHE="+filepath.Join(t.TempDir(), "gocache"),
	)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("build test plugin: %v: %s", err, stderr.String())
	}
}

func repoRootForTest(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test file")
	}
	return filepath.Dir(filepath.Dir(filepath.Dir(file)))
}
