package main

import (
	"embed"
	"log"
	"sync"

	"github.com/nzlov/ra/internal/app"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

// Wails uses Go's `embed` package to embed the frontend files into the binary.
// Any files in the frontend/dist folder will be embedded into the binary and
// made available to the frontend.
// See https://pkg.go.dev/embed for more information.

//go:embed frontend/dist
var assets embed.FS

func main() {
	configureWebKitEnvironment()

	launcher := app.NewDefaultLauncherService()
	var searchWindow application.Window
	showSearchWindow := func() {
		if searchWindow == nil {
			return
		}
		searchWindow.Show().Focus()
	}

	wailsApp := application.New(application.Options{
		Name:        "RA",
		Description: "Linux-first launcher with single-WASM plugins",
		Services: []application.Service{
			application.NewServiceWithOptions(launcher, application.ServiceOptions{Route: "/plugins"}),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: false,
		},
		Linux: application.LinuxOptions{
			ProgramName: "ra",
		},
		SingleInstance: &application.SingleInstanceOptions{
			UniqueID: "nzlov.ra.launcher",
			OnSecondInstanceLaunch: func(application.SecondInstanceData) {
				showSearchWindow()
			},
		},
	})

	searchWindow = wailsApp.Window.NewWithOptions(launcherWindowOptions())
	searchWindow.RegisterHook(events.Common.WindowClosing, func(event *application.WindowEvent) {
		event.Cancel()
		searchWindow.Hide()
	})
	configureTray(wailsApp, searchWindow)
	registerInitialSearchWindowShow(searchWindow, showSearchWindow)

	if err := wailsApp.Run(); err != nil {
		log.Fatal(err)
	}
}

type windowHookRegistrar interface {
	RegisterHook(events.WindowEventType, func(*application.WindowEvent)) func()
}

func registerInitialSearchWindowShow(window windowHookRegistrar, showSearchWindow func()) {
	var once sync.Once
	window.RegisterHook(events.Common.WindowRuntimeReady, func(*application.WindowEvent) {
		once.Do(showSearchWindow)
	})
}

func configureTray(wailsApp *application.App, searchWindow application.Window) {
	menu := application.NewMenu()
	menu.Add("Open RA").OnClick(func(*application.Context) {
		searchWindow.Show().Focus()
	})
	menu.AddSeparator()
	menu.Add("Quit RA").OnClick(func(*application.Context) {
		wailsApp.Quit()
	})

	tray := wailsApp.SystemTray.New()
	tray.SetLabel("RA")
	tray.SetTooltip("RA")
	tray.SetIcon(trayIcon())
	tray.SetMenu(menu)
	tray.AttachWindow(searchWindow)
	tray.OnClick(func() {
		searchWindow.Show().Focus()
	})
}

func launcherWindowOptions() application.WebviewWindowOptions {
	return application.WebviewWindowOptions{
		Name:             "search",
		Title:            "RA",
		Width:            920,
		Height:           560,
		AlwaysOnTop:      true,
		DisableResize:    true,
		Frameless:        true,
		Hidden:           true,
		InitialPosition:  application.WindowCentered,
		BackgroundType:   application.BackgroundTypeTransparent,
		BackgroundColour: application.NewRGBA(0, 0, 0, 0),
		Mac: application.MacWindow{
			Backdrop:    application.MacBackdropTransparent,
			TitleBar:    application.MacTitleBarHidden,
			WindowLevel: application.MacWindowLevelFloating,
			CollectionBehavior: application.MacWindowCollectionBehaviorCanJoinAllSpaces |
				application.MacWindowCollectionBehaviorFullScreenAuxiliary,
		},
		Linux: application.LinuxWindow{
			WindowIsTranslucent: true,
		},
		URL: "/",
	}
}

func trayIcon() []byte {
	return []byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24">
  <rect width="24" height="24" rx="5" fill="#111827"/>
  <path d="M6 17V7h7a4 4 0 0 1 1.6 7.7L18 17h-4.1l-2.9-2.1H9V17H6Zm3-5h4a1.5 1.5 0 0 0 0-3H9v3Z" fill="#f9fafb"/>
</svg>`)
}
