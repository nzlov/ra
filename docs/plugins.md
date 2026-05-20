# RA Plugin Contract

RA plugins are single Go/WASI `.wasm` files. RA core owns the launcher window, plugin registry, UI hosting, enable/disable state, and controlled host APIs. Plugin code owns capabilities, search behavior, icons, and UI resources.

Built-in plugins are Go source packages under `plugins/`. RA builds them with `GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared` and loads the resulting wasm bytes; generated `.wasm` files are not committed. User-installed plugins live as files under `~/.local/share/ra/plugins/<plugin-id>.wasm`.

## Built-In Plugins

- `ra-app-launcher`: provides the `apps` capability for desktop app search and launch.
- `ra-calculator`: provides the `calculate` capability for `=` calculator queries.
- `ra-plugin-manager`: provides the `manage` capability for plugin install, uninstall, enable, disable, and refresh.

Only `ra-plugin-manager.manage` is protected. Other built-in plugins can be disabled, but built-ins cannot be uninstalled or replaced by user plugin packages with the same ID.

## Go Plugin Source

Plugins use `pkg/raplugin` and register themselves from Go code:

```go
package main

import (
	"embed"

	"github.com/nzlov/ra/pkg/raplugin"
)

//go:embed assets/**
var assets embed.FS

func init() {
	raplugin.Register(raplugin.Plugin{
		Manifest: raplugin.Manifest{
			ID:          "codec-tools",
			Name:        "Codec Tools",
			Version:     "0.1.0",
			Permissions: []string{"clipboard:write"},
		},
		Capabilities: []raplugin.Capability{{
			ID:       "base64",
			Title:    "Base64 Convert",
			Icon:     "/icons/codec.svg",
			UI:       "/base64/index.html",
			Keywords: []string{"base64", "b64"},
		}},
		Assets: raplugin.MustAssets(assets, "assets"),
	})
}

func main() {}
```

Rules:

- Plugin IDs and capability IDs must match `^[a-z0-9][a-z0-9-_.]*$`.
- A plugin may expose multiple capabilities.
- Every capability has a UI asset path.
- Asset paths in `Assets`, `icon`, and `ui` must start with `/`.
- Capability UI assets must be `.html`, live below a capability-specific directory, and not share that UI directory with another capability.
- Permissions are declared in the manifest for user review and are enforced when the capability asks RA to run host actions.

Build a user plugin package with:

```sh
GOOS=wasip1 GOARCH=wasm go build -buildvcs=false -buildmode=c-shared -o codec-tools.wasm ./examples/codec-tools
```

## Search And Launch

RA asks each enabled plugin to search by calling its `Search` function with the query and limit. RA does not implement default capability search and does not push app data into the search request. Each plugin owns its trigger rules, matching, ranking, and result construction.

Plugins call RA host APIs when they need controlled system data. `ra-app-launcher` declares `apps:read`, calls `raplugin.AppsList()` from inside its search function, filters the returned apps itself, and returns `app.launch` results containing only an `appId`. App discovery and launch command resolution stay in RA core.

When a capability matches a query, the plugin returns a `capability.open` action with:

- `capabilityId`
- `query`

RA stamps the result with the plugin ID currently being executed and the UI path from the matching capability manifest, so a plugin cannot claim another plugin's identity or point a result at another capability UI.

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
- `apps:read`: allows `raplugin.AppsList()` from WASM search code.
- `apps:launch`: allows `{type: 'app.launch', appId: string}`. RA ignores plugin-supplied commands and launches the command from its own loaded app list.

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
