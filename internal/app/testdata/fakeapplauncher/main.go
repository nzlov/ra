package main

import "github.com/nzlov/ra/pkg/raplugin"

func init() {
	raplugin.Register(raplugin.Plugin{
		Manifest: raplugin.Manifest{
			ID:      "ra-app-launcher",
			Name:    "Fake Apps",
			Version: "0.1.0",
		},
		Capabilities: []raplugin.Capability{{
			ID:    "apps",
			Title: "Applications",
			UI:    "/apps/index.html",
		}},
		Assets: map[string][]byte{
			"/apps/index.html": []byte("<main>apps</main>"),
		},
	})
}

func main() {}
