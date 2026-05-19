# Answer Command Plugin

This is the smallest command-style WASM plugin RA can execute today.

`answer.wasm` exports one no-argument function:

```wat
(module
  (func (export "answer") (result i32)
    i32.const 42))
```

Selecting `Run Answer WASM` runs the `answer` export and copies the integer result when a supported clipboard command is available.
