# RA Plugins

This directory contains built-in plugin source packages.

Each directory is a Go/WASI plugin package that calls `raplugin.Register` from Go code. RA builds these packages into temporary `.wasm` files when loading built-ins. Do not commit generated `.wasm` files here.

- `ra-app-launcher`
- `ra-calculator`
- `ra-plugin-manager`

Example plugin source packages live in `../examples/` so demos do not load by default.
