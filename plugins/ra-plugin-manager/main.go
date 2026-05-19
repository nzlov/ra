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
			ID:          "ra-plugin-manager",
			Name:        "RA Plugin Manager",
			Version:     "0.1.0",
			Permissions: []string{"plugins:manage"},
		},
		Capabilities: []raplugin.Capability{{
			ID:       "manage",
			Title:    "Plugin Manager",
			UI:       "/manager/index.html",
			Icon:     "/icons/plugins.svg",
			Keywords: []string{"plugin", "plugins", "manager"},
		}},
		Assets: raplugin.MustAssets(assets, "assets"),
	})
}

func main() {}
