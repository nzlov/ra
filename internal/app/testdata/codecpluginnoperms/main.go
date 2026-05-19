package main

import "github.com/nzlov/ra/pkg/raplugin"

func init() {
	raplugin.Register(raplugin.Plugin{
		Manifest: raplugin.Manifest{
			ID:      "codec-tools",
			Name:    "Codec Tools",
			Version: "0.1.0",
		},
		Capabilities: []raplugin.Capability{{
			ID:    "base64",
			Title: "Base64 Convert",
			UI:    "/base64/index.html",
		}},
		Assets: map[string][]byte{
			"/base64/index.html": []byte("<main>base64</main>"),
		},
	})
}

func main() {}
