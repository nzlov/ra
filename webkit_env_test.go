package main

import (
	"os"
	"testing"
)

func TestConfigureWebKitEnvironmentEnablesDMABufRenderer(t *testing.T) {
	t.Setenv("WEBKIT_DISABLE_DMABUF_RENDERER", "1")

	configureWebKitEnvironment()

	if got := os.Getenv("WEBKIT_DISABLE_DMABUF_RENDERER"); got != "0" {
		t.Fatalf("WEBKIT_DISABLE_DMABUF_RENDERER = %q, want 0", got)
	}
}
