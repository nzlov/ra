# RA

Simplified Chinese documentation: [README.zh-CN.md](README.zh-CN.md)

RA is a Linux-first launcher prototype built with Go and Wails v3. It aims for a small uTools-like workflow: search apps and open local plugin capabilities.

## Current MVP

- Scans `.desktop` files from `/usr/share/applications` and `~/.local/share/applications`.
- Provides app search and launch through the built-in `ra-app-launcher` plugin.
- Supports calculator queries through the built-in `ra-calculator` plugin, for example `=6*7`.
- Loads built-in plugins from embedded source under `plugins/`.
- Loads user plugin packages from `~/.local/share/ra/plugins/*.wasm`.
- Provides the built-in `ra-plugin-manager` plugin for local plugin install, enable, disable, uninstall, and refresh workflows.
- Models plugins as Go/WASI `.wasm` files built from plugin-owned Go source, manifest, capabilities, permissions, search behavior, and embedded UI assets.
- Supports capability-level enable and disable.
- Serves enabled capability UI assets under `/plugins/<plugin-id>/<capability-id>/...` in a sandboxed iframe.
- Exposes permission-checked RA APIs to plugins, including WASM host APIs such as `apps.list` and UI actions through `window.ra.invoke()`.

## Requirements

- Go 1.25+
- Wails v3 alpha CLI installed with `go install github.com/wailsapp/wails/v3/cmd/wails3@latest`
- Node.js and npm
- Linux desktop environment with GTK4/WebKitGTK 6.0 dependencies required by Wails

On CachyOS/Arch, the relevant packages are `base-devel`, `gtk4`, and `webkitgtk-6.0`.

## Development

```sh
cd frontend
npm install
cd ..
env GOCACHE=/tmp/ra-go-cache CGO_ENABLED=1 go test ./... -count=1
env GOCACHE=/tmp/ra-go-cache CGO_ENABLED=1 wails3 task dev
```

If you want to run the frontend by itself:

```sh
wails3 task dev:frontend
```

If you prefer the explicit manual flow instead of the Wails task wrapper:

```sh
cd frontend
npm run build
cd ..
env GOCACHE=/tmp/ra-go-cache CGO_ENABLED=1 wails3 generate bindings -f '-gcflags=all=\"-l\"' -ts
env GOCACHE=/tmp/ra-go-cache CGO_ENABLED=1 wails3 dev -config ./build/config.yml -port 9245
```

## Build

Build for the current OS:

```sh
env GOCACHE=/tmp/ra-go-cache CGO_ENABLED=1 wails3 task build
```

Common Linux builds:

```sh
env GOCACHE=/tmp/ra-go-cache CGO_ENABLED=1 wails3 task build:linux:debug:amd64
env GOCACHE=/tmp/ra-go-cache CGO_ENABLED=1 wails3 task build:linux:prod:amd64
```

On Linux the built binary is written to `bin/ra`.

## Package

Package for the current OS:

```sh
env GOCACHE=/tmp/ra-go-cache CGO_ENABLED=1 wails3 task package
```

Linux AppImage packaging:

```sh
env GOCACHE=/tmp/ra-go-cache CGO_ENABLED=1 wails3 task package:linux
```

This machine has `CGO_ENABLED=0` in the Go environment. Wails on Linux needs cgo for WebKitGTK, so set `CGO_ENABLED=1` when testing or building the desktop target.

On Linux the packaged AppImage is written under `bin/`. The task file also includes macOS `.app` packaging and Windows NSIS installer packaging.

## Run

Run the built application for the current OS:

```sh
wails3 task run
```

On Linux you can also run the binary directly:

```sh
./bin/ra
```

## Plugin Format

See `docs/plugins.md` for the current local plugin contract.

Built-in plugin source lives in the repository `plugins/` directory. Demo plugin source lives under `examples/`. User-installed plugin packages should live under `~/.local/share/ra/plugins/<plugin-id>.wasm`.

Build a plugin package with:

```sh
GOOS=wasip1 GOARCH=wasm go build -buildvcs=false -buildmode=c-shared -o codec-tools.wasm ./examples/codec-tools
```

Plugin and capability enable/disable state is stored in `~/.config/ra/plugins.json`. The plugin manager can disable built-in plugins such as `ra-app-launcher`, but it only uninstalls user plugin files and refuses to disable or uninstall its own management capability.

## Next Steps

- Add more explicit host APIs for storage and result rendering.
- Add Niri-friendly show/hide integration and document a compositor keybinding.
