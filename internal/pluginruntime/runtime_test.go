package pluginruntime

import (
	"os"
	"os/exec"
	"path/filepath"
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
