package main

import (
	"strings"

	"github.com/nzlov/ra/pkg/raplugin"
)

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
		Search: search,
	})
}

func search(request raplugin.SearchRequest) []raplugin.SearchResult {
	query := strings.ToLower(strings.TrimSpace(request.Query))
	capabilityID := ""
	title := ""
	switch {
	case strings.Contains(query, "base64") || strings.Contains(query, "b64"):
		capabilityID = "base64"
		title = "Base64 Convert"
	case strings.Contains(query, "json"):
		capabilityID = "json"
		title = "JSON Convert"
	default:
		return nil
	}
	return []raplugin.SearchResult{{
		ID:       "capability:codec-tools:" + capabilityID,
		Title:    title,
		Subtitle: "Codec Tools",
		Kind:     "capability",
		Action: raplugin.Action{
			Type:         "capability.open",
			CapabilityID: capabilityID,
			Query:        request.Query,
		},
	}}
}

func main() {}
