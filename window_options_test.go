package main

import (
	"testing"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
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
	if options.Name != "search" {
		t.Fatalf("Name = %q, want search", options.Name)
	}
	if !options.Hidden {
		t.Fatal("Hidden = false, want true")
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

func TestInitialSearchWindowShowIsRegisteredForWindowRuntimeReady(t *testing.T) {
	registrar := &recordingWindowHookRegistrar{}
	showCalls := 0

	registerInitialSearchWindowShow(registrar, func() {
		showCalls++
	})

	if !registrar.registered {
		t.Fatal("initial search window show was not registered")
	}
	if registrar.eventType != events.Common.WindowRuntimeReady {
		t.Fatalf("registered event = %d, want WindowRuntimeReady", registrar.eventType)
	}
	if showCalls != 0 {
		t.Fatalf("show called before runtime ready: got %d calls", showCalls)
	}

	registrar.callback(&application.WindowEvent{})

	if showCalls != 1 {
		t.Fatalf("show calls after runtime ready = %d, want 1", showCalls)
	}

	registrar.callback(&application.WindowEvent{})

	if showCalls != 1 {
		t.Fatalf("show calls after second runtime ready = %d, want 1", showCalls)
	}
}

type recordingWindowHookRegistrar struct {
	registered bool
	eventType  events.WindowEventType
	callback   func(*application.WindowEvent)
}

func (r *recordingWindowHookRegistrar) RegisterHook(eventType events.WindowEventType, callback func(*application.WindowEvent)) func() {
	r.registered = true
	r.eventType = eventType
	r.callback = callback
	return func() {}
}
