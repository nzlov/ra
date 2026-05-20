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
		Search: searchManager,
	})
}

func searchManager(request raplugin.SearchRequest) []raplugin.SearchResult {
	query := strings.ToLower(strings.TrimSpace(request.Query))
	if query != "" && !matches(query, "plugin plugins manager") {
		return nil
	}
	return []raplugin.SearchResult{{
		ID:       "capability:ra-plugin-manager:manage",
		Title:    "Plugin Manager",
		Subtitle: "RA Plugin Manager",
		Kind:     "capability",
		Action: raplugin.Action{
			Type:         "capability.open",
			CapabilityID: "manage",
			Query:        request.Query,
		},
	}}
}

func matches(query string, text string) bool {
	for _, token := range strings.Fields(query) {
		if !strings.Contains(text, token) {
			return false
		}
	}
	return true
}

func main() {}
