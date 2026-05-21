package plugins

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
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

	results := registry.SearchWithContext(context.Background(), SearchRequest{
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

func TestSearchWithContextSkipsPluginWhenMatchersDoNotMatch(t *testing.T) {
	previousSearch := runPluginSearch
	t.Cleanup(func() {
		runPluginSearch = previousSearch
	})

	var called []string
	runPluginSearch = func(ctx context.Context, plugin Plugin, request raplugin.SearchRequest, api pluginruntime.HostAPI) ([]raplugin.SearchResult, error) {
		called = append(called, plugin.ID)
		return []raplugin.SearchResult{{
			ID:    "result:" + plugin.ID,
			Title: plugin.ID,
			Action: raplugin.Action{
				CapabilityID: "main",
			},
		}}, nil
	}

	registry := Registry{Plugins: []Plugin{
		searchTestPluginWithMatch("regex-plugin", raplugin.Match{Regex: `^\s*=`}),
		searchTestPluginWithMatch("contains-plugin", raplugin.Match{Mode: "contains_all_tokens", Pattern: "plugin manager"}),
		searchTestPlugin("fallback-plugin"),
	}}

	results := registry.SearchWithContext(context.Background(), SearchRequest{Query: "plugin manager", Limit: 10})

	if len(results) != 2 {
		t.Fatalf("len(results) = %d, want 2: %#v", len(results), results)
	}
	if len(called) != 2 {
		t.Fatalf("len(called) = %d, want 2: %#v", len(called), called)
	}
	sort.Strings(called)
	if got, want := strings.Join(called, ","), "contains-plugin,fallback-plugin"; got != want {
		t.Fatalf("called plugins = %q, want %q", got, want)
	}
}

func TestSearchWithContextSkipsPluginWhenMatchingCapabilityDisabled(t *testing.T) {
	previousSearch := runPluginSearch
	t.Cleanup(func() {
		runPluginSearch = previousSearch
	})

	var called bool
	runPluginSearch = func(ctx context.Context, plugin Plugin, request raplugin.SearchRequest, api pluginruntime.HostAPI) ([]raplugin.SearchResult, error) {
		called = true
		return nil, nil
	}

	registry := Registry{Plugins: []Plugin{{
		ID: "plugin-with-disabled-match",
		Capabilities: []Capability{
			testCapabilityWithMatch(t, "disabled-match", raplugin.Match{Mode: "contains_all_tokens", Pattern: "plugin manager"}, true),
			testCapabilityWithMatch(t, "enabled-other", raplugin.Match{Regex: `^\s*=`}, false),
		},
	}}}

	results := registry.SearchWithContext(context.Background(), SearchRequest{Query: "plugin manager", Limit: 10})

	if called {
		t.Fatal("plugin search was called")
	}
	if len(results) != 0 {
		t.Fatalf("results = %#v", results)
	}
}

func TestSearchWithContextBindsStoreHostAPIToCurrentPlugin(t *testing.T) {
	previousSearch := runPluginSearch
	t.Cleanup(func() {
		runPluginSearch = previousSearch
	})

	var calls []string
	runPluginSearch = func(ctx context.Context, plugin Plugin, request raplugin.SearchRequest, api pluginruntime.HostAPI) ([]raplugin.SearchResult, error) {
		if api.StoreGet == nil {
			t.Fatal("StoreGet is nil")
		}
		if api.StoreSet == nil {
			t.Fatal("StoreSet is nil")
		}
		if api.StoreDelete == nil {
			t.Fatal("StoreDelete is nil")
		}
		if api.StoreList == nil {
			t.Fatal("StoreList is nil")
		}
		value, found, err := api.StoreGet("prefs/theme")
		if err != nil {
			t.Fatalf("StoreGet error = %v", err)
		}
		if !found {
			t.Fatal("StoreGet found = false")
		}
		if string(value) != `{"theme":"dark"}` {
			t.Fatalf("StoreGet value = %s", value)
		}
		if err := api.StoreSet("prefs/theme", json.RawMessage(`{"theme":"light"}`)); err != nil {
			t.Fatalf("StoreSet error = %v", err)
		}
		if err := api.StoreDelete("prefs/theme"); err != nil {
			t.Fatalf("StoreDelete error = %v", err)
		}
		list, err := api.StoreList("prefs/")
		if err != nil {
			t.Fatalf("StoreList error = %v", err)
		}
		if string(list) != `[{"key":"prefs/theme"}]` {
			t.Fatalf("StoreList value = %s", list)
		}
		return []raplugin.SearchResult{{
			ID:    "result:" + plugin.ID,
			Title: plugin.ID,
			Action: raplugin.Action{
				CapabilityID: "main",
			},
		}}, nil
	}

	registry := Registry{Plugins: []Plugin{searchTestPlugin("store-plugin")}}
	results := registry.SearchWithContext(context.Background(), SearchRequest{
		Query: "query",
		Limit: 10,
		HostAPI: HostAPI{
			StoreGet: func(pluginID string, key string) (json.RawMessage, bool, error) {
				calls = append(calls, "get:"+pluginID+":"+key)
				return json.RawMessage(`{"theme":"dark"}`), true, nil
			},
			StoreSet: func(pluginID string, key string, value json.RawMessage) error {
				calls = append(calls, "set:"+pluginID+":"+key+":"+string(value))
				return nil
			},
			StoreDelete: func(pluginID string, key string) error {
				calls = append(calls, "delete:"+pluginID+":"+key)
				return nil
			},
			StoreList: func(pluginID string, prefix string) (json.RawMessage, error) {
				calls = append(calls, "list:"+pluginID+":"+prefix)
				return json.RawMessage(`[{"key":"prefs/theme"}]`), nil
			},
		},
	})

	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1: %#v", len(results), results)
	}
	wantCalls := []string{
		"get:store-plugin:prefs/theme",
		"set:store-plugin:prefs/theme:{\"theme\":\"light\"}",
		"delete:store-plugin:prefs/theme",
		"list:store-plugin:prefs/",
	}
	if got, want := strings.Join(calls, ","), strings.Join(wantCalls, ","); got != want {
		t.Fatalf("store calls = %q, want %q", got, want)
	}
}

func TestLoadWASMBytesCopiesCapabilityMatch(t *testing.T) {
	raw := buildTestPlugin(t, "./plugins/ra-calculator")
	plugin, err := loadWASMBytes(raw, "builtin", "ra-calculator")
	if err != nil {
		t.Fatal(err)
	}
	if len(plugin.Capabilities) != 1 {
		t.Fatalf("len(Capabilities) = %d", len(plugin.Capabilities))
	}
	if plugin.Capabilities[0].Match.Regex != `^\s*=` {
		t.Fatalf("calculator match regex = %q", plugin.Capabilities[0].Match.Regex)
	}
}

func TestBuiltinPluginsDeclareMatchRules(t *testing.T) {
	cases := []struct {
		name    string
		pkg     string
		id      string
		mode    string
		pattern string
		regex   string
	}{
		{
			name:  "calculator regex trigger",
			pkg:   "./plugins/ra-calculator",
			id:    "calculate",
			regex: `^\s*=`,
		},
		{
			name:    "app launcher token matcher",
			pkg:     "./plugins/ra-app-launcher",
			id:      "apps",
			mode:    "contains_all_tokens",
			pattern: "app apps application launch",
		},
		{
			name:    "plugin manager token matcher",
			pkg:     "./plugins/ra-plugin-manager",
			id:      "manage",
			mode:    "contains_all_tokens",
			pattern: "plugin plugins manager",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			raw := buildTestPlugin(t, tc.pkg)
			plugin, err := loadWASMBytes(raw, "builtin", tc.pkg)
			if err != nil {
				t.Fatal(err)
			}
			capability, ok := findCapability(plugin.Capabilities, tc.id)
			if !ok {
				t.Fatalf("capability %q not found", tc.id)
			}
			if capability.Match.Regex != tc.regex {
				t.Fatalf("regex = %q, want %q", capability.Match.Regex, tc.regex)
			}
			if capability.Match.Mode != tc.mode {
				t.Fatalf("mode = %q, want %q", capability.Match.Mode, tc.mode)
			}
			if capability.Match.Pattern != tc.pattern {
				t.Fatalf("pattern = %q, want %q", capability.Match.Pattern, tc.pattern)
			}
		})
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

	runPluginSearch = func(ctx context.Context, plugin Plugin, request raplugin.SearchRequest, api pluginruntime.HostAPI) ([]raplugin.SearchResult, error) {
		id := string(plugin.Raw)
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
		done <- registry.SearchWithContext(context.Background(), SearchRequest{Query: "query", Limit: 10})
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

func TestSearchWithContextStopsStartingPluginSearchesAfterCancellation(t *testing.T) {
	previousGOMAXPROCS := runtime.GOMAXPROCS(1)
	t.Cleanup(func() {
		runtime.GOMAXPROCS(previousGOMAXPROCS)
	})
	previousSearch := runPluginSearch
	t.Cleanup(func() {
		runPluginSearch = previousSearch
	})

	ctx, cancel := context.WithCancel(context.Background())
	started := make(chan string, 3)
	release := make(chan struct{})
	var releaseOnce sync.Once
	t.Cleanup(func() {
		releaseOnce.Do(func() { close(release) })
	})

	runPluginSearch = func(ctx context.Context, plugin Plugin, request raplugin.SearchRequest, api pluginruntime.HostAPI) ([]raplugin.SearchResult, error) {
		started <- string(plugin.Raw)
		cancel()
		<-release
		return []raplugin.SearchResult{{
			ID:    "result:" + string(plugin.Raw),
			Title: string(plugin.Raw),
			Action: raplugin.Action{
				CapabilityID: "main",
			},
		}}, nil
	}

	registry := Registry{Plugins: []Plugin{
		searchTestPlugin("plugin-0"),
		searchTestPlugin("plugin-1"),
		searchTestPlugin("plugin-2"),
	}}
	done := make(chan []Result, 1)
	go func() {
		done <- registry.SearchWithContext(ctx, SearchRequest{Query: "query", Limit: 10})
	}()

	waitStarted(t, started, 1)
	select {
	case id := <-started:
		t.Fatalf("started %s after cancellation", id)
	case <-time.After(50 * time.Millisecond):
	}

	releaseOnce.Do(func() { close(release) })
	var results []Result
	select {
	case results = <-done:
	case <-time.After(time.Second):
		t.Fatal("search did not finish")
	}
	if len(results) != 0 {
		t.Fatalf("results = %#v", results)
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

func searchTestPluginWithMatch(id string, match raplugin.Match) Plugin {
	plugin := searchTestPlugin(id)
	plugin.Capabilities[0].Match = match
	matcher, err := compileCapabilityMatcher(match)
	if err != nil {
		panic(err)
	}
	plugin.Capabilities[0].matcher = matcher
	return plugin
}

func testCapabilityWithMatch(t *testing.T, id string, match raplugin.Match, disabled bool) Capability {
	t.Helper()
	matcher, err := compileCapabilityMatcher(match)
	if err != nil {
		t.Fatal(err)
	}
	return Capability{
		ID:       id,
		UI:       "/" + id + ".html",
		Match:    match,
		Disabled: disabled,
		matcher:  matcher,
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
