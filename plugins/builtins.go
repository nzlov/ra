package builtinplugins

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	pluginregistry "github.com/nzlov/ra/internal/plugins"
)

var builtinIDs = []string{"ra-app-launcher", "ra-calculator", "ra-plugin-manager"}
var compiledBuiltinPlugins []pluginregistry.BuiltinPlugin
var compiledBuiltinOK bool

func List() []pluginregistry.BuiltinPlugin {
	if compiledBuiltinOK {
		return cloneBuiltins(compiledBuiltinPlugins)
	}
	items := make([]pluginregistry.BuiltinPlugin, 0, len(builtinIDs))
	for _, id := range builtinIDs {
		raw, err := buildFromSource(id)
		name := id
		if err != nil {
			raw = nil
			name = id + ": " + err.Error()
		}
		items = append(items, pluginregistry.BuiltinPlugin{Name: name, Raw: raw})
	}
	return items
}

func cloneBuiltins(items []pluginregistry.BuiltinPlugin) []pluginregistry.BuiltinPlugin {
	out := make([]pluginregistry.BuiltinPlugin, 0, len(items))
	for _, item := range items {
		out = append(out, pluginregistry.BuiltinPlugin{
			Name: item.Name,
			Raw:  append([]byte(nil), item.Raw...),
		})
	}
	return out
}

func buildFromSource(id string) ([]byte, error) {
	root, err := repoRoot()
	if err != nil {
		return nil, err
	}
	output := filepath.Join(os.TempDir(), "ra-builtin-"+id+".wasm")
	cmd := exec.Command("go", "build", "-buildvcs=false", "-buildmode=c-shared", "-o", output, "./plugins/"+id)
	cmd.Dir = root
	cmd.Env = append(os.Environ(),
		"GOOS=wasip1",
		"GOARCH=wasm",
		"GOCACHE="+filepath.Join(os.TempDir(), "ra-plugin-gocache"),
	)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("build plugin: %w: %s", err, stderr.String())
	}
	raw, err := os.ReadFile(output)
	_ = os.Remove(output)
	return raw, err
}

func repoRoot() (string, error) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("resolve plugin source root")
	}
	return filepath.Dir(filepath.Dir(file)), nil
}
