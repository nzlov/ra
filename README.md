# RA

RA is a Linux-first launcher prototype built with Go and Wails v3. It aims for a small uTools-like workflow: search apps, run quick commands, and expose local plugins.

## Current MVP

- Scans `.desktop` files from `/usr/share/applications` and `~/.local/share/applications`.
- Provides app search and launch through the built-in `ra-app-launcher` plugin.
- Supports calculator queries with `=`, for example `=6*7`.
- Loads local plugin manifests from `plugins/*/manifest.json`.
- Also loads user plugins from `~/.local/share/ra/plugins`.
- Provides the built-in `ra-plugin-manager` plugin for local plugin install, enable, disable, uninstall, and refresh workflows.
- Includes a webview plugin package shape with `index.html`, `plugin.js`, and an optional `plugin.wasm`.
- Opens webview plugin entries through `file://` URLs returned from the Go service.
- Writes calculator results to the clipboard through `wl-copy` or `xclip` when available.
- Runs command-style WASM plugins through `wazero` for no-argument `() -> i32` exports.

## Requirements

- Go 1.25+
- Wails v3 alpha CLI installed with `go install github.com/wailsapp/wails/v3/cmd/wails3@latest`
- Node.js and npm
- Linux desktop environment with GTK4/WebKitGTK 6.0 dependencies required by Wails

On CachyOS/Arch, the relevant packages are `base-devel`, `gtk4`, and `webkitgtk-6.0`.

## Development

```sh
CGO_ENABLED=1 go test ./...
cd frontend
npm install
npm run build
cd ..
wails3 generate bindings -f '-gcflags=all="-l"' -ts
wails3 dev
```

For a production-style local build:

```sh
CGO_ENABLED=1 wails3 task build
```

This machine has `CGO_ENABLED=0` in the Go environment. Wails on Linux needs cgo for WebKitGTK, so set `CGO_ENABLED=1` when testing or building the desktop target.

## Plugin Format

See `docs/plugins.md` for the current local plugin contract. RA can open webview plugin entries and run narrow command WASM plugins with a `() -> i32` export.

Development plugins can stay in the repository `plugins/` directory. User-installed plugins should live under `~/.local/share/ra/plugins/<plugin-id>/`.

Plugin enable/disable state is stored in `~/.config/ra/plugins.json`. The plugin manager can disable built-in plugins such as `ra-app-launcher`, but it only uninstalls user plugins and refuses to disable or uninstall itself.

## Next Steps

- Add explicit host APIs for clipboard, storage, app launch, and result rendering.
- Expand the command WASM ABI beyond `() -> i32`.
- Add Niri-friendly show/hide integration and document a compositor keybinding.
