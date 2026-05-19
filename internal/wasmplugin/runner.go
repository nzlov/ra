package wasmplugin

import (
	"context"
	"fmt"
	"os"

	"github.com/tetratelabs/wazero"
)

type Runner struct{}

func NewRunner() Runner {
	return Runner{}
}

func (Runner) CallI32(ctx context.Context, path string, exportName string) (int32, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	runtime := wazero.NewRuntime(ctx)
	defer runtime.Close(ctx)

	module, err := runtime.Instantiate(ctx, raw)
	if err != nil {
		return 0, err
	}
	fn := module.ExportedFunction(exportName)
	if fn == nil {
		return 0, fmt.Errorf("wasm export %q not found", exportName)
	}
	values, err := fn.Call(ctx)
	if err != nil {
		return 0, err
	}
	if len(values) != 1 {
		return 0, fmt.Errorf("wasm export %q returned %d values", exportName, len(values))
	}
	return int32(values[0]), nil
}
