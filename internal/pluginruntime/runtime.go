package pluginruntime

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/nzlov/ra/pkg/raplugin"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

const callTimeout = 2 * time.Second

type Plugin struct {
	Manifest     raplugin.Manifest
	Capabilities []raplugin.Capability
	Assets       map[string][]byte
	Raw          []byte
}

type Runtime struct {
	raw []byte
}

func Load(raw []byte) (Plugin, error) {
	rt := Runtime{raw: append([]byte(nil), raw...)}
	manifestRaw, err := rt.readExport("ra_manifest_ptr", "ra_manifest_len")
	if err != nil {
		return Plugin{}, fmt.Errorf("read manifest: %w", err)
	}
	var manifest raplugin.Manifest
	if err := json.Unmarshal(manifestRaw, &manifest); err != nil {
		return Plugin{}, fmt.Errorf("read manifest: %w", err)
	}

	capabilitiesRaw, err := rt.readExport("ra_capabilities_ptr", "ra_capabilities_len")
	if err != nil {
		return Plugin{}, fmt.Errorf("read capabilities: %w", err)
	}
	var capabilities []raplugin.Capability
	if len(capabilitiesRaw) > 0 {
		if err := json.Unmarshal(capabilitiesRaw, &capabilities); err != nil {
			return Plugin{}, fmt.Errorf("read capabilities: %w", err)
		}
	}

	assetsRaw, err := rt.readExport("ra_assets_ptr", "ra_assets_len")
	if err != nil {
		return Plugin{}, fmt.Errorf("read assets: %w", err)
	}
	assets, err := decodeAssets(assetsRaw)
	if err != nil {
		return Plugin{}, err
	}

	return Plugin{
		Manifest:     manifest,
		Capabilities: capabilities,
		Assets:       assets,
		Raw:          append([]byte(nil), raw...),
	}, nil
}

func Search(raw []byte, request raplugin.SearchRequest) ([]raplugin.SearchResult, error) {
	rt := Runtime{raw: append([]byte(nil), raw...)}
	input, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), callTimeout)
	defer cancel()
	r, mod, err := rt.instantiate(ctx)
	if err != nil {
		return nil, err
	}
	defer r.Close(ctx)

	alloc := mod.ExportedFunction("ra_alloc")
	search := mod.ExportedFunction("ra_search")
	if alloc == nil || search == nil {
		return nil, errors.New("missing plugin search exports")
	}
	ptrs, err := alloc.Call(ctx, uint64(len(input)))
	if err != nil {
		return nil, err
	}
	ptr := uint32(ptrs[0])
	if len(input) > 0 && !mod.Memory().Write(ptr, input) {
		return nil, errors.New("write search request to plugin memory")
	}
	packedResults, err := search.Call(ctx, uint64(ptr), uint64(len(input)))
	if err != nil {
		return nil, err
	}
	raw, err = readPackedData(mod.Memory(), packedResults[0])
	if err != nil {
		return nil, err
	}
	var decoded []raplugin.SearchResult
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &decoded); err != nil {
			return nil, fmt.Errorf("read search results: %w", err)
		}
	}
	return decoded, nil
}

func (rt Runtime) readExport(ptrName string, lenName string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), callTimeout)
	defer cancel()
	r, mod, err := rt.instantiate(ctx)
	if err != nil {
		return nil, err
	}
	defer r.Close(ctx)

	ptrFn := mod.ExportedFunction(ptrName)
	lenFn := mod.ExportedFunction(lenName)
	if ptrFn == nil || lenFn == nil {
		return nil, fmt.Errorf("missing plugin exports %q/%q", ptrName, lenName)
	}
	ptrResult, err := ptrFn.Call(ctx)
	if err != nil {
		return nil, err
	}
	lenResult, err := lenFn.Call(ctx)
	if err != nil {
		return nil, err
	}
	ptr := uint32(ptrResult[0])
	size := uint32(lenResult[0])
	if size == 0 {
		return nil, nil
	}
	raw, ok := mod.Memory().Read(ptr, size)
	if !ok {
		return nil, fmt.Errorf("read plugin memory at %d length %d", ptr, size)
	}
	return append([]byte(nil), raw...), nil
}

func (rt Runtime) instantiate(ctx context.Context) (wazero.Runtime, api.Module, error) {
	r := wazero.NewRuntimeWithConfig(ctx, wazero.NewRuntimeConfigInterpreter())
	wasi_snapshot_preview1.MustInstantiate(ctx, r)
	mod, err := r.InstantiateWithConfig(ctx, rt.raw, wazero.NewModuleConfig().WithStartFunctions("_initialize"))
	if err != nil {
		_ = r.Close(ctx)
		return nil, nil, err
	}
	return r, mod, nil
}

func readPackedData(memory api.Memory, packed uint64) ([]byte, error) {
	ptr := uint32(packed >> 32)
	size := uint32(packed)
	if size == 0 {
		return nil, nil
	}
	raw, ok := memory.Read(ptr, size)
	if !ok {
		return nil, fmt.Errorf("read plugin memory at %d length %d", ptr, size)
	}
	return append([]byte(nil), raw...), nil
}

func decodeAssets(raw []byte) (map[string][]byte, error) {
	assets := map[string][]byte{}
	if len(raw) == 0 {
		return assets, nil
	}
	var encoded map[string]string
	if err := json.Unmarshal(raw, &encoded); err != nil {
		return nil, fmt.Errorf("read assets: %w", err)
	}
	for path, value := range encoded {
		data, err := base64.StdEncoding.DecodeString(value)
		if err != nil {
			return nil, fmt.Errorf("read asset %q: %w", path, err)
		}
		assets[path] = data
	}
	return assets, nil
}
