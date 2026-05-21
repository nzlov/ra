package main

import (
	"embed"
	"encoding/json"
	"strings"

	"github.com/nzlov/ra/pkg/raplugin"
)

//go:embed assets/**
var assets embed.FS

const capabilityID = "editor"

func init() {
	raplugin.Register(raplugin.Plugin{
		Manifest: raplugin.Manifest{
			ID:      "ra-json-editor",
			Name:    "JSON Editor",
			Version: "0.1.0",
		},
		Capabilities: []raplugin.Capability{
			{
				ID:       capabilityID,
				Title:    "JSON Editor",
				UI:       "/editor/index.html",
				Icon:     "/icons/json.svg",
				Keywords: []string{"json", "json edit", "json format"},
			},
		},
		Assets: raplugin.MustAssets(assets, "assets"),
		Search: search,
	})
}

func search(request raplugin.SearchRequest) []raplugin.SearchResult {
	query := strings.TrimSpace(request.Query)
	lowerQuery := strings.ToLower(query)
	if !matchesJSONEditor(lowerQuery) && !looksLikeJSON(query) {
		return nil
	}

	return []raplugin.SearchResult{{
		ID:       "capability:ra-json-editor:" + capabilityID,
		Title:    "JSON Editor",
		Subtitle: "Format, minify, and validate JSON",
		Kind:     "capability",
		Action: raplugin.Action{
			Type:         "capability.open",
			CapabilityID: capabilityID,
			Query:        request.Query,
		},
	}}
}

func matchesJSONEditor(query string) bool {
	return query == "json" || strings.Contains(query, "json edit") || strings.Contains(query, "json format")
}

func looksLikeJSON(query string) bool {
	if query == "" {
		return false
	}
	var value any
	return json.Unmarshal([]byte(query), &value) == nil
}

func main() {}
