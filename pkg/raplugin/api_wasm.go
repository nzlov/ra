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

type storeGetResponse struct {
	OK    bool            `json:"ok"`
	Error string          `json:"error,omitempty"`
	Found bool            `json:"found"`
	Value json.RawMessage `json:"value,omitempty"`
}

type storeListResponse struct {
	OK    bool            `json:"ok"`
	Error string          `json:"error,omitempty"`
	Items json.RawMessage `json:"items,omitempty"`
}

type storeWriteResponse struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

func StoreGet(key string, target any) (bool, error) {
	var response storeGetResponse
	if err := callHost("store.get", struct {
		Key string `json:"key"`
	}{Key: key}, &response); err != nil {
		return false, err
	}
	if !response.OK {
		if response.Error == "" {
			response.Error = "store.get failed"
		}
		return false, errors.New(response.Error)
	}
	if !response.Found {
		return false, nil
	}
	if target != nil {
		if err := json.Unmarshal(response.Value, target); err != nil {
			return false, err
		}
	}
	return true, nil
}

func StoreSet(key string, value any) error {
	var response storeWriteResponse
	if err := callHost("store.set", struct {
		Key   string `json:"key"`
		Value any    `json:"value"`
	}{Key: key, Value: value}, &response); err != nil {
		return err
	}
	if !response.OK {
		if response.Error == "" {
			response.Error = "store.set failed"
		}
		return errors.New(response.Error)
	}
	return nil
}

func StoreDelete(key string) error {
	var response storeWriteResponse
	if err := callHost("store.delete", struct {
		Key string `json:"key"`
	}{Key: key}, &response); err != nil {
		return err
	}
	if !response.OK {
		if response.Error == "" {
			response.Error = "store.delete failed"
		}
		return errors.New(response.Error)
	}
	return nil
}

func StoreList(prefix string, target any) error {
	var response storeListResponse
	if err := callHost("store.list", struct {
		Prefix string `json:"prefix"`
	}{Prefix: prefix}, &response); err != nil {
		return err
	}
	if !response.OK {
		if response.Error == "" {
			response.Error = "store.list failed"
		}
		return errors.New(response.Error)
	}
	if target != nil {
		return json.Unmarshal(response.Items, target)
	}
	return nil
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
