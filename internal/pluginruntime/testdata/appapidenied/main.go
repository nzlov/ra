package main

import "github.com/nzlov/ra/pkg/raplugin"

func init() {
	raplugin.Register(raplugin.Plugin{
		Manifest: raplugin.Manifest{
			ID:      "app-denied",
			Name:    "App Denied",
			Version: "0.1.0",
		},
		Capabilities: []raplugin.Capability{{
			ID:    "apps",
			Title: "Applications",
			UI:    "/apps/index.html",
		}},
		Search: func(request raplugin.SearchRequest) []raplugin.SearchResult {
			apps, err := raplugin.AppsList()
			title := "denied"
			if err == nil {
				title = "allowed"
			}
			return []raplugin.SearchResult{{
				ID:    "apps-check",
				Title: title,
				Kind:  "capability",
				Action: raplugin.Action{
					Type:         "capability.open",
					CapabilityID: "apps",
					Query:        request.Query,
				},
				Subtitle: string(rune(len(apps))),
			}}
		},
	})
}

func main() {}
