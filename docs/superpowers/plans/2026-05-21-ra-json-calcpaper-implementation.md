# RA JSON Editor And Calc Paper Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a SQLite-backed private plugin storage API to RA, upgrade the built-in calculator into a persistent calc paper plugin, and add a JSON editor example plugin with text/tree editing, formatting, and validation.

**Architecture:** Extend the existing Go/WASM host boundary with a minimal storage API scoped by the executing plugin ID, keep persistence in a single SQLite database managed by the host, upgrade `plugins/ra-calculator` in place, and add `examples/ra-json-editor` as a source-only example plugin. Preserve the current plugin/result model and avoid any host special cases for these plugins.

**Tech Stack:** Go 1.21, `database/sql`, SQLite driver, wazero WASI runtime, Wails v3, embedded plugin HTML/JS assets, Node-based plugin asset tests.

---

### Task 1: Define Plugin Storage Contract In `pkg/raplugin`

**Files:**
- Modify: `pkg/raplugin/api_wasm.go`
- Modify: `pkg/raplugin/types.go`
- Test: `internal/pluginruntime/runtime_test.go`

- [ ] **Step 1: Write the failing runtime integration test for plugin storage host calls**

Add a new test plugin under `internal/pluginruntime/testdata/storeplugin` in a later task, then add a failing assertion in `internal/pluginruntime/runtime_test.go` that expects:

```go
func TestStoreHostAPIReadsAndWritesPluginScopedJSON(t *testing.T) {
	wasm := buildTestPlugin(t, "storeplugin")

	results, err := Search(wasm, raplugin.SearchRequest{Query: "store smoke"})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("len(results) = %d: %#v", len(results), results)
	}
	if results[0].Title != "store-ok" {
		t.Fatalf("result = %#v", results[0])
	}
}
```

- [ ] **Step 2: Run the runtime test to verify it fails for missing storage API**

Run: `go test ./internal/pluginruntime -run TestStoreHostAPIReadsAndWritesPluginScopedJSON -count=1`

Expected: FAIL because the test plugin or storage host method does not exist yet.

- [ ] **Step 3: Add plugin-facing storage helper types and methods**

Update `pkg/raplugin/api_wasm.go` to add minimal wrappers alongside `AppsList()`:

```go
func StoreGet(key string, target any) (bool, error)
func StoreSet(key string, value any) error
func StoreDelete(key string) error
func StoreList(prefix string, target any) error
```

Use `callHost(...)` with four new method names:

- `store.get`
- `store.set`
- `store.delete`
- `store.list`

Use request/response payloads encoded as JSON rather than adding new ABI functions.

- [ ] **Step 4: Keep the contract narrow and JSON-only**

Implement the request/response shapes inside `api_wasm.go` as private structs similar to:

```go
type storeGetResponse struct {
	OK    bool            `json:"ok"`
	Error string          `json:"error,omitempty"`
	Found bool            `json:"found"`
	Value json.RawMessage `json:"value,omitempty"`
}
```

`StoreGet` should return `(false, nil)` when the key does not exist. `StoreList` should unmarshal the host response into the caller-provided `target`.

- [ ] **Step 5: Run the runtime test again to keep it red for the right reason**

Run: `go test ./internal/pluginruntime -run TestStoreHostAPIReadsAndWritesPluginScopedJSON -count=1`

Expected: FAIL because the runtime host does not implement `store.*` yet, not because the wrapper functions are missing.

- [ ] **Step 6: Commit the contract-only change**

```bash
git add pkg/raplugin/api_wasm.go pkg/raplugin/types.go internal/pluginruntime/runtime_test.go
git commit -m "feat: add plugin storage API contract"
```

### Task 2: Add SQLite-Backed Plugin Store In The Host

**Files:**
- Create: `internal/app/pluginstore.go`
- Create: `internal/app/pluginstore_test.go`
- Modify: `internal/app/service.go`
- Test: `internal/app/pluginstore_test.go`

- [ ] **Step 1: Write failing tests for SQLite-backed plugin-scoped storage**

Create `internal/app/pluginstore_test.go` with focused tests like:

```go
func TestPluginStoreSetGetDeleteList(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "plugin-store.db")
	store, err := OpenPluginStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	if err := store.Set("ra-calculator", "papers/current", map[string]any{"id": "paper-1"}); err != nil {
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
}

func TestPluginStoreIsolationByPluginID(t *testing.T) {
	// write as ra-calculator, assert ra-json-editor cannot read it
}
```

- [ ] **Step 2: Run the new plugin store tests to verify they fail**

Run: `go test ./internal/app -run 'TestPluginStore(SetGetDeleteList|IsolationByPluginID)' -count=1`

Expected: FAIL because `OpenPluginStore` and the store implementation do not exist yet.

- [ ] **Step 3: Implement the SQLite storage type**

Create `internal/app/pluginstore.go` with:

```go
type PluginStore struct {
	db *sql.DB
}

func OpenPluginStore(path string) (*PluginStore, error)
func (s *PluginStore) Close() error
func (s *PluginStore) Get(pluginID string, key string, target any) (bool, error)
func (s *PluginStore) Set(pluginID string, key string, value any) error
func (s *PluginStore) Delete(pluginID string, key string) error
func (s *PluginStore) List(pluginID string, prefix string, target any) error
```

Create the schema on open:

```sql
CREATE TABLE IF NOT EXISTS plugin_store (
  plugin_id TEXT NOT NULL,
  key TEXT NOT NULL,
  value_json TEXT NOT NULL,
  updated_at INTEGER NOT NULL,
  PRIMARY KEY (plugin_id, key)
);
```

- [ ] **Step 4: Keep the JSON encoding rules strict**

`Set` should reject values that `json.Marshal` cannot encode. `Get` and `List` should unmarshal JSON back into the caller-provided target and return a clear error on malformed stored JSON.

- [ ] **Step 5: Wire the store path into launcher configuration**

Extend `internal/app/service.go` config with a plugin store database path, defaulting to:

```go
filepath.Join(home, ".config", "ra", "plugin-store.db")
```

Do not yet expose this to plugins; only initialize and keep it available from the service layer.

- [ ] **Step 6: Run the plugin store tests to verify they pass**

Run: `go test ./internal/app -run 'TestPluginStore(SetGetDeleteList|IsolationByPluginID)' -count=1`

Expected: PASS

- [ ] **Step 7: Commit the host store implementation**

```bash
git add internal/app/pluginstore.go internal/app/pluginstore_test.go internal/app/service.go
git commit -m "feat: add sqlite-backed plugin store"
```

### Task 3: Implement Runtime `store.*` Host Methods

**Files:**
- Modify: `internal/pluginruntime/runtime.go`
- Modify: `internal/pluginruntime/runtime_test.go`
- Create: `internal/pluginruntime/testdata/storeplugin/main.go`
- Test: `internal/pluginruntime/runtime_test.go`

- [ ] **Step 1: Add a failing test plugin that uses the new storage contract**

Create `internal/pluginruntime/testdata/storeplugin/main.go`:

```go
package main

import "github.com/nzlov/ra/pkg/raplugin"

func init() {
	raplugin.Register(raplugin.Plugin{
		Manifest: raplugin.Manifest{ID: "store-plugin", Name: "Store Plugin", Version: "0.1.0"},
		Capabilities: []raplugin.Capability{{ID: "store", Title: "Store", UI: "/store/index.html"}},
		Assets: map[string][]byte{"/store/index.html": []byte("<main>store</main>")},
		Search: func(request raplugin.SearchRequest) []raplugin.SearchResult {
			if request.Query != "store smoke" {
				return nil
			}
			_ = raplugin.StoreSet("checks/one", map[string]any{"value": "ok"})
			var got map[string]any
			found, _ := raplugin.StoreGet("checks/one", &got)
			title := "store-fail"
			if found && got["value"] == "ok" {
				title = "store-ok"
			}
			return []raplugin.SearchResult{{ID: "store", Title: title, Kind: "capability", Action: raplugin.Action{Type: "capability.open", CapabilityID: "store"}}}
		},
	})
}

func main() {}
```

- [ ] **Step 2: Run the runtime storage test to verify host methods are still missing**

Run: `go test ./internal/pluginruntime -run TestStoreHostAPIReadsAndWritesPluginScopedJSON -count=1`

Expected: FAIL because `internal/pluginruntime/runtime.go` does not implement `store.get`, `store.set`, `store.delete`, and `store.list`.

- [ ] **Step 3: Extend runtime host API state to carry plugin storage**

Add the minimum host API hook in `internal/pluginruntime/runtime.go`:

```go
type PluginStore interface {
	Get(pluginID string, key string, target any) (bool, error)
	Set(pluginID string, key string, value any) error
	Delete(pluginID string, key string) error
	List(pluginID string, prefix string, target any) error
}

type HostAPI struct {
	Permissions []string
	Apps        []raplugin.App
	Store       PluginStore
	PluginID    string
}
```

`PluginID` must be set by the caller based on the plugin being executed, not from the plugin request payload.

- [ ] **Step 4: Implement the `store.*` methods in `handleHostRequest`**

Add switch branches that:

- decode the request payload
- use `rt.api.PluginID`
- call the configured `Store`
- return JSON responses with `ok`, `error`, and optional `found` / `value` / `items`

Do not add any SQL or filesystem logic here; runtime only forwards to `HostAPI.Store`.

- [ ] **Step 5: Add a plugin ID enforcement test**

Add a second test in `internal/pluginruntime/runtime_test.go` asserting the runtime ignores any attempt to escape plugin scope:

```go
func TestStoreHostAPIScopesByExecutingPluginID(t *testing.T) {
	// configure a fake store, run as plugin A, verify reads/writes land under plugin A only
}
```

- [ ] **Step 6: Run the runtime storage tests to verify they pass**

Run: `go test ./internal/pluginruntime -run 'TestStoreHostAPI(Read|Scopes)' -count=1`

Expected: PASS

- [ ] **Step 7: Commit the runtime host API implementation**

```bash
git add internal/pluginruntime/runtime.go internal/pluginruntime/runtime_test.go internal/pluginruntime/testdata/storeplugin/main.go
git commit -m "feat: add plugin storage host methods"
```

### Task 4: Wire Plugin Store Into Registry / Service Search Execution

**Files:**
- Modify: `internal/plugins/plugins.go`
- Modify: `internal/app/service.go`
- Modify: `internal/app/service_test.go`
- Test: `internal/app/service_test.go`

- [ ] **Step 1: Write a failing service-level test that proves plugin searches receive store access**

Add a test in `internal/app/service_test.go` that loads the new example/test plugin, runs a search, and asserts the plugin can persist and reread data through the launcher service path.

- [ ] **Step 2: Run the service test to verify it fails**

Run: `go test ./internal/app -run TestSearchProvidesPluginScopedStoreAccess -count=1`

Expected: FAIL because the service and registry do not yet pass `PluginID` and `Store` into runtime search calls.

- [ ] **Step 3: Thread plugin store through the search path**

Update the service and plugin registry path so that when a plugin search function runs via WASM:

- `HostAPI.Store` points at the service-owned SQLite store
- `HostAPI.PluginID` is set from the plugin manifest ID currently being searched

Do not mutate global state in `pkg/raplugin`; pass it per runtime invocation.

- [ ] **Step 4: Keep non-storage plugins unaffected**

Preserve the existing behavior for:

- built-in calculator search
- app launcher search
- codec-tools example search

Only extend the host call context; do not change search result stamping behavior or plugin validation.

- [ ] **Step 5: Run focused service tests**

Run: `go test ./internal/app -run 'Test(SearchProvidesPluginScopedStoreAccess|SearchMergesCalculatorAppsAndCapabilities)' -count=1`

Expected: PASS

- [ ] **Step 6: Commit the service integration**

```bash
git add internal/plugins/plugins.go internal/app/service.go internal/app/service_test.go
git commit -m "feat: wire plugin store into runtime searches"
```

### Task 5: Upgrade `ra-calculator` Search And Persistence Model

**Files:**
- Modify: `plugins/ra-calculator/main.go`
- Modify: `internal/app/service_test.go`
- Test: `internal/app/service_test.go`

- [ ] **Step 1: Add a failing test for calculator search behavior that stays compatible**

Extend `internal/app/service_test.go` with a focused assertion that `=6*7` still resolves to `ra-calculator.calculate` and continues to pass the original query through the action.

- [ ] **Step 2: Run the focused calculator search test**

Run: `go test ./internal/app -run TestSearchMergesCalculatorAppsAndCapabilities -count=1`

Expected: either PASS already or FAIL only if later edits regressed behavior. This test becomes the guardrail before UI rewrites.

- [ ] **Step 3: Update `plugins/ra-calculator/main.go` only as needed for calc paper semantics**

Keep:

- plugin ID `ra-calculator`
- capability ID `calculate`
- regex trigger `^\s*=`

Update metadata only if needed to better describe the new calc paper UI, but do not change the external search trigger or capability ID.

- [ ] **Step 4: Add storage key constants for the calc paper plugin**

Inside `plugins/ra-calculator/main.go`, define internal constants that the UI script will also follow conceptually:

```go
const (
	currentPaperKey = "papers/current"
	paperOrderKey   = "papers/order"
	paperPrefix     = "papers/by-id/"
)
```

These are documentation and consistency anchors for the plugin package.

- [ ] **Step 5: Commit the calculator metadata/storage-key prep**

```bash
git add plugins/ra-calculator/main.go internal/app/service_test.go
git commit -m "refactor: prepare calculator plugin for calc paper UI"
```

### Task 6: Replace Calculator UI With Calc Paper UI

**Files:**
- Modify: `plugins/ra-calculator/assets/calculator/index.html`
- Modify: `plugins/ra-calculator/calculate.test.mjs`
- Test: `plugins/ra-calculator/calculate.test.mjs`

- [ ] **Step 1: Replace the existing calculator asset test with a failing calc paper behavior test**

Update `plugins/ra-calculator/calculate.test.mjs` to assert:

- opening with `?q=%3D2%2B1` creates or updates the current paper with a line for `2+1`
- results are computed for valid expressions
- invalid expressions are rendered as error state without throwing
- a second open appends a new line instead of replacing all existing content

Use the same `vm`-style asset test pattern already in the repo.

- [ ] **Step 2: Run the Node asset test to verify it fails**

Run: `node plugins/ra-calculator/calculate.test.mjs`

Expected: FAIL because the current HTML is only a single-query calculator, not a persisted calc paper UI.

- [ ] **Step 3: Implement the calc paper UI in one embedded HTML asset**

Rewrite `plugins/ra-calculator/assets/calculator/index.html` to include:

- left-hand paper list
- current paper editor
- new paper / rename / delete controls
- per-line expression and result rendering
- query parameter import from `q`

Use the new plugin store API from browser-exposed plugin JS rather than `localStorage`.

- [ ] **Step 4: Keep the expression evaluator intentionally minimal**

Reuse or adapt the existing safe arithmetic evaluator already exercised by the old test. Do not add variables, functions, or implicit previous-line references in this pass.

- [ ] **Step 5: Add debounced persistence to plugin storage**

Persist:

- `papers/current`
- `papers/order`
- `papers/by-id/<paper-id>`

Save on edit with a small debounce and flush when switching papers.

- [ ] **Step 6: Run the calculator asset test to verify it passes**

Run: `node plugins/ra-calculator/calculate.test.mjs`

Expected: PASS

- [ ] **Step 7: Commit the calc paper UI**

```bash
git add plugins/ra-calculator/assets/calculator/index.html plugins/ra-calculator/calculate.test.mjs
git commit -m "feat: turn built-in calculator into calc paper"
```

### Task 7: Add The JSON Editor Example Plugin Search Skeleton

**Files:**
- Create: `examples/ra-json-editor/main.go`
- Create: `examples/ra-json-editor/README.md`
- Test: `internal/pluginruntime/runtime_test.go`

- [ ] **Step 1: Write a failing runtime load/search test for the new example plugin**

Add a new testdata-style build or example build assertion that expects:

- plugin ID `ra-json-editor`
- one capability `json-editor`
- search matches `json`
- search also matches complete JSON object/array text

- [ ] **Step 2: Run the focused runtime test to verify it fails**

Run: `go test ./internal/pluginruntime -run TestLoadReadsRAJSONEditorExamplePlugin -count=1`

Expected: FAIL because the example plugin does not exist yet.

- [ ] **Step 3: Create the Go example plugin source**

Create `examples/ra-json-editor/main.go` registering:

```go
raplugin.Plugin{
	Manifest: raplugin.Manifest{
		ID: "ra-json-editor",
		Name: "RA JSON Editor",
		Version: "0.1.0",
	},
	Capabilities: []raplugin.Capability{{
		ID: "json-editor",
		Title: "JSON Editor",
		UI: "/editor/index.html",
		Keywords: []string{"json", "json edit", "json format"},
	}},
	Assets: raplugin.MustAssets(assets, "assets"),
	Search: searchJSONEditor,
}
```

- [ ] **Step 4: Implement minimal search matching**

In `searchJSONEditor`, match:

- queries containing `json`
- queries that trim to a complete JSON object (`{...}`) or array (`[...]`)

Return one `capability.open` result with the original query passed through `Action.Query`.

- [ ] **Step 5: Run the runtime test to verify it passes**

Run: `go test ./internal/pluginruntime -run TestLoadReadsRAJSONEditorExamplePlugin -count=1`

Expected: PASS

- [ ] **Step 6: Commit the JSON editor plugin skeleton**

```bash
git add examples/ra-json-editor/main.go examples/ra-json-editor/README.md internal/pluginruntime/runtime_test.go
git commit -m "feat: add json editor example plugin skeleton"
```

### Task 8: Implement JSON Editor Asset UI

**Files:**
- Create: `examples/ra-json-editor/assets/editor/index.html`
- Create: `examples/ra-json-editor/editor.test.mjs`
- Test: `examples/ra-json-editor/editor.test.mjs`

- [ ] **Step 1: Write the failing JSON editor asset test first**

Create `examples/ra-json-editor/editor.test.mjs` asserting:

- `?q=` with JSON text hydrates the editor
- `format` pretty-prints JSON
- `minify` compresses JSON to one line
- invalid JSON produces a visible validation error
- switching to tree mode preserves the parsed value for valid JSON

- [ ] **Step 2: Run the Node asset test to verify it fails**

Run: `node examples/ra-json-editor/editor.test.mjs`

Expected: FAIL because the UI asset does not exist yet.

- [ ] **Step 3: Implement the editor asset**

Create `examples/ra-json-editor/assets/editor/index.html` with:

- toolbar buttons for format, minify, validate, text/tree toggle, clear
- text editor area
- tree view area
- status area
- initial query parsing from `q`

Keep dependencies embedded or bundled within the asset; do not require host filesystem or network access.

- [ ] **Step 4: Keep tree mode narrow in scope**

Tree mode may be editable or read-mostly, but it must at least:

- show structured JSON nodes
- reflect the current valid parsed document
- switch back to text mode without clearing content

Do not add schema validation, tabs, file-open actions, or external conversions.

- [ ] **Step 5: Optionally persist last draft and view mode**

If the asset implementation stays simple, persist:

- `drafts/latest`
- `ui/view-mode`

through the new plugin store API. If this creates undue complexity, defer it and keep the test focused on the editor behaviors above.

- [ ] **Step 6: Run the JSON editor asset test to verify it passes**

Run: `node examples/ra-json-editor/editor.test.mjs`

Expected: PASS

- [ ] **Step 7: Commit the JSON editor UI**

```bash
git add examples/ra-json-editor/assets/editor/index.html examples/ra-json-editor/editor.test.mjs
git commit -m "feat: implement json editor example UI"
```

### Task 9: Document The New Storage API And Plugin Examples

**Files:**
- Modify: `docs/plugins.md`
- Modify: `README.md`
- Modify: `README.zh-CN.md`
- Modify: `examples/README.md`

- [ ] **Step 1: Add a failing docs coverage checklist in your working notes**

Before editing docs, verify the final docs must mention:

- plugin private storage API
- SQLite store path
- `ra-calculator` as calc paper
- `ra-json-editor` example plugin

This is a manual red step; no command required.

- [ ] **Step 2: Update `docs/plugins.md`**

Document:

- new storage API surface for plugin authors
- plugin-scoped isolation guarantee
- `ra-calculator` current capability behavior
- `ra-json-editor` example plugin behavior and limits

- [ ] **Step 3: Update top-level READMEs**

Add concise user-facing notes to `README.md` and `README.zh-CN.md` covering:

- SQLite-backed plugin storage
- calc paper replacing the simple calculator UI
- JSON editor example plugin scope

- [ ] **Step 4: Update example plugin docs**

Update `examples/README.md` so future contributors know `ra-json-editor` is source-only, not committed as a built `.wasm`, and how to build it for manual testing.

- [ ] **Step 5: Commit the documentation updates**

```bash
git add docs/plugins.md README.md README.zh-CN.md examples/README.md
git commit -m "docs: describe plugin storage and recreated plugins"
```

### Task 10: Final Verification

**Files:**
- Test: `internal/app/...`
- Test: `internal/pluginruntime/...`
- Test: `plugins/ra-calculator/calculate.test.mjs`
- Test: `examples/ra-json-editor/editor.test.mjs`

- [ ] **Step 1: Run focused Go verification**

Run: `go test ./internal/app ./internal/pluginruntime -count=1`

Expected: PASS

- [ ] **Step 2: Run full Go verification**

Run: `go test ./... -count=1`

Expected: PASS

- [ ] **Step 3: Run calculator asset verification**

Run: `node plugins/ra-calculator/calculate.test.mjs`

Expected: PASS

- [ ] **Step 4: Run JSON editor asset verification**

Run: `node examples/ra-json-editor/editor.test.mjs`

Expected: PASS

- [ ] **Step 5: Run frontend type/build verification**

Run: `cd frontend && npm run check && npm run build`

Expected: PASS

- [ ] **Step 6: Run final desktop app build verification**

Run: `wails3 task build`

Expected: PASS, or record the exact blocker if unavailable in the environment.

- [ ] **Step 7: Commit any final verification-only fixes**

```bash
git add -A
git commit -m "test: fix verification issues for plugin storage and recreated plugins"
```
