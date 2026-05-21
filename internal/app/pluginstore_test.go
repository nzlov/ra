package app

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestPluginStoreSetGetDeleteList(t *testing.T) {
	store := openTestPluginStore(t)
	defer store.Close()

	if err := store.Set("ra-calculator", "papers/current", map[string]any{"id": "paper-1"}); err != nil {
		t.Fatal(err)
	}
	if err := store.Set("ra-calculator", "papers/archive", map[string]any{"id": "paper-0"}); err != nil {
		t.Fatal(err)
	}
	if err := store.Set("ra-calculator", "settings/theme", map[string]any{"name": "plain"}); err != nil {
		t.Fatal(err)
	}

	var got map[string]any
	found, err := store.Get("ra-calculator", "papers/current", &got)
	if err != nil {
		t.Fatal(err)
	}
	if !found || got["id"] != "paper-1" {
		t.Fatalf("found=%v got=%#v", found, got)
	}

	var papers []map[string]any
	if err := store.List("ra-calculator", "papers/", &papers); err != nil {
		t.Fatal(err)
	}
	if ids := valueIDs(papers); !reflect.DeepEqual(ids, []string{"paper-0", "paper-1"}) {
		t.Fatalf("paper ids = %#v", ids)
	}

	if err := store.Delete("ra-calculator", "papers/current"); err != nil {
		t.Fatal(err)
	}
	found, err = store.Get("ra-calculator", "papers/current", &got)
	if err != nil {
		t.Fatal(err)
	}
	if found {
		t.Fatal("deleted value was found")
	}
}

func TestPluginStoreIsolationByPluginID(t *testing.T) {
	store := openTestPluginStore(t)
	defer store.Close()

	if err := store.Set("ra-calculator", "shared", map[string]any{"id": "calculator"}); err != nil {
		t.Fatal(err)
	}
	if err := store.Set("ra-json-editor", "shared", map[string]any{"id": "json"}); err != nil {
		t.Fatal(err)
	}

	var got map[string]any
	found, err := store.Get("ra-json-editor", "shared", &got)
	if err != nil {
		t.Fatal(err)
	}
	if !found || got["id"] != "json" {
		t.Fatalf("found=%v got=%#v", found, got)
	}

	var calculatorValues []map[string]any
	if err := store.List("ra-calculator", "", &calculatorValues); err != nil {
		t.Fatal(err)
	}
	if ids := valueIDs(calculatorValues); !reflect.DeepEqual(ids, []string{"calculator"}) {
		t.Fatalf("calculator ids = %#v", ids)
	}
}

func TestPluginStoreNilTargets(t *testing.T) {
	store := openTestPluginStore(t)
	defer store.Close()

	if err := store.Set("ra-calculator", "papers/current", map[string]any{"id": "paper-1"}); err != nil {
		t.Fatal(err)
	}

	found, err := store.Get("ra-calculator", "papers/current", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !found {
		t.Fatal("expected nil-target get to report found")
	}

	if err := store.List("ra-calculator", "papers/", nil); err != nil {
		t.Fatal(err)
	}
}

func openTestPluginStore(t *testing.T) *PluginStore {
	t.Helper()
	store, err := OpenPluginStore(filepath.Join(t.TempDir(), "plugin-store.db"))
	if err != nil {
		t.Fatal(err)
	}
	return store
}

func valueIDs(values []map[string]any) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, value["id"].(string))
	}
	return out
}
