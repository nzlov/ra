package main

import "os"

func configureWebKitEnvironment() {
	// Wails disables the DMABUF renderer on NVIDIA before main; on this stack it makes focused inputs repaint on a hot CPU path.
	_ = os.Setenv("WEBKIT_DISABLE_DMABUF_RENDERER", "0")
}
