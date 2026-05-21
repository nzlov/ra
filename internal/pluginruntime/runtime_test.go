package pluginruntime

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/nzlov/ra/pkg/raplugin"
)

func TestLoadReadsRealGoWASMPluginExports(t *testing.T) {
	wasm := buildTestPlugin(t, "basicplugin")

	plugin, err := Load(wasm)
	if err != nil {
		t.Fatal(err)
	}
	if plugin.Manifest.ID != "codec-tools" {
		t.Fatalf("ID = %q", plugin.Manifest.ID)
	}
	if plugin.Manifest.Permissions[0] != "clipboard:write" {
		t.Fatalf("Permissions = %#v", plugin.Manifest.Permissions)
	}
	if len(plugin.Capabilities) != 1 || plugin.Capabilities[0].ID != "base64" {
		t.Fatalf("Capabilities = %#v", plugin.Capabilities)
	}
	if string(plugin.Assets["/base64/index.html"]) != "<main>base64</main>" {
		t.Fatalf("Assets = %#v", plugin.Assets)
	}
}

func TestSearchCallsRealGoWASMPluginWithRAAPIData(t *testing.T) {
	wasm := buildTestPlugin(t, "appsearch")

	results, err := Search(wasm, raplugin.SearchRequest{
		Query: "fire",
		Limit: 5,
	}, HostAPI{
		Permissions: []string{"apps:read", "apps:launch"},
		Apps: []raplugin.App{{
			ID:      "firefox",
			Name:    "Firefox",
			Comment: "Browser",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("len(results) = %d: %#v", len(results), results)
	}
	result := results[0]
	if result.Title != "Firefox" || result.Action.Type != "app.launch" {
		t.Fatalf("result = %#v", result)
	}
	if result.Action.AppID != "firefox" {
		t.Fatalf("action = %#v", result.Action)
	}
}

func TestSearchWithContextReturnsCanceledContextError(t *testing.T) {
	wasm := buildTestPlugin(t, "appsearch")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := SearchWithContext(ctx, wasm, raplugin.SearchRequest{Query: "fire"})
	if err == nil {
		t.Fatal("SearchWithContext error = nil, want context canceled")
	}
	if err != context.Canceled {
		t.Fatalf("SearchWithContext error = %v, want %v", err, context.Canceled)
	}
}

func TestCompiledPluginSearchesWithoutRecompilingWASM(t *testing.T) {
	wasm := buildTestPlugin(t, "appsearch")
	compiled, err := Compile(wasm)
	if err != nil {
		t.Fatal(err)
	}
	defer compiled.Close()

	for _, query := range []string{"fire", "browser"} {
		results, err := compiled.Search(raplugin.SearchRequest{
			Query: query,
			Limit: 5,
		}, HostAPI{
			Permissions: []string{"apps:read", "apps:launch"},
			Apps: []raplugin.App{{
				ID:      "firefox",
				Name:    "Firefox",
				Comment: "Browser",
			}},
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(results) != 1 || results[0].Action.AppID != "firefox" {
			t.Fatalf("results for %q = %#v", query, results)
		}
	}
}

func TestCompiledPluginSerializesConcurrentSearches(t *testing.T) {
	wasm := buildTestPlugin(t, "appsearch")
	compiled, err := Compile(wasm)
	if err != nil {
		t.Fatal(err)
	}
	defer compiled.Close()

	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results, err := compiled.Search(raplugin.SearchRequest{
				Query: "fire",
				Limit: 5,
			}, HostAPI{
				Permissions: []string{"apps:read", "apps:launch"},
				Apps: []raplugin.App{{
					ID:      "firefox",
					Name:    "Firefox",
					Comment: "Browser",
				}},
			})
			if err != nil {
				t.Error(err)
				return
			}
			if len(results) != 1 || results[0].Action.AppID != "firefox" {
				t.Errorf("results = %#v", results)
			}
		}()
	}
	wg.Wait()
}

func TestHostAPIRejectsAppListWithoutPermission(t *testing.T) {
	wasm := buildTestPlugin(t, "appapidenied")

	results, err := Search(wasm, raplugin.SearchRequest{Query: "apps"}, HostAPI{
		Apps: []raplugin.App{{ID: "firefox", Name: "Firefox"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Title != "denied" {
		t.Fatalf("results = %#v", results)
	}
}

func TestStoreHostAPIReadsAndWritesPluginScopedJSON(t *testing.T) {
	wasm := buildTestPlugin(t, "storeplugin")
	store := newMemoryStoreHostAPI()

	results, err := Search(wasm, raplugin.SearchRequest{Query: "store smoke"}, store.hostAPI())
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("len(results) = %d: %#v", len(results), results)
	}
	if results[0].Title != "store-ok" {
		t.Fatalf("result = %#v", results[0])
	}
}

func TestStoreHostAPIListDeleteAndMissingKey(t *testing.T) {
	store := newMemoryStoreHostAPI()
	rt := Runtime{api: store.hostAPI()}

	if response := rt.handleHostRequest(mustJSON(t, hostRequest{
		Method: "store.get",
		Params: mustJSON(t, struct {
			Key string `json:"key"`
		}{Key: "missing"}),
	})); !response.OK || response.Found {
		t.Fatalf("missing get response = %#v", response)
	}

	if response := rt.handleHostRequest(mustJSON(t, hostRequest{
		Method: "store.set",
		Params: mustJSON(t, struct {
			Key   string         `json:"key"`
			Value map[string]any `json:"value"`
		}{Key: "papers/current", Value: map[string]any{"id": "paper-1"}}),
	})); !response.OK {
		t.Fatalf("set current response = %#v", response)
	}
	if response := rt.handleHostRequest(mustJSON(t, hostRequest{
		Method: "store.set",
		Params: mustJSON(t, struct {
			Key   string         `json:"key"`
			Value map[string]any `json:"value"`
		}{Key: "papers/archive", Value: map[string]any{"id": "paper-0"}}),
	})); !response.OK {
		t.Fatalf("set archive response = %#v", response)
	}
	if response := rt.handleHostRequest(mustJSON(t, hostRequest{
		Method: "store.set",
		Params: mustJSON(t, struct {
			Key   string         `json:"key"`
			Value map[string]any `json:"value"`
		}{Key: "settings/theme", Value: map[string]any{"id": "theme"}}),
	})); !response.OK {
		t.Fatalf("set theme response = %#v", response)
	}

	response := rt.handleHostRequest(mustJSON(t, hostRequest{
		Method: "store.list",
		Params: mustJSON(t, struct {
			Prefix string `json:"prefix"`
		}{Prefix: "papers/"}),
	}))
	if !response.OK {
		t.Fatalf("list response = %#v", response)
	}
	var values []map[string]any
	if err := json.Unmarshal(response.Items, &values); err != nil {
		t.Fatal(err)
	}
	if ids := valueIDs(values); len(ids) != 2 || ids[0] != "paper-0" || ids[1] != "paper-1" {
		t.Fatalf("ids = %#v", ids)
	}

	if response := rt.handleHostRequest(mustJSON(t, hostRequest{
		Method: "store.delete",
		Params: mustJSON(t, struct {
			Key string `json:"key"`
		}{Key: "papers/current"}),
	})); !response.OK {
		t.Fatalf("delete response = %#v", response)
	}
	if response := rt.handleHostRequest(mustJSON(t, hostRequest{
		Method: "store.get",
		Params: mustJSON(t, struct {
			Key string `json:"key"`
		}{Key: "papers/current"}),
	})); !response.OK || response.Found {
		t.Fatalf("deleted get response = %#v", response)
	}
}

func buildTestPlugin(t *testing.T, name string) []byte {
	t.Helper()
	output := filepath.Join(t.TempDir(), name+".wasm")
	cmd := exec.Command("go", "build", "-buildvcs=false", "-buildmode=c-shared", "-o", output, "./testdata/"+name)
	cmd.Env = append(os.Environ(),
		"GOOS=wasip1",
		"GOARCH=wasm",
		"GOCACHE="+filepath.Join(t.TempDir(), "gocache"),
	)
	cmd.Dir = packageDir(t)
	raw, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build test plugin: %v\n%s", err, raw)
	}
	wasm, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	return wasm
}

func packageDir(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return wd
}

type memoryStoreHostAPI struct {
	values map[string]json.RawMessage
}

func newMemoryStoreHostAPI() *memoryStoreHostAPI {
	return &memoryStoreHostAPI{values: map[string]json.RawMessage{}}
}

func (s *memoryStoreHostAPI) hostAPI() HostAPI {
	return HostAPI{
		StoreGet: func(key string) (json.RawMessage, bool, error) {
			value, ok := s.values[key]
			return append(json.RawMessage(nil), value...), ok, nil
		},
		StoreSet: func(key string, value json.RawMessage) error {
			s.values[key] = append(json.RawMessage(nil), value...)
			return nil
		},
		StoreDelete: func(key string) error {
			delete(s.values, key)
			return nil
		},
		StoreList: func(prefix string) (json.RawMessage, error) {
			keys := make([]string, 0, len(s.values))
			for key := range s.values {
				if strings.HasPrefix(key, prefix) {
					keys = append(keys, key)
				}
			}
			sort.Strings(keys)
			items := make([]json.RawMessage, 0, len(keys))
			for _, key := range keys {
				items = append(items, s.values[key])
			}
			return json.Marshal(items)
		},
	}
}

func mustJSON(t *testing.T, value any) json.RawMessage {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func valueIDs(values []map[string]any) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, value["id"].(string))
	}
	return out
}
