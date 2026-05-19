# RA Plugin Contract

RA currently supports a minimal local plugin package format. Built-in development plugins live under `plugins/<plugin-id>/`; user plugins live under `~/.local/share/ra/plugins/<plugin-id>/`.

The built-in `ra-plugin-manager` entry appears in launcher search results as a protected plugin. It opens the plugin management view and talks to RA through controlled service APIs instead of manipulating files from plugin page JavaScript.

The built-in `ra-app-launcher` plugin provides desktop app search and launch. It is not protected, so it can be disabled in the plugin manager, but it cannot be uninstalled because it is built into RA.

## Package Layout

```text
plugins/example-webview/
  manifest.json
  index.html
  plugin.js
  plugin.wasm
```

`plugin.wasm` is optional for page plugins. The page can load it with standard browser WebAssembly APIs.

## Manifest

```json
{
  "id": "example-webview",
  "name": "Example Webview",
  "type": "webview",
  "entry": "index.html",
  "permissions": ["clipboard:write"],
  "commands": [
    {
      "id": "open",
      "title": "Open Example Plugin",
      "subtitle": "HTML page with a WASM slot"
    }
  ]
}
```

Rules:

- `id` and command IDs must match `^[a-z0-9][a-z0-9-_.]*$`.
- `type` is `webview` or `command`.
- `entry` must be a relative path inside the plugin directory.
- `commands` are surfaced in the launcher search results.

## Runtime Behavior

When a `webview` command is selected, RA returns a `plugin.open` action result with a `file://` URL for the plugin entry page. The frontend opens that URL in a new browser/webview target.

Calculator results use `clipboard.write`; on Linux RA writes through `wl-copy` when available, then falls back to `xclip -selection clipboard`.

Command plugins run a WASM module through `wazero`. The current command ABI is intentionally tiny: the selected command names a no-argument export that returns one `i32`. RA displays that integer result and copies it when clipboard support is available.

```json
{
  "id": "answer-command",
  "name": "Answer Command",
  "type": "command",
  "entry": "answer.wasm",
  "permissions": ["clipboard:write"],
  "commands": [
    {
      "id": "answer",
      "title": "Run Answer WASM",
      "subtitle": "Returns 42 and copies it",
      "export": "answer"
    }
  ]
}
```

## Plugin Management

The built-in plugin manager supports:

- Listing built-in plugins, user plugins, and load failures.
- Enabling or disabling plugins by writing `~/.config/ra/plugins.json`.
- Installing from a selected local plugin directory into `~/.local/share/ra/plugins/<plugin-id>`.
- Uninstalling user plugins by deleting their user plugin directory.
- Refreshing the registry after management actions.

Boundaries:

- Remote plugin markets are not supported yet.
- Plugin ID conflicts are rejected; install never overwrites an existing plugin.
- Built-in plugins cannot be uninstalled.
- `ra-plugin-manager` cannot be disabled, uninstalled, or replaced by an external plugin package.
- `ra-app-launcher` can be disabled, but external plugin packages cannot replace its built-in ID.
- External manifests may declare `webview` or `command`; `manager` and `app` are reserved for RA itself.

## MVP Boundary

The current command WASM ABI only supports `() -> i32`. The intended next step is to add explicit host functions for clipboard, storage, app launch, and structured UI result generation.
