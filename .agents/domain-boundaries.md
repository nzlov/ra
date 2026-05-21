# Domain Boundaries

## 0. Table of Contents

- [1. Project](#1-project)
- [2. Global Rules](#2-global-rules)
- [3. Bounded Contexts](#3-bounded-contexts)
  - [3.1 Core App Service And Search](#31-core-app-service-and-search)
  - [3.2 Plugin Registry And Management](#32-plugin-registry-and-management)
  - [3.3 Plugin Runtime And Host API](#33-plugin-runtime-and-host-api)
  - [3.4 Plugin Author Contract](#34-plugin-author-contract)
  - [3.5 Built-In Plugins](#35-built-in-plugins)
  - [3.6 Frontend Launcher UI](#36-frontend-launcher-ui)
  - [3.7 Desktop Integration](#37-desktop-integration)
  - [3.8 Documentation And Examples](#38-documentation-and-examples)
- [4. Shared Areas](#4-shared-areas)
- [5. Coordination Rules](#5-coordination-rules)
- [6. Open Questions](#6-open-questions)
- [7. Change Log](#7-change-log)

## 1. Project

- Block: 1
- Name: RA
- Last Updated: 2026-05-21
- Source: mixed
- Summary: Linux-first Wails v3 launcher with desktop app search, local WASM plugins, built-in plugins, and a Svelte frontend.

## 2. Global Rules

- Block: 2
- Prefer reusing these boundaries over re-inferring them.
- Do not assign parallel write tasks across overlapping ownership.
- Keep generated and build outputs out of ordinary implementation tasks unless the task explicitly targets generation or packaging.
- Treat plugin-facing contracts as shared public interfaces; changes there require coordination across runtime, registry, built-in plugins, frontend, docs, and examples.
- Do not revert user changes or other agent changes; adapt or stop and report conflicts.

## 3. Bounded Contexts

### 3.1 Core App Service And Search

- Block: 3.1
- Purpose: Own launcher service orchestration, desktop entry loading, search result shaping, action execution, plugin asset serving, and the service APIs exposed to Wails/frontend.
- Does Not Own: Low-level plugin package validation, WASM execution internals, plugin management persistence policy, plugin source behavior, frontend presentation.
- Primary Paths:
  - internal/app/
  - internal/app/actions.go
  - main.go
  - window_options_test.go
  - webkit_env.go
- Key Modules:
  - LauncherService
  - ActionExecutor
  - plugin config and capability enablement logic
- Public Interfaces:
  - Wails-bound service methods
  - app.Result, app.Action, app.Status, app.PluginActionRequest
- Upstream Dependencies:
  - 3.2 Plugin Registry And Management
  - 3.3 Plugin Runtime And Host API
  - 3.7 Desktop Integration
- Downstream Dependents:
  - 3.6 Frontend Launcher UI
  - 3.5 Built-In Plugins
- Parallel Work Rules:
  - Safe Reads: yes
  - Safe Writes: no
  - Notes: Writes often affect shared behavior and should be serial unless limited to tests or one service method.

### 3.2 Plugin Registry And Management

- Block: 3.2
- Purpose: Own plugin discovery, registry loading, manifest/capability validation, built-in vs user plugin source rules, install/uninstall/enable/disable persistence, and load error reporting.
- Does Not Own: WASM ABI implementation, plugin author package APIs, frontend manager UI, individual plugin behavior.
- Primary Paths:
  - internal/plugins/
  - internal/app/plugin_management.go
- Key Modules:
  - plugins.Registry
  - plugins.Plugin
  - plugin management service operations
- Public Interfaces:
  - registry load/search APIs used by internal/app
  - plugin management APIs used by ra-plugin-manager and frontend
- Upstream Dependencies:
  - 3.3 Plugin Runtime And Host API
  - 3.4 Plugin Author Contract
- Downstream Dependents:
  - 3.1 Core App Service And Search
  - 3.5 Built-In Plugins
  - 3.6 Frontend Launcher UI
- Parallel Work Rules:
  - Safe Reads: yes
  - Safe Writes: no
  - Notes: Coordinate writes with 3.1 when changing management API shapes or config semantics.

### 3.3 Plugin Runtime And Host API

- Block: 3.3
- Purpose: Own WASI/WASM loading, exported plugin data extraction, search invocation, timeout/concurrency behavior, and permission-scoped host API calls.
- Does Not Own: Registry policy, UI asset serving policy, plugin manager workflows, frontend bridge layout.
- Primary Paths:
  - internal/pluginruntime/
- Key Modules:
  - pluginruntime.Runtime
  - pluginruntime.HostAPI
  - WASM host imports
- Public Interfaces:
  - Compile, Load, LoadFromRuntime, Search
  - runtime host API functions callable from plugins
- Upstream Dependencies:
  - 3.4 Plugin Author Contract
- Downstream Dependents:
  - 3.2 Plugin Registry And Management
  - 3.1 Core App Service And Search
- Parallel Work Rules:
  - Safe Reads: yes
  - Safe Writes: no
  - Notes: ABI or host API changes must be coordinated with pkg/raplugin and built-in plugins.

### 3.4 Plugin Author Contract

- Block: 3.4
- Purpose: Own the Go package and WASM-side types/functions used by plugin authors.
- Does Not Own: RA service implementation, registry storage, frontend UI, specific built-in plugin product behavior.
- Primary Paths:
  - pkg/raplugin/
- Key Modules:
  - Manifest
  - Capability
  - SearchRequest
  - SearchResult
  - WASM exports and host API stubs
- Public Interfaces:
  - github.com/nzlov/ra/pkg/raplugin
  - exported WASM symbols consumed by internal/pluginruntime
- Upstream Dependencies:
  - none
- Downstream Dependents:
  - 3.3 Plugin Runtime And Host API
  - 3.2 Plugin Registry And Management
  - 3.5 Built-In Plugins
  - 3.8 Documentation And Examples
- Parallel Work Rules:
  - Safe Reads: yes
  - Safe Writes: no
  - Notes: Treat as a cross-context contract; write serially and update docs/examples/tests together.

### 3.5 Built-In Plugins

- Block: 3.5
- Purpose: Own built-in plugin source packages and their plugin-owned capabilities, search behavior, UI assets, and embedded resources.
- Does Not Own: Core service management rules, registry validation, runtime ABI, frontend launcher shell.
- Primary Paths:
  - plugins/
- Key Modules:
  - ra-app-launcher
  - ra-calculator
  - ra-plugin-manager
- Public Interfaces:
  - plugin manifests, capabilities, assets, and search functions exposed through pkg/raplugin
  - plugins.List for embedded built-ins
- Upstream Dependencies:
  - 3.4 Plugin Author Contract
  - 3.1 Core App Service And Search
  - 3.2 Plugin Registry And Management
- Downstream Dependents:
  - 3.1 Core App Service And Search
  - 3.6 Frontend Launcher UI
- Parallel Work Rules:
  - Safe Reads: yes
  - Safe Writes: yes
  - Notes: Writes are safe by individual plugin directory unless touching plugins/builtins.go or generated built-in data.

### 3.6 Frontend Launcher UI

- Block: 3.6
- Purpose: Own Svelte launcher presentation, client-side search scheduling, window behavior, iframe hosting surface, generated Wails bindings consumption, and frontend tests/build config.
- Does Not Own: Service behavior, plugin registry policy, runtime permission checks, plugin package contracts.
- Primary Paths:
  - frontend/
- Key Modules:
  - frontend/src/App.svelte
  - frontend/src/searchScheduler.js
  - frontend/src/launcherWindowBehavior.js
- Public Interfaces:
  - Wails-generated bindings consumed by frontend code
  - window.ra bridge usage from plugin capability pages
- Upstream Dependencies:
  - 3.1 Core App Service And Search
  - 3.2 Plugin Registry And Management
- Downstream Dependents:
  - none
- Parallel Work Rules:
  - Safe Reads: yes
  - Safe Writes: yes
  - Notes: Do not hand-edit generated bindings unless the task explicitly targets generated output.

### 3.7 Desktop Integration

- Block: 3.7
- Purpose: Own Linux desktop entry parsing, default application directories, app launch data, and desktop/environment integration helpers.
- Does Not Own: Plugin search ranking, frontend layout, registry policy, plugin management UI.
- Primary Paths:
  - internal/desktop/
  - webkit_env.go
- Key Modules:
  - desktop.Entry
  - desktop.LoadDirs
- Public Interfaces:
  - desktop entries and default directory APIs consumed by internal/app
- Upstream Dependencies:
  - Linux desktop files and environment
- Downstream Dependents:
  - 3.1 Core App Service And Search
  - 3.5 Built-In Plugins
- Parallel Work Rules:
  - Safe Reads: yes
  - Safe Writes: yes
  - Notes: Coordinate with 3.1 when changing app entry fields or launch semantics.

### 3.8 Documentation And Examples

- Block: 3.8
- Purpose: Own user-facing docs, plugin contract docs, examples, development notes, and plans.
- Does Not Own: Runtime behavior or API truth when code differs from docs.
- Primary Paths:
  - README.md
  - README.zh-CN.md
  - docs/
  - examples/
  - plugins/README.md
- Key Modules:
  - plugin documentation
  - example plugin packages
- Public Interfaces:
  - documented commands and plugin author guidance
- Upstream Dependencies:
  - all implementation contexts
- Downstream Dependents:
  - developers and plugin authors
- Parallel Work Rules:
  - Safe Reads: yes
  - Safe Writes: yes
  - Notes: Docs-only writes are safe unless claiming behavior from a context that has not been verified.

## 4. Shared Areas

- Block: 4
- Path: pkg/raplugin/
- Risk: high
- Rule: Public contract; coordinate writes across runtime, registry, plugins, docs, and examples.
- Path: internal/app/
- Risk: high
- Rule: Central orchestration; avoid parallel writes unless scopes are disjoint and API shape is unchanged.
- Path: internal/plugins/
- Risk: medium
- Rule: Registry behavior affects app service and plugin manager; coordinate behavioral changes.
- Path: internal/pluginruntime/
- Risk: high
- Rule: Runtime/ABI changes require plugin contract and built-in plugin coordination.
- Path: frontend/bindings/
- Risk: medium
- Rule: Generated Wails bindings; regenerate through the established toolchain rather than hand-editing.
- Path: Taskfile.yml
- Risk: medium
- Rule: Shared build, package, and generation orchestration; do not use as a casual parallel-write target.
- Path: plugins/builtins_data.go
- Risk: medium
- Rule: Generated/build output; do not commit or edit unless the task explicitly requires generated built-in assets.
- Path: bin/
- Risk: low
- Rule: Build output; do not edit or commit for ordinary source tasks.
- Path: frontend/dist/
- Risk: low
- Rule: Build output; do not edit or commit for ordinary frontend tasks.

## 5. Coordination Rules

- Block: 5
- Split work by bounded context.
- Keep cross-context refactors serial by default.
- Require main-agent approval before writing shared areas.
- Assign built-in plugin work by plugin directory when possible.
- Treat changes to plugin management, permissions, result actions, or UI asset routing as cross-context until proven local.
- Verification should match touched contexts: Go tests for backend/runtime, npm tests/build for frontend, Wails generation/build only when bindings or desktop packaging are affected.

## 6. Open Questions

- Block: 6
- None.

## 7. Change Log

- Block: 7
- 2026-05-21 Initial minimal boundary map for RA repository-level DDD orchestration.
- 2026-05-21 Clarified plugin management persistence belongs to registry/management, not core search orchestration.
- 2026-05-21 Added explicit app action, Chinese README, and Taskfile shared-orchestration boundaries.
