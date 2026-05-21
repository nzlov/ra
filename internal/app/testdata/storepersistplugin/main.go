package main

import "github.com/nzlov/ra/pkg/raplugin"

func init() {
	raplugin.Register(raplugin.Plugin{
		Manifest: raplugin.Manifest{ID: "store-persist", Name: "Store Persist", Version: "0.1.0"},
		Capabilities: []raplugin.Capability{{
			ID:       "persist",
			Title:    "Store Persist",
			UI:       "/persist/index.html",
			Keywords: []string{"store", "persist"},
		}},
		Assets: map[string][]byte{"/persist/index.html": []byte("<main>persist</main>")},
		Search: search,
	})
}

func search(request raplugin.SearchRequest) []raplugin.SearchResult {
	if request.Query != "store persist" {
		return nil
	}

	var state struct {
		Count int `json:"count"`
	}
	_, _ = raplugin.StoreGet("state", &state)
	state.Count++
	_ = raplugin.StoreSet("state", state)

	return []raplugin.SearchResult{{
		ID:       "store-persist",
		Title:    "count",
		Subtitle: string(rune('0' + state.Count)),
		Kind:     "capability",
		Action: raplugin.Action{
			Type:         "capability.open",
			CapabilityID: "persist",
		},
	}}
}

func main() {}
