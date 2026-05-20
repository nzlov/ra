package plugins

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nzlov/ra/internal/pluginruntime"
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
	writeTestPlugin(t, root, "invalid-id", "./internal/plugins/testdata/invalidid")
	writeTestPlugin(t, root, "missing-ui", "./internal/plugins/testdata/missinguibundle")
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
	if len(registry.Errors) != 4 {
		t.Fatalf("len(Errors) = %d: %#v", len(registry.Errors), registry.Errors)
	}
	if !hasLoadError(registry.Errors, "invalid plugin id") {
		t.Fatalf("missing invalid id error: %#v", registry.Errors)
	}
	if !hasLoadError(registry.Errors, "missing capability UI asset") {
		t.Fatalf("missing UI asset error: %#v", registry.Errors)
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
		HostAPI: HostAPI{Apps: []raplugin.App{{
			ID:      "firefox",
			Name:    "Firefox",
			Comment: "Browser",
		}}},
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

func TestSearchWithContextRunsPluginSearchConcurrentlyWithRuntimeLimit(t *testing.T) {
	previousGOMAXPROCS := runtime.GOMAXPROCS(2)
	t.Cleanup(func() {
		runtime.GOMAXPROCS(previousGOMAXPROCS)
	})
	previousSearch := runPluginSearch
	t.Cleanup(func() {
		runPluginSearch = previousSearch
	})

	var active int32
	var maxActive int32
	started := make(chan string, 4)
	release := make(chan struct{})
	var releaseOnce sync.Once
	t.Cleanup(func() {
		releaseOnce.Do(func() { close(release) })
	})

	runPluginSearch = func(raw []byte, request raplugin.SearchRequest, api ...pluginruntime.HostAPI) ([]raplugin.SearchResult, error) {
		id := string(raw)
		current := atomic.AddInt32(&active, 1)
		for {
			seen := atomic.LoadInt32(&maxActive)
			if current <= seen || atomic.CompareAndSwapInt32(&maxActive, seen, current) {
				break
			}
		}
		started <- id
		<-release
		atomic.AddInt32(&active, -1)
		return []raplugin.SearchResult{{
			ID:    "result:" + id,
			Title: id,
			Action: raplugin.Action{
				CapabilityID: "main",
			},
		}}, nil
	}

	registry := Registry{Plugins: []Plugin{
		searchTestPlugin("plugin-0"),
		searchTestPlugin("plugin-1"),
		searchTestPlugin("plugin-2"),
		searchTestPlugin("plugin-3"),
	}}
	done := make(chan []Result, 1)
	go func() {
		done <- registry.SearchWithContext(SearchRequest{Query: "query", Limit: 10})
	}()

	waitStarted(t, started, 2)
	select {
	case id := <-started:
		t.Fatalf("started %s before respecting runtime concurrency limit", id)
	case <-time.After(50 * time.Millisecond):
	}

	releaseOnce.Do(func() { close(release) })
	var results []Result
	select {
	case results = <-done:
	case <-time.After(time.Second):
		t.Fatal("search did not finish")
	}
	if maxActive != 2 {
		t.Fatalf("max concurrent searches = %d, want 2", maxActive)
	}
	wantTitles := []string{"plugin-0", "plugin-1", "plugin-2", "plugin-3"}
	if len(results) != len(wantTitles) {
		t.Fatalf("len(results) = %d, want %d: %#v", len(results), len(wantTitles), results)
	}
	for i, want := range wantTitles {
		if results[i].Title != want {
			t.Fatalf("result %d title = %q, want %q; results = %#v", i, results[i].Title, want, results)
		}
	}
}

func TestSearchStampsPluginOwnedActionMetadata(t *testing.T) {
	plugin := Plugin{
		ID: "trusted-plugin",
		Capabilities: []Capability{{
			ID: "main",
			UI: "/trusted/index.html",
		}},
	}

	result := resultFromPlugin(plugin, raplugin.SearchResult{
		Action: raplugin.Action{
			Type:         "capability.open",
			CapabilityID: "main",
		},
	}, "query")

	if result.Action.PluginID != "trusted-plugin" {
		t.Fatalf("PluginID = %q", result.Action.PluginID)
	}
	if result.Action.UI != "/trusted/index.html" {
		t.Fatalf("UI = %q", result.Action.UI)
	}
}

func searchTestPlugin(id string) Plugin {
	return Plugin{
		ID:  id,
		Raw: []byte(id),
		Capabilities: []Capability{{
			ID: "main",
			UI: "/main.html",
		}},
	}
}

func waitStarted(t *testing.T, started <-chan string, count int) {
	t.Helper()
	for i := 0; i < count; i++ {
		select {
		case <-started:
		case <-time.After(time.Second):
			t.Fatalf("started %d searches, want %d", i, count)
		}
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

func hasLoadError(errors []LoadError, needle string) bool {
	for _, item := range errors {
		if strings.Contains(item.Error, needle) {
			return true
		}
	}
	return false
}
