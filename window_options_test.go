package main

import (
	"testing"

	"github.com/wailsapp/wails/v3/pkg/application"
)

func TestLauncherWindowOptionsAreFloatingAndTransparent(t *testing.T) {
	options := launcherWindowOptions()

	if options.Width != 920 {
		t.Fatalf("Width = %d, want 920", options.Width)
	}
	if options.Height != 560 {
		t.Fatalf("Height = %d, want 560", options.Height)
	}
	if !options.Frameless {
		t.Fatal("Frameless = false, want true")
	}
	if !options.AlwaysOnTop {
		t.Fatal("AlwaysOnTop = false, want true")
	}
	if !options.DisableResize {
		t.Fatal("DisableResize = false, want true")
	}
	if options.InitialPosition != application.WindowCentered {
		t.Fatalf("InitialPosition = %d, want WindowCentered", options.InitialPosition)
	}
	if options.BackgroundType != application.BackgroundTypeTransparent {
		t.Fatalf("BackgroundType = %d, want BackgroundTypeTransparent", options.BackgroundType)
	}
	if options.BackgroundColour != application.NewRGBA(0, 0, 0, 0) {
		t.Fatalf("BackgroundColour = %#v, want transparent", options.BackgroundColour)
	}
	if !options.Linux.WindowIsTranslucent {
		t.Fatal("Linux.WindowIsTranslucent = false, want true")
	}
}
