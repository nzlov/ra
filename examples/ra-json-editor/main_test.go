package main

import (
	"testing"

	"github.com/nzlov/ra/pkg/raplugin"
)

func TestSearchTriggersJsonEditor(t *testing.T) {
	for _, query := range []string{"json", "json edit", "json format"} {
		t.Run(query, func(t *testing.T) {
			results := search(raplugin.SearchRequest{Query: query})
			if len(results) != 1 {
				t.Fatalf("search(%q) returned %d results, want 1", query, len(results))
			}
			result := results[0]
			if result.Action.Type != "capability.open" {
				t.Fatalf("action type = %q, want capability.open", result.Action.Type)
			}
			if result.Action.CapabilityID != "editor" {
				t.Fatalf("capability id = %q, want editor", result.Action.CapabilityID)
			}
			if result.Action.Query != query {
				t.Fatalf("action query = %q, want %q", result.Action.Query, query)
			}
		})
	}
}

func TestSearchPassesJSONQueryThrough(t *testing.T) {
	query := `{"name":"ra","features":["format","validate"]}`

	results := search(raplugin.SearchRequest{Query: query})
	if len(results) != 1 {
		t.Fatalf("search returned %d results, want 1", len(results))
	}
	if results[0].Action.Query != query {
		t.Fatalf("action query = %q, want original JSON text", results[0].Action.Query)
	}
}

func TestSearchIgnoresUnrelatedQuery(t *testing.T) {
	results := search(raplugin.SearchRequest{Query: "calculator"})
	if len(results) != 0 {
		t.Fatalf("search returned %d results, want 0", len(results))
	}
}
