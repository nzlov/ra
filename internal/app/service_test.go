package app

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nzlov/ra/internal/desktop"
	"github.com/nzlov/ra/internal/pluginbundle"
	"github.com/nzlov/ra/internal/plugins"
)

func TestSearchMergesCalculatorAppsAndCapabilities(t *testing.T) {
	root := t.TempDir()
	user := filepath.Join(root, "user")
	writeWASMPlugin(t, user, pluginbundle.Manifest{ID: "codec-tools", Name: "Codec Tools", Version: "0.1.0"}, []pluginbundle.Capability{{
		ID:       "base64",
		Title:    "Base64 Convert",
		UI:       "/base64/index.html",
		Keywords: []string{"base64", "b64"},
	}})

	service := NewLauncherService(Config{
		PluginRoots:    []string{user},
		UserPluginRoot: user,
		BuiltinPlugins: builtinTestPlugins(t),
	})
	service.setDesktopEntries([]desktop.Entry{{ID: "firefox", Name: "Firefox", Exec: "firefox %U"}})
	if err := service.RefreshPlugins(); err != nil {
		t.Fatal(err)
	}

	calc := service.Search("=6*7")
	if len(calc) != 1 || calc[0].Kind != "capability" || calc[0].Title != "Calculator" {
		t.Fatalf("calculator results = %#v", calc)
	}
	if calc[0].Action.PluginID != "ra-calculator" || calc[0].Action.CapabilityID != "calculate" {
		t.Fatalf("calculator Action = %#v", calc[0].Action)
	}
	if calc[0].Action.Type != "capability.open" || calc[0].Action.Query != "=6*7" {
		t.Fatalf("calculator Action = %#v", calc[0].Action)
	}

	apps := service.Search("fire")
	if len(apps) != 1 || apps[0].Kind != "app" || apps[0].Title != "Firefox" {
		t.Fatalf("app results = %#v", apps)
	}
	if apps[0].Action.PluginID != "ra-app-launcher" || apps[0].Action.CapabilityID != "apps" {
		t.Fatalf("app Action = %#v", apps[0].Action)
	}

	capabilities := service.Search("b64 hello")
	if len(capabilities) != 1 || capabilities[0].Kind != "capability" {
		t.Fatalf("capability results = %#v", capabilities)
	}
	if capabilities[0].Action.PluginID != "codec-tools" || capabilities[0].Action.CapabilityID != "base64" {
		t.Fatalf("capability Action = %#v", capabilities[0].Action)
	}
	if capabilities[0].Action.Query != "b64 hello" {
		t.Fatalf("Query = %q", capabilities[0].Action.Query)
	}
}

func TestRefreshPluginsLoadsBuiltinAndUserWASMPlugins(t *testing.T) {
	root := t.TempDir()
	user := filepath.Join(root, "user")
	writeWASMPlugin(t, user, pluginbundle.Manifest{ID: "codec-tools", Name: "Codec Tools", Version: "0.1.0"}, nil)

	service := NewLauncherService(Config{
		PluginRoots:    []string{user},
		UserPluginRoot: user,
		BuiltinPlugins: builtinTestPlugins(t),
	})
	if err := service.RefreshPlugins(); err != nil {
		t.Fatal(err)
	}
	status := service.Status()
	if status.PluginCount != 4 {
		t.Fatalf("PluginCount = %d", status.PluginCount)
	}
	if status.PluginErrorCount != 0 {
		t.Fatalf("PluginErrorCount = %d", status.PluginErrorCount)
	}
}

func TestDisablingBuiltinAppLauncherRemovesAppSearchResults(t *testing.T) {
	root := t.TempDir()
	user := filepath.Join(root, "user")
	configPath := filepath.Join(root, "config", "plugins.json")

	service := NewLauncherService(Config{
		PluginRoots:      []string{user},
		UserPluginRoot:   user,
		PluginConfigPath: configPath,
		BuiltinPlugins:   builtinTestPlugins(t),
	})
	service.setDesktopEntries([]desktop.Entry{{ID: "firefox", Name: "Firefox", Exec: "firefox %U"}})
	if err := service.RefreshPlugins(); err != nil {
		t.Fatal(err)
	}

	state, err := service.SetPluginEnabled("ra-app-launcher", false)
	if err != nil {
		t.Fatal(err)
	}
	appLauncher := findManagedPlugin(t, state, "ra-app-launcher")
	if appLauncher.Enabled {
		t.Fatalf("ra-app-launcher enabled = true")
	}
	if appLauncher.Protected {
		t.Fatalf("ra-app-launcher protected = true")
	}
	if appLauncher.Uninstallable {
		t.Fatalf("ra-app-launcher uninstallable = true")
	}

	results := service.Search("fire")
	if len(results) != 0 {
		t.Fatalf("disabled app launcher results = %#v", results)
	}
}

func TestDisablingBuiltinCalculatorRemovesCalculatorSearchResults(t *testing.T) {
	root := t.TempDir()
	user := filepath.Join(root, "user")
	configPath := filepath.Join(root, "config", "plugins.json")

	service := NewLauncherService(Config{
		PluginRoots:      []string{user},
		UserPluginRoot:   user,
		PluginConfigPath: configPath,
		BuiltinPlugins:   builtinTestPlugins(t),
	})
	if err := service.RefreshPlugins(); err != nil {
		t.Fatal(err)
	}

	state, err := service.SetPluginEnabled("ra-calculator", false)
	if err != nil {
		t.Fatal(err)
	}
	calculator := findManagedPlugin(t, state, "ra-calculator")
	if calculator.Enabled {
		t.Fatalf("ra-calculator enabled = true")
	}
	if calculator.Protected {
		t.Fatalf("ra-calculator protected = true")
	}

	results := service.Search("calculator")
	if len(results) != 0 {
		t.Fatalf("disabled calculator results = %#v", results)
	}
}

func TestDisablingCapabilityRemovesCapabilitySearchResults(t *testing.T) {
	root := t.TempDir()
	user := filepath.Join(root, "user")
	configPath := filepath.Join(root, "config", "plugins.json")
	writeWASMPlugin(t, user, pluginbundle.Manifest{ID: "codec-tools", Name: "Codec Tools", Version: "0.1.0"}, []pluginbundle.Capability{{
		ID:       "base64",
		Title:    "Base64 Convert",
		UI:       "/base64/index.html",
		Keywords: []string{"base64"},
	}})

	service := NewLauncherService(Config{
		PluginRoots:      []string{user},
		UserPluginRoot:   user,
		PluginConfigPath: configPath,
		BuiltinPlugins:   builtinTestPlugins(t),
	})
	if err := service.RefreshPlugins(); err != nil {
		t.Fatal(err)
	}

	if _, err := service.SetCapabilityEnabled("codec-tools", "base64", false); err != nil {
		t.Fatal(err)
	}
	results := service.Search("base64")
	if len(results) != 0 {
		t.Fatalf("disabled capability results = %#v", results)
	}
	state := service.PluginManagerState()
	capability := findManagedCapability(t, findManagedPlugin(t, state, "codec-tools"), "base64")
	if capability.Enabled {
		t.Fatalf("base64 enabled = true")
	}
}

func TestServeHTTPReturnsEnabledCapabilityAsset(t *testing.T) {
	root := t.TempDir()
	user := filepath.Join(root, "user")
	writeWASMPlugin(t, user, pluginbundle.Manifest{ID: "codec-tools", Name: "Codec Tools", Version: "0.1.0"}, []pluginbundle.Capability{{
		ID:       "base64",
		Title:    "Base64 Convert",
		UI:       "/base64/index.html",
		Keywords: []string{"base64"},
	}})

	service := NewLauncherService(Config{
		PluginRoots:    []string{user},
		UserPluginRoot: user,
		BuiltinPlugins: builtinTestPlugins(t),
	})
	if err := service.RefreshPlugins(); err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/codec-tools/base64/base64/index.html", nil)
	service.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d body = %q", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), "<main>base64</main>") {
		t.Fatalf("body = %q", recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), "window.ra") {
		t.Fatalf("missing plugin bridge: %q", recorder.Body.String())
	}
	if got := recorder.Header().Get("Content-Security-Policy"); !strings.Contains(got, "connect-src 'none'") {
		t.Fatalf("Content-Security-Policy = %q", got)
	}
	if got := recorder.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("X-Content-Type-Options = %q", got)
	}
}

func TestServeHTTPAcceptsRoutePrefixedCapabilityAsset(t *testing.T) {
	root := t.TempDir()
	user := filepath.Join(root, "user")
	writeWASMPlugin(t, user, pluginbundle.Manifest{ID: "codec-tools", Name: "Codec Tools", Version: "0.1.0"}, []pluginbundle.Capability{{
		ID:    "base64",
		Title: "Base64 Convert",
		UI:    "/base64/index.html",
	}})

	service := NewLauncherService(Config{
		PluginRoots:    []string{user},
		UserPluginRoot: user,
		BuiltinPlugins: builtinTestPlugins(t),
	})
	if err := service.RefreshPlugins(); err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/plugins/codec-tools/base64/base64/index.html", nil)
	service.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d body = %q", recorder.Code, recorder.Body.String())
	}
}

func TestServeHTTPReturnsPluginSharedAssetForEnabledCapability(t *testing.T) {
	root := t.TempDir()
	user := filepath.Join(root, "user")
	raw := mustBundleWithAssets(t,
		pluginbundle.Manifest{ID: "codec-tools", Name: "Codec Tools", Version: "0.1.0"},
		[]pluginbundle.Capability{{
			ID:    "base64",
			Title: "Base64 Convert",
			UI:    "/base64/index.html",
			Icon:  "/icons/base64.svg",
		}},
		map[string][]byte{
			"/base64/index.html": []byte("<main><img src=\"../icons/base64.svg\"></main>"),
			"/icons/base64.svg":  []byte("<svg></svg>"),
		},
	)
	writeRawWASMPlugin(t, user, "codec-tools", raw)

	service := NewLauncherService(Config{
		PluginRoots:    []string{user},
		UserPluginRoot: user,
		BuiltinPlugins: builtinTestPlugins(t),
	})
	if err := service.RefreshPlugins(); err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/codec-tools/base64/icons/base64.svg", nil)
	service.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d body = %q", recorder.Code, recorder.Body.String())
	}
	if recorder.Body.String() != "<svg></svg>" {
		t.Fatalf("body = %q", recorder.Body.String())
	}
}

func TestServeHTTPRejectsDisabledCapabilityAsset(t *testing.T) {
	root := t.TempDir()
	user := filepath.Join(root, "user")
	configPath := filepath.Join(root, "config", "plugins.json")
	writeWASMPlugin(t, user, pluginbundle.Manifest{ID: "codec-tools", Name: "Codec Tools", Version: "0.1.0"}, []pluginbundle.Capability{{
		ID:       "base64",
		Title:    "Base64 Convert",
		UI:       "/base64/index.html",
		Keywords: []string{"base64"},
	}})

	service := NewLauncherService(Config{
		PluginRoots:      []string{user},
		UserPluginRoot:   user,
		PluginConfigPath: configPath,
		BuiltinPlugins:   builtinTestPlugins(t),
	})
	if err := service.RefreshPlugins(); err != nil {
		t.Fatal(err)
	}
	if _, err := service.SetCapabilityEnabled("codec-tools", "base64", false); err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/codec-tools/base64/base64/index.html", nil)
	service.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d body = %q", recorder.Code, recorder.Body.String())
	}
}

func TestServeHTTPRejectsAnotherCapabilityUIAsset(t *testing.T) {
	root := t.TempDir()
	user := filepath.Join(root, "user")
	configPath := filepath.Join(root, "config", "plugins.json")
	raw := mustBundleWithAssets(t,
		pluginbundle.Manifest{ID: "codec-tools", Name: "Codec Tools", Version: "0.1.0"},
		[]pluginbundle.Capability{
			{ID: "base64", Title: "Base64 Convert", UI: "/base64/index.html"},
			{ID: "json", Title: "JSON Convert", UI: "/json/index.html"},
		},
		map[string][]byte{
			"/base64/index.html": []byte("<main>base64</main>"),
			"/json/index.html":   []byte("<main>json</main>"),
		},
	)
	writeRawWASMPlugin(t, user, "codec-tools", raw)

	service := NewLauncherService(Config{
		PluginRoots:      []string{user},
		UserPluginRoot:   user,
		PluginConfigPath: configPath,
		BuiltinPlugins:   builtinTestPlugins(t),
	})
	if err := service.RefreshPlugins(); err != nil {
		t.Fatal(err)
	}
	if _, err := service.SetCapabilityEnabled("codec-tools", "json", false); err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/codec-tools/base64/json/index.html", nil)
	service.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d body = %q", recorder.Code, recorder.Body.String())
	}
}

func TestPluginInvokeRequiresDeclaredPermission(t *testing.T) {
	root := t.TempDir()
	user := filepath.Join(root, "user")
	configPath := filepath.Join(root, "config", "plugins.json")
	writeWASMPlugin(t, user,
		pluginbundle.Manifest{ID: "codec-tools", Name: "Codec Tools", Version: "0.1.0"},
		[]pluginbundle.Capability{{
			ID:    "base64",
			Title: "Base64 Convert",
			UI:    "/base64/index.html",
		}},
	)

	service := NewLauncherService(Config{
		PluginRoots:      []string{user},
		UserPluginRoot:   user,
		PluginConfigPath: configPath,
		BuiltinPlugins:   builtinTestPlugins(t),
	})
	service.actions = ActionExecutor{
		ClipboardCommand: []string{"copy"},
		RunCommand: func(command string, args []string, stdin string) error {
			t.Fatalf("unexpected host command %s", command)
			return nil
		},
	}
	if err := service.RefreshPlugins(); err != nil {
		t.Fatal(err)
	}

	_, err := service.InvokePluginAction(PluginActionRequest{
		PluginID:     "codec-tools",
		CapabilityID: "base64",
		Action:       Action{Type: "clipboard.write", Text: "secret"},
	})
	if err == nil {
		t.Fatal("expected permission error")
	}
	if got := err.Error(); got != `plugin "codec-tools" does not declare permission "clipboard:write"` {
		t.Fatalf("error = %q", got)
	}
}

func TestPluginInvokeAllowsDeclaredPermissionForEnabledCapability(t *testing.T) {
	root := t.TempDir()
	user := filepath.Join(root, "user")
	var gotStdin string
	writeWASMPlugin(t, user,
		pluginbundle.Manifest{
			ID:          "codec-tools",
			Name:        "Codec Tools",
			Version:     "0.1.0",
			Permissions: []string{"clipboard:write"},
		},
		[]pluginbundle.Capability{{
			ID:    "base64",
			Title: "Base64 Convert",
			UI:    "/base64/index.html",
		}},
	)

	service := NewLauncherService(Config{
		PluginRoots:    []string{user},
		UserPluginRoot: user,
		BuiltinPlugins: builtinTestPlugins(t),
	})
	service.actions = ActionExecutor{
		ClipboardCommand: []string{"copy"},
		RunCommand: func(command string, args []string, stdin string) error {
			gotStdin = stdin
			return nil
		},
	}
	if err := service.RefreshPlugins(); err != nil {
		t.Fatal(err)
	}

	result, err := service.InvokePluginAction(PluginActionRequest{
		PluginID:     "codec-tools",
		CapabilityID: "base64",
		Action:       Action{Type: "clipboard.write", Text: "copied"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Message != "copied" {
		t.Fatalf("result = %#v", result)
	}
	if gotStdin != "copied" {
		t.Fatalf("stdin = %q", gotStdin)
	}
}

func TestPluginInvokeRejectsAppLaunchCommand(t *testing.T) {
	root := t.TempDir()
	user := filepath.Join(root, "user")
	writeWASMPlugin(t, user,
		pluginbundle.Manifest{
			ID:          "codec-tools",
			Name:        "Codec Tools",
			Version:     "0.1.0",
			Permissions: []string{"apps:launch"},
		},
		[]pluginbundle.Capability{{
			ID:    "base64",
			Title: "Base64 Convert",
			UI:    "/base64/index.html",
		}},
	)

	service := NewLauncherService(Config{
		PluginRoots:    []string{user},
		UserPluginRoot: user,
		BuiltinPlugins: builtinTestPlugins(t),
	})
	service.actions = ActionExecutor{
		RunCommand: func(command string, args []string, stdin string) error {
			t.Fatalf("unexpected command execution: %s %#v", command, args)
			return nil
		},
	}
	if err := service.RefreshPlugins(); err != nil {
		t.Fatal(err)
	}

	_, err := service.InvokePluginAction(PluginActionRequest{
		PluginID:     "codec-tools",
		CapabilityID: "base64",
		Action:       Action{Type: "app.launch", Command: "sh -lc 'touch /tmp/ra-owned'"},
	})
	if err == nil {
		t.Fatal("expected app.launch bridge rejection")
	}
	if got := err.Error(); got != `plugin action "app.launch" is not supported` {
		t.Fatalf("error = %q", got)
	}
}

func TestInvokeLaunchesLoadedDesktopEntryOnly(t *testing.T) {
	root := t.TempDir()
	user := filepath.Join(root, "user")
	var gotCommand string
	var gotArgs []string
	service := NewLauncherService(Config{
		PluginRoots:    []string{user},
		UserPluginRoot: user,
		BuiltinPlugins: builtinTestPlugins(t),
	})
	service.actions = ActionExecutor{
		RunCommand: func(command string, args []string, stdin string) error {
			gotCommand = command
			gotArgs = append([]string(nil), args...)
			return nil
		},
	}
	service.setDesktopEntries([]desktop.Entry{{ID: "firefox", Name: "Firefox", Exec: "firefox %U"}})
	if err := service.RefreshPlugins(); err != nil {
		t.Fatal(err)
	}

	result, err := service.Invoke(Action{Type: "app.launch", AppID: "firefox", Command: "firefox"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Message != "launched" {
		t.Fatalf("result = %#v", result)
	}
	if gotCommand != "sh" || len(gotArgs) != 2 || gotArgs[0] != "-lc" || gotArgs[1] != "firefox" {
		t.Fatalf("run = %q %#v", gotCommand, gotArgs)
	}

	if _, err := service.Invoke(Action{Type: "app.launch", AppID: "firefox", Command: "sh -lc 'touch /tmp/ra-owned'"}); err == nil {
		t.Fatal("expected mismatched command rejection")
	}
	if _, err := service.Invoke(Action{Type: "app.launch", AppID: "unknown", Command: "firefox"}); err == nil {
		t.Fatal("expected unknown app rejection")
	}
	if _, err := service.Invoke(Action{Type: "clipboard.write", Text: "secret"}); err == nil {
		t.Fatal("expected direct clipboard rejection")
	}
}

func TestInvokeRejectsAppLaunchWhenAppLauncherDisabled(t *testing.T) {
	root := t.TempDir()
	user := filepath.Join(root, "user")
	configPath := filepath.Join(root, "config", "plugins.json")
	service := NewLauncherService(Config{
		PluginRoots:      []string{user},
		UserPluginRoot:   user,
		PluginConfigPath: configPath,
		BuiltinPlugins:   builtinTestPlugins(t),
	})
	service.actions = ActionExecutor{
		RunCommand: func(command string, args []string, stdin string) error {
			t.Fatalf("unexpected command execution: %s %#v", command, args)
			return nil
		},
	}
	service.setDesktopEntries([]desktop.Entry{{ID: "firefox", Name: "Firefox", Exec: "firefox %U"}})
	if err := service.RefreshPlugins(); err != nil {
		t.Fatal(err)
	}
	if _, err := service.SetPluginEnabled("ra-app-launcher", false); err != nil {
		t.Fatal(err)
	}

	if _, err := service.Invoke(Action{Type: "app.launch", AppID: "firefox", Command: "firefox"}); err == nil {
		t.Fatal("expected disabled app launcher rejection")
	}
}

func TestBuiltinPluginsLoadFromEmbeddedBundles(t *testing.T) {
	service := NewLauncherService(Config{BuiltinPlugins: builtinTestPlugins(t)})
	if err := service.RefreshPlugins(); err != nil {
		t.Fatal(err)
	}

	state := service.PluginManagerState()
	appLauncher := findManagedPlugin(t, state, "ra-app-launcher")
	if appLauncher.Source != "builtin" || appLauncher.Type != "wasm" || appLauncher.Protected {
		t.Fatalf("app launcher = %#v", appLauncher)
	}
	if appLauncher.Path == "" {
		t.Fatalf("app launcher paths = %#v", appLauncher)
	}
	if got := findManagedCapability(t, appLauncher, "apps"); got.UI != "/apps/index.html" {
		t.Fatalf("app capability = %#v", got)
	}

	manager := findManagedPlugin(t, state, "ra-plugin-manager")
	if manager.Source != "builtin" || manager.Type != "wasm" || !manager.Protected {
		t.Fatalf("manager = %#v", manager)
	}
	if got := findManagedCapability(t, manager, "manage"); got.UI != "/manager/index.html" {
		t.Fatalf("manager capability = %#v", got)
	}

	calculator := findManagedPlugin(t, state, "ra-calculator")
	if calculator.Source != "builtin" || calculator.Type != "wasm" || calculator.Protected {
		t.Fatalf("calculator = %#v", calculator)
	}
	if got := findManagedCapability(t, calculator, "calculate"); got.UI != "/calculator/index.html" {
		t.Fatalf("calculator capability = %#v", got)
	}
}

func builtinTestPlugins(t *testing.T) []plugins.BuiltinPlugin {
	t.Helper()
	return []plugins.BuiltinPlugin{
		{
			Name: "ra-calculator",
			Raw: mustBundle(t, pluginbundle.Manifest{
				ID:          "ra-calculator",
				Name:        "RA Calculator",
				Version:     "0.1.0",
				Permissions: []string{"clipboard:write"},
			}, []pluginbundle.Capability{{
				ID:       "calculate",
				Title:    "Calculator",
				UI:       "/calculator/index.html",
				Keywords: []string{"=", "calculator", "calc", "math"},
			}}),
		},
		{
			Name: "ra-app-launcher",
			Raw: mustBundle(t, pluginbundle.Manifest{
				ID:          "ra-app-launcher",
				Name:        "RA App Launcher",
				Version:     "0.1.0",
				Permissions: []string{"apps:read", "apps:launch"},
			}, []pluginbundle.Capability{{
				ID:       "apps",
				Title:    "Applications",
				UI:       "/apps/index.html",
				Keywords: []string{"app", "apps", "fire"},
			}}),
		},
		{
			Name: "ra-plugin-manager",
			Raw: mustBundle(t, pluginbundle.Manifest{
				ID:          "ra-plugin-manager",
				Name:        "RA Plugin Manager",
				Version:     "0.1.0",
				Permissions: []string{"plugins:manage"},
			}, []pluginbundle.Capability{{
				ID:       "manage",
				Title:    "Plugin Manager",
				UI:       "/manager/index.html",
				Keywords: []string{"plugin", "manager"},
			}}),
		},
	}
}

func writeWASMPlugin(t *testing.T, root string, manifest pluginbundle.Manifest, capabilities []pluginbundle.Capability) string {
	t.Helper()
	raw := mustBundle(t, manifest, capabilities)
	return writeRawWASMPlugin(t, root, manifest.ID, raw)
}

func writeRawWASMPlugin(t *testing.T, root string, id string, raw []byte) string {
	t.Helper()
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, id+".wasm")
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func mustBundle(t *testing.T, manifest pluginbundle.Manifest, capabilities []pluginbundle.Capability) []byte {
	t.Helper()
	raw := mustBundleWithAssets(t, manifest, capabilities, bundleAssets(capabilities))
	return raw
}

func mustBundleWithAssets(t *testing.T, manifest pluginbundle.Manifest, capabilities []pluginbundle.Capability, assets map[string][]byte) []byte {
	t.Helper()
	raw, err := pluginbundle.Build(manifest, capabilities, assets)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func bundleAssets(capabilities []pluginbundle.Capability) map[string][]byte {
	assets := map[string][]byte{
		"/index.html": []byte("<main></main>"),
	}
	for _, capability := range capabilities {
		assets[capability.UI] = []byte("<main>" + capability.ID + "</main>")
		if capability.Icon != "" {
			assets[capability.Icon] = []byte("<svg></svg>")
		}
	}
	return assets
}
