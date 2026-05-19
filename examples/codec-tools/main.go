package main

import (
	"embed"

	"github.com/nzlov/ra/pkg/raplugin"
)

//go:embed assets/**
var assets embed.FS

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
				Icon:     "/icons/codec.svg",
				Keywords: []string{"base64", "b64"},
			},
			{
				ID:       "json-xml",
				Title:    "JSON to XML",
				UI:       "/json-xml/index.html",
				Icon:     "/icons/codec.svg",
				Keywords: []string{"json", "xml"},
			},
		},
		Assets: raplugin.MustAssets(assets, "assets"),
	})
}

func main() {}
