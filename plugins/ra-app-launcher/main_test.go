package main

import (
	"testing"

	"github.com/nzlov/ra/pkg/raplugin"
)

func TestSearchAppsHonorsLimitWhileCollectingResults(t *testing.T) {
	raplugin.SetAppsListForTesting([]raplugin.App{
		{ID: "firefox", Name: "Firefox", Comment: "Browser"},
		{ID: "firefox-beta", Name: "Firefox Beta", Comment: "Browser"},
		{ID: "firefox-nightly", Name: "Firefox Nightly", Comment: "Browser"},
	})
	t.Cleanup(raplugin.ResetAppsListForTesting)

	results := searchApps(raplugin.SearchRequest{
		Query: "fire",
		Limit: 2,
	})

	if len(results) != 2 {
		t.Fatalf("len(results) = %d, want 2", len(results))
	}
	if results[0].Title != "Firefox" || results[1].Title != "Firefox Beta" {
		t.Fatalf("results = %#v", results)
	}
}
