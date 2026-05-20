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
	case strings.Contains(query, "json") || strings.Contains(query, "xml"):
		capabilityID = "json-xml"
		title = "JSON to XML"
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
