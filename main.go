package main

import (
	"embed"
	"log"

	"github.com/nzlov/ra/internal/app"
	"github.com/wailsapp/wails/v3/pkg/application"
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
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
		Linux: application.LinuxOptions{
			ProgramName: "ra",
		},
	})

	wailsApp.Window.NewWithOptions(launcherWindowOptions())

	if err := wailsApp.Run(); err != nil {
		log.Fatal(err)
	}
}

func launcherWindowOptions() application.WebviewWindowOptions {
	return application.WebviewWindowOptions{
		Title:            "RA",
		Width:            920,
		Height:           560,
		AlwaysOnTop:      true,
		DisableResize:    true,
		Frameless:        true,
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
