package main

import (
	"embed"
	"strings"

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
			Match:    raplugin.Match{Mode: "contains_all_tokens", Pattern: "app apps application launch"},
		}},
		Assets: raplugin.MustAssets(assets, "assets"),
		Search: searchApps,
	})
}

func searchApps(request raplugin.SearchRequest) []raplugin.SearchResult {
	apps, err := raplugin.AppsList()
	if err != nil {
		return nil
	}
	query := strings.ToLower(strings.TrimSpace(request.Query))
	var results []raplugin.SearchResult
	for _, app := range apps {
		if !matchesApp(app, query) {
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
				CapabilityID: "apps",
			},
		})
		if request.Limit > 0 && len(results) >= request.Limit {
			break
		}
	}
	return results
}

func matchesApp(app raplugin.App, query string) bool {
	if query == "" {
		return true
	}
	haystack := strings.ToLower(app.Name + " " + app.Comment)
	for _, token := range strings.Fields(query) {
		if !strings.Contains(haystack, token) {
			return false
		}
	}
	return true
}

func main() {}
