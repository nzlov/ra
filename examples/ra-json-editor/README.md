# JSON Editor

This is a source-only Go/WASI plugin example. Build it into one `.wasm` plugin file with:

```sh
GOOS=wasip1 GOARCH=wasm go build -buildvcs=false -buildmode=c-shared -o ra-json-editor.wasm ./examples/ra-json-editor
```

The generated `.wasm` file should not be committed.

Search triggers:

- `json`
- `json edit`
- `json format`

If the search query is valid JSON text, RA passes it through to the editor and the page preloads it. Non-JSON keyword queries open an empty editor or the last local draft.
