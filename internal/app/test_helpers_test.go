package app

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"testing"

	"github.com/nzlov/ra/internal/plugins"
	builtinplugins "github.com/nzlov/ra/plugins"
)

var builtinOnce struct {
	sync.Once
	items []plugins.BuiltinPlugin
}

var testPluginCache sync.Map

func builtinTestPlugins(t *testing.T) []plugins.BuiltinPlugin {
	t.Helper()
	builtinOnce.Do(func() {
		builtinOnce.items = builtinplugins.List()
	})
	items := cloneBuiltinPlugins(builtinOnce.items)
	for _, item := range items {
		if len(item.Raw) == 0 {
			t.Fatalf("builtin plugin %q did not build", item.Name)
		}
	}
	return items
}

func writeCodecPlugin(t *testing.T, root string) string {
	t.Helper()
	return writeTestPlugin(t, root, "codec-tools", "./internal/app/testdata/codecplugin")
}

func writeCodecPluginNoPermissions(t *testing.T, root string) string {
	t.Helper()
	return writeTestPlugin(t, root, "codec-tools", "./internal/app/testdata/codecpluginnoperms")
}

func writeCodecPluginAppLaunch(t *testing.T, root string) string {
	t.Helper()
	return writeTestPlugin(t, root, "codec-tools", "./internal/app/testdata/codecpluginapplaunch")
}

func writeFakeAppLauncherPlugin(t *testing.T, root string) string {
	t.Helper()
	return writeTestPlugin(t, root, "ra-app-launcher", "./internal/app/testdata/fakeapplauncher")
}

func writeTestPlugin(t *testing.T, root string, id string, pkg string) string {
	t.Helper()
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(root, id+".wasm")
	if cached, ok := testPluginCache.Load(pkg); ok {
		if err := os.WriteFile(output, cached.([]byte), 0o644); err != nil {
			t.Fatal(err)
		}
		return output
	}
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
		t.Fatalf("build test plugin %s: %v: %s", id, err, stderr.String())
	}
	raw, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	testPluginCache.Store(pkg, raw)
	return output
}

func cloneBuiltinPlugins(items []plugins.BuiltinPlugin) []plugins.BuiltinPlugin {
	out := make([]plugins.BuiltinPlugin, 0, len(items))
	for _, item := range items {
		out = append(out, plugins.BuiltinPlugin{
			Name: item.Name,
			Raw:  append([]byte(nil), item.Raw...),
		})
	}
	return out
}

func repoRootForTest(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test file")
	}
	return filepath.Dir(filepath.Dir(filepath.Dir(file)))
}
