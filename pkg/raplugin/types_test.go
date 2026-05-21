package raplugin

import (
	"encoding/json"
	"testing"
)

func TestCapabilityMatchMarshalsAsObject(t *testing.T) {
	raw, err := json.Marshal(Capability{
		ID:    "calculate",
		Title: "Calculator",
		UI:    "/calculator/index.html",
		Match: Match{Regex: `^\s*=`},
	})
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != `{"id":"calculate","title":"Calculator","ui":"/calculator/index.html","match":{"regex":"^\\s*="}}` {
		t.Fatalf("marshal = %s", raw)
	}
}
