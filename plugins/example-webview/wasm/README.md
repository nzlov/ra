# Example WASM Module

This directory is reserved for the source of `../plugin.wasm`.

The host plugin shape is intentionally plain:

- `index.html` owns the page UI.
- `plugin.js` loads `plugin.wasm` if it exists.
- `plugin.wasm` may export small functions such as `answer() -> i32`.

Build tooling is not pinned yet because the MVP should run without requiring a Rust, TinyGo, or AssemblyScript toolchain. A later plugin SDK can add one blessed path.
