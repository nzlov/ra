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
			ID:          "ra-app-launcher",
			Name:        "RA App Launcher",
			Version:     "0.1.0",
			Permissions: []string{"apps:read", "apps:launch"},
		},
		Capabilities: []raplugin.Capability{{
			ID:       "apps",
			Title:    "Applications",
			UI:       "/apps/index.html",
			Icon:     "/icons/apps.svg",
			Keywords: []string{"app", "apps", "application", "launch"},
		}},
		Assets: raplugin.MustAssets(assets, "assets"),
		Search: searchApps,
	})
}

func searchApps(request raplugin.SearchRequest) []raplugin.SearchResult {
	var results []raplugin.SearchResult
	for _, app := range request.Apps {
		if !raplugin.Matches(app.Name+" "+app.Comment, request.Query) {
			continue
		}
		results = append(results, raplugin.SearchResult{
			ID:       "app:" + app.ID,
			Title:    app.Name,
			Subtitle: app.Comment,
			Kind:     "app",
			Action: raplugin.Action{
				Type:         "app.launch",
				AppID:        app.ID,
				Command:      app.Command,
				PluginID:     "ra-app-launcher",
				CapabilityID: "apps",
			},
		})
	}
	return results
}

func main() {}
