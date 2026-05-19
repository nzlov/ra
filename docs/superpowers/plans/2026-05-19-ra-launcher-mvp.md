# RA Launcher MVP Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a Wails v3 Linux launcher MVP with application search, calculator results, plugin manifest discovery, and an HTML/WASM plugin package shape.

**Architecture:** Keep launcher behavior in small Go packages under `internal/`, expose one Wails service from `internal/app`, and keep the Svelte frontend as a thin search/result view. Plugin execution is deliberately narrow in this pass: manifests and webview entries are discovered and exposed, while WASM host APIs are documented and represented by an example plugin package.

**Tech Stack:** Go 1.21, Wails v3 alpha.7, Svelte + TypeScript + Vite, Linux desktop entries, local plugin manifests.

---

### Task 1: Project Skeleton

**Files:**
- Modify: `go.mod`
- Modify: `main.go`
- Delete: `greetservice.go`
- Create: `internal/app/service.go`
- Create: `README.md`

- [ ] Rename module to `github.com/nzlov/ra`.
- [ ] Replace the Wails demo service with `LauncherService`.
- [ ] Update README with build and run commands.

### Task 2: Desktop Entries

**Files:**
- Create: `internal/desktop/desktop.go`
- Create: `internal/desktop/desktop_test.go`

- [ ] Add tests for parsing desktop names, comments, exec commands, hidden entries, and search ranking.
- [ ] Implement desktop entry parsing and search.
- [ ] Verify with `go test ./internal/desktop`.

### Task 3: Calculator

**Files:**
- Create: `internal/calculator/calculator.go`
- Create: `internal/calculator/calculator_test.go`

- [ ] Add tests for arithmetic, parentheses, invalid expressions, and query prefix behavior.
- [ ] Implement a small expression parser for `+ - * /` and parentheses.
- [ ] Verify with `go test ./internal/calculator`.

### Task 4: Plugin Registry

**Files:**
- Create: `internal/plugins/plugins.go`
- Create: `internal/plugins/plugins_test.go`

- [ ] Add tests for valid webview manifests, invalid IDs, missing entries, and command result generation.
- [ ] Implement manifest loading and validation.
- [ ] Verify with `go test ./internal/plugins`.

### Task 5: Launcher Service

**Files:**
- Create: `internal/app/service.go`
- Create: `internal/app/service_test.go`

- [ ] Add tests for merged search results and action dispatch.
- [ ] Implement service methods for frontend search, refresh, and action invocation.
- [ ] Verify with `go test ./internal/app`.

### Task 6: Frontend

**Files:**
- Modify: `frontend/src/App.svelte`
- Modify: `frontend/public/style.css`

- [ ] Replace the template UI with a launcher input and stable result list.
- [ ] Wire frontend to generated Wails bindings when available and use a fallback for local Vite rendering.
- [ ] Verify with `npm run build` inside `frontend`.

### Task 7: Example Plugin and Docs

**Files:**
- Create: `plugins/example-webview/manifest.json`
- Create: `plugins/example-webview/index.html`
- Create: `plugins/example-webview/plugin.js`
- Create: `plugins/example-webview/wasm/README.md`
- Create: `docs/plugins.md`

- [ ] Add a page-style plugin package.
- [ ] Document command and webview plugin contract.
- [ ] Verify plugin manifests load in Go tests.

### Task 8: Final Verification

- [ ] Run `go test ./...`.
- [ ] Run `npm run build` in `frontend`.
- [ ] Run `wails3 task build` or record why it is unavailable.
