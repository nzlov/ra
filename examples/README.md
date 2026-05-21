# RA Plugin Examples

This directory contains demo plugin source packages.

Build one into a single Go/WASI `.wasm` plugin package and install it through the plugin manager when you want to test it. Generated `.wasm` packages should not be committed. Demos are kept outside `../plugins/` so they do not load by default.

- `codec-tools`: basic example with text transforms and capability routing.
- `ra-json-editor`: JSON editor example with text editing, format/minify, validation, and a read-only structure view.
