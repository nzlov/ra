package main

import "github.com/nzlov/ra/pkg/raplugin"

func init() {
	raplugin.Register(raplugin.Plugin{
		Manifest: raplugin.Manifest{
			ID:          "codec-tools",
			Name:        "Codec Tools",
			Version:     "0.1.0",
			Permissions: []string{"clipboard:write"},
		},
		Capabilities: []raplugin.Capability{
			{
				ID:       "base64",
				Title:    "Base64 Convert",
				UI:       "/base64/index.html",
				Icon:     "/icons/base64.svg",
				Keywords: []string{"base64", "b64"},
			},
			{
				ID:       "json",
				Title:    "JSON Convert",
				UI:       "/json/index.html",
				Keywords: []string{"json"},
			},
		},
		Assets: map[string][]byte{
			"/base64/index.html": []byte("<main>base64</main>"),
			"/json/index.html":   []byte("<main>json</main>"),
			"/icons/base64.svg":  []byte("<svg></svg>"),
		},
	})
}

func main() {}
