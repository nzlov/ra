//go:build wasip1 || (js && wasm)

package raplugin

import "unsafe"

var allocations [][]byte
var searchResult []byte

//go:wasmexport ra_manifest_ptr
func raManifestPtr() uint32 {
	return dataPtr(ManifestData())
}

//go:wasmexport ra_manifest_len
func raManifestLen() uint32 {
	return uint32(len(ManifestData()))
}

//go:wasmexport ra_capabilities_ptr
func raCapabilitiesPtr() uint32 {
	return dataPtr(CapabilitiesData())
}

//go:wasmexport ra_capabilities_len
func raCapabilitiesLen() uint32 {
	return uint32(len(CapabilitiesData()))
}

//go:wasmexport ra_assets_ptr
func raAssetsPtr() uint32 {
	return dataPtr(AssetsData())
}

//go:wasmexport ra_assets_len
func raAssetsLen() uint32 {
	return uint32(len(AssetsData()))
}

//go:wasmexport ra_alloc
func raAlloc(size uint32) uint32 {
	if size == 0 {
		return 0
	}
	buffer := make([]byte, size)
	allocations = append(allocations, buffer)
	return dataPtr(buffer)
}

//go:wasmexport ra_search
func raSearch(ptr uint32, size uint32) uint64 {
	input := unsafe.Slice((*byte)(unsafe.Pointer(uintptr(ptr))), size)
	searchResult = SearchData(append([]byte(nil), input...))
	return packPtrLen(dataPtr(searchResult), uint32(len(searchResult)))
}

func dataPtr(data []byte) uint32 {
	if len(data) == 0 {
		return 0
	}
	return uint32(uintptr(unsafe.Pointer(&data[0])))
}

func packPtrLen(ptr uint32, size uint32) uint64 {
	return uint64(ptr)<<32 | uint64(size)
}
