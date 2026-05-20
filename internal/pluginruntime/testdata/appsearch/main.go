package main

import "github.com/nzlov/ra/pkg/raplugin"

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
			Keywords: []string{"app", "apps"},
		}},
		Assets: map[string][]byte{
			"/apps/index.html": []byte("<main>apps</main>"),
		},
		Search: func(request raplugin.SearchRequest) []raplugin.SearchResult {
			apps, err := raplugin.AppsList()
			if err != nil {
				return nil
			}
			var results []raplugin.SearchResult
			for _, app := range apps {
				if app.Name != "Firefox" {
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
			}
			return results
		},
	})
}

func main() {}
