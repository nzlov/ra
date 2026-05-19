# Codec Tools

This is a source-only Go/WASI plugin example. Build it into one `.wasm` plugin file with:

```sh
GOOS=wasip1 GOARCH=wasm go build -buildvcs=false -buildmode=c-shared -o codec-tools.wasm ./examples/codec-tools
```

The generated `.wasm` file should not be committed.

The example has one plugin with two independent capabilities:

- `base64`
- `json-xml`
