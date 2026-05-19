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
	launcher := app.NewDefaultLauncherService()

	wailsApp := application.New(application.Options{
		Name:        "RA",
		Description: "Linux-first launcher with HTML/WASM plugins",
		Services: []application.Service{
			application.NewService(launcher),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	wailsApp.NewWebviewWindowWithOptions(application.WebviewWindowOptions{
		Title: "RA",
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 50,
			Backdrop:                application.MacBackdropTranslucent,
			TitleBar:                application.MacTitleBarHiddenInset,
		},
		BackgroundColour: application.NewRGB(18, 20, 23),
		URL:              "/",
	})

	if err := wailsApp.Run(); err != nil {
		log.Fatal(err)
	}
}
