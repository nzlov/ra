package main

import "github.com/nzlov/ra/pkg/raplugin"

func init() {
	raplugin.Register(raplugin.Plugin{
		Manifest: raplugin.Manifest{
			ID:          "store-actions-other",
			Name:        "Store Actions Other",
			Version:     "0.1.0",
			Permissions: []string{"store:read", "store:write"},
		},
		Capabilities: []raplugin.Capability{{
			ID:    "persist",
			Title: "Store Actions Other",
			UI:    "/persist/index.html",
		}},
		Assets: map[string][]byte{"/persist/index.html": []byte("<main>persist other</main>")},
	})
}

func main() {}
