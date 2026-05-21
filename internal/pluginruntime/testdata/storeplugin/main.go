package main

import "github.com/nzlov/ra/pkg/raplugin"

func init() {
	raplugin.Register(raplugin.Plugin{
		Manifest: raplugin.Manifest{ID: "store-plugin", Name: "Store Plugin", Version: "0.1.0"},
		Capabilities: []raplugin.Capability{{
			ID:    "store",
			Title: "Store",
			UI:    "/store/index.html",
		}},
		Assets: map[string][]byte{"/store/index.html": []byte("<main>store</main>")},
		Search: func(request raplugin.SearchRequest) []raplugin.SearchResult {
			if request.Query != "store smoke" {
				return nil
			}
			setErr := raplugin.StoreSet("checks/one", map[string]any{"value": "ok"})
			var got map[string]any
			found, getErr := raplugin.StoreGet("checks/one", &got)
			title := "store-fail"
			subtitle := ""
			if found && got["value"] == "ok" {
				title = "store-ok"
			} else if setErr != nil {
				subtitle = setErr.Error()
			} else if getErr != nil {
				subtitle = getErr.Error()
			}
			return []raplugin.SearchResult{{
				ID:       "store",
				Title:    title,
				Subtitle: subtitle,
				Kind:     "capability",
				Action: raplugin.Action{
					Type:         "capability.open",
					CapabilityID: "store",
				},
			}}
		},
	})
}

func main() {}
