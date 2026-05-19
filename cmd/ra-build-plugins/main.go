package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

var builtinIDs = []string{"ra-app-launcher", "ra-calculator", "ra-plugin-manager"}

func main() {
	root, err := os.Getwd()
	if err != nil {
		fatal(err)
	}
	outputPath := filepath.Join(root, "plugins", "builtins_data.go")
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		fatal(err)
	}
	var out bytes.Buffer
	out.WriteString("package builtinplugins\n\n")
	out.WriteString("import (\n")
	out.WriteString("\t\"encoding/base64\"\n\n")
	out.WriteString("\tpluginregistry \"github.com/nzlov/ra/internal/plugins\"\n")
	out.WriteString(")\n\n")
	out.WriteString("func init() {\n")
	out.WriteString("\tcompiledBuiltinOK = true\n")
	out.WriteString("\tcompiledBuiltinPlugins = []pluginregistry.BuiltinPlugin{\n")
	for _, id := range builtinIDs {
		raw, err := buildPlugin(root, id)
		if err != nil {
			fatal(err)
		}
		out.WriteString("\t\t{Name: ")
		out.WriteString(quote(id))
		out.WriteString(", Raw: mustDecodeBuiltin(")
		out.WriteString(quote(base64.StdEncoding.EncodeToString(raw)))
		out.WriteString(")},\n")
	}
	out.WriteString("\t}\n")
	out.WriteString("}\n\n")
	out.WriteString("func mustDecodeBuiltin(value string) []byte {\n")
	out.WriteString("\traw, err := base64.StdEncoding.DecodeString(value)\n")
	out.WriteString("\tif err != nil { panic(err) }\n")
	out.WriteString("\treturn raw\n")
	out.WriteString("}\n")
	if err := os.WriteFile(outputPath, out.Bytes(), 0o644); err != nil {
		fatal(err)
	}
}

func buildPlugin(root string, id string) ([]byte, error) {
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
		return nil, fmt.Errorf("build %s: %w: %s", id, err, stderr.String())
	}
	raw, err := os.ReadFile(output)
	_ = os.Remove(output)
	return raw, err
}

func quote(value string) string {
	return fmt.Sprintf("%q", value)
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
