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
			ID:          "ra-calculator",
			Name:        "RA Calculator",
			Version:     "0.1.0",
			Permissions: []string{"clipboard:write"},
		},
		Capabilities: []raplugin.Capability{{
			ID:       "calculate",
			Title:    "Calculator",
			UI:       "/calculator/index.html",
			Icon:     "/icons/calculator.svg",
			Keywords: []string{"=", "calculator", "calc", "math"},
		}},
		Assets: raplugin.MustAssets(assets, "assets"),
		Search: searchCalculator,
	})
}

func searchCalculator(request raplugin.SearchRequest) []raplugin.SearchResult {
	if len(request.Query) == 0 || request.Query[0] != '=' {
		return nil
	}
	return []raplugin.SearchResult{{
		ID:       "capability:ra-calculator:calculate",
		Title:    "Calculator",
		Subtitle: "RA Calculator",
		Kind:     "capability",
		Action: raplugin.Action{
			Type:         "capability.open",
			PluginID:     "ra-calculator",
			CapabilityID: "calculate",
			UI:           "/calculator/index.html",
			Query:        request.Query,
		},
	}}
}

func main() {}
