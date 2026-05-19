# RA Plugin Contract

RA plugins are single `.wasm` package files. RA core owns the launcher window, search integration, plugin registry, and controlled host APIs. Plugin code and UI resources are carried by the plugin package.

Built-in plugins are stored as source directories under `plugins/`. At build/runtime RA embeds those source files and assembles valid WASM plugin bundles in memory; generated `.wasm` files are not committed. User-installed plugins live as files under `~/.local/share/ra/plugins/<plugin-id>.wasm`.

## Built-In Plugins

- `ra-app-launcher`: provides the `apps` capability for desktop app search and launch.
- `ra-calculator`: provides the `calculate` capability for `=` calculator queries.
- `ra-plugin-manager`: provides the `manage` capability for plugin install, uninstall, enable, disable, and refresh.

Only `ra-plugin-manager.manage` is protected. Other built-in plugins can be disabled, but built-ins cannot be uninstalled or replaced by user plugin packages with the same ID.

## Package Sections

Each plugin bundle is a WASM file with RA custom sections:

- `ra.manifest`: JSON object with plugin metadata.
- `ra.capabilities`: JSON array describing capability entries.
- `ra.assets`: JSON object whose keys are absolute asset paths and whose values are base64-encoded bytes.

Manifest example:

```json
{
  "id": "codec-tools",
  "name": "Codec Tools",
  "version": "0.1.0",
  "permissions": ["clipboard:write"]
}
```

Capabilities example:

```json
[
  {
    "id": "base64",
    "title": "Base64 Convert",
    "icon": "/icons/codec.svg",
    "ui": "/base64/index.html",
    "keywords": ["base64", "b64", "encode", "decode"]
  }
]
```

Rules:

- Plugin IDs and capability IDs must match `^[a-z0-9][a-z0-9-_.]*$`.
- A plugin may expose multiple capabilities.
- Every capability has a UI asset path.
- Asset paths in `ra.assets`, `icon`, and `ui` must start with `/`.
- Capability UI assets must be `.html`, live below a capability-specific directory, and not share that UI directory with another capability.
- Permissions are declared in the manifest for user review and are enforced when the capability asks RA to run host actions.

Asset section example:

```json
{
  "/base64/index.html": "PG1haW4+PC9tYWluPg==",
  "/icons/codec.svg": "PHN2Zz48L3N2Zz4="
}
```

## Search And Launch

RA searches capabilities, not legacy command entries. When a capability matches a query, RA returns a `capability.open` action with:

- `pluginId`
- `capabilityId`
- `ui`
- `query`

The launcher can pass the query into the capability UI so a plugin such as `codec-tools` can route `base64 hello` directly into its Base64 interface.

Capability UI assets are served by RA at:

```text
/plugins/<plugin-id>/<capability-id>/<asset-path>
```

For example, `/base64/index.html` in `codec-tools.base64` is opened as `/plugins/codec-tools/base64/base64/index.html?q=base64%20hello`. RA only serves assets for loaded and enabled capabilities.

Capability pages run in a sandboxed iframe without same-origin access to the RA window. The iframe allows scripts only, and plugin asset responses include a restrictive Content Security Policy. Capability pages cannot call Wails bindings directly. RA injects a small `window.ra` bridge into HTML assets:

```js
await window.ra.invoke({type: 'clipboard.write', text: 'copied text'});
```

RA accepts only supported host actions through this bridge, checks that the plugin and capability are still enabled, and then checks the plugin's declared permissions. Current permissions:

- `clipboard:write`: allows `{type: 'clipboard.write', text: string}`.

Use relative URLs from the capability page for packaged assets. RA resolves `/plugins/<plugin-id>/<capability-id>/icons/codec.svg` to the package asset `/icons/codec.svg`, so a page at `/base64/index.html` can load `../icons/codec.svg`. HTML assets are capability-scoped: RA serves the current capability UI and files under that UI directory, but it will not serve another capability's HTML page through the current capability route.

## Plugin Management

The plugin manager uses RA service APIs rather than direct filesystem access.

Supported operations:

- List built-in plugins, user plugins, capabilities, declared permissions, and load failures.
- Enable or disable plugins by writing `~/.config/ra/plugins.json`.
- Enable or disable individual capabilities through `disabledCapabilities`.
- Install from a selected local `.wasm` file into `~/.local/share/ra/plugins/<plugin-id>.wasm`.
- Uninstall user plugins by deleting only that user `.wasm` file.
- Refresh the registry.

Boundaries:

- Remote plugin markets are not supported yet.
- Plugin ID conflicts are rejected; install never overwrites an existing package.
- Built-in plugin IDs are reserved.
- Built-in plugins are not uninstallable.
- `ra-plugin-manager.manage` cannot be disabled.
