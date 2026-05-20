//go:build wasip1 || (js && wasm)

package raplugin

import (
	"encoding/json"
	"errors"
	"unsafe"
)

//go:wasmimport ra call
func hostCall(ptr uint32, size uint32) uint64

func AppsList() ([]App, error) {
	var response struct {
		OK    bool   `json:"ok"`
		Error string `json:"error,omitempty"`
		Apps  []App  `json:"apps,omitempty"`
	}
	if err := callHost("apps.list", nil, &response); err != nil {
		return nil, err
	}
	if !response.OK {
		if response.Error == "" {
			response.Error = "apps.list failed"
		}
		return nil, errors.New(response.Error)
	}
	return response.Apps, nil
}

func callHost(method string, params any, response any) error {
	request := struct {
		Method string `json:"method"`
		Params any    `json:"params,omitempty"`
	}{Method: method, Params: params}
	raw, err := json.Marshal(request)
	if err != nil {
		return err
	}
	packed := hostCall(dataPtr(raw), uint32(len(raw)))
	result := readHostResult(packed)
	if len(result) == 0 {
		return errors.New("empty host response")
	}
	return json.Unmarshal(result, response)
}

func readHostResult(packed uint64) []byte {
	ptr := uint32(packed >> 32)
	size := uint32(packed)
	if size == 0 {
		return nil
	}
	raw := unsafe.Slice((*byte)(unsafe.Pointer(uintptr(ptr))), size)
	return append([]byte(nil), raw...)
}
