package app

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nzlov/ra/internal/desktop"
)

func TestSearchMergesCalculatorAppsAndCapabilities(t *testing.T) {
	root := t.TempDir()
	user := filepath.Join(root, "user")
	writeCodecPlugin(t, user)

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
	writeCodecPlugin(t, user)

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
	writeCodecPlugin(t, user)

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
	writeCodecPlugin(t, user)

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
	writeCodecPlugin(t, user)

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
	writeCodecPlugin(t, user)

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
	writeCodecPlugin(t, user)

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
	writeCodecPlugin(t, user)

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
	writeCodecPluginNoPermissions(t, user)

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
	writeCodecPlugin(t, user)

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

func TestPluginInvokeRejectsMismatchedActionCapability(t *testing.T) {
	root := t.TempDir()
	user := filepath.Join(root, "user")
	configPath := filepath.Join(root, "config", "plugins.json")
	writeCodecPlugin(t, user)

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

	if _, err := service.SetCapabilityEnabled("codec-tools", "json", false); err != nil {
		t.Fatal(err)
	}

	_, err := service.InvokePluginAction(PluginActionRequest{
		PluginID:     "codec-tools",
		CapabilityID: "base64",
		Action: Action{
			Type:         "clipboard.write",
			Text:         "copied",
			PluginID:     "ra-plugin-manager",
			CapabilityID: "json",
		},
	})
	if err == nil {
		t.Fatal("expected mismatched capability rejection")
	}
}

func TestPluginInvokeLaunchesLoadedAppByIDOnly(t *testing.T) {
	root := t.TempDir()
	user := filepath.Join(root, "user")
	var gotCommand string
	var gotArgs []string
	writeCodecPluginAppLaunch(t, user)

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

	result, err := service.InvokePluginAction(PluginActionRequest{
		PluginID:     "codec-tools",
		CapabilityID: "base64",
		Action:       Action{Type: "app.launch", AppID: "firefox"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Message != "launched" {
		t.Fatalf("result = %#v", result)
	}
	if gotCommand != "sh" || len(gotArgs) != 2 || gotArgs[0] != "-lc" || gotArgs[1] != "firefox" {
		t.Fatalf("run = %q %#v", gotCommand, gotArgs)
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

	result, err := service.InvokePluginAction(PluginActionRequest{
		PluginID:     "ra-app-launcher",
		CapabilityID: "apps",
		Action:       Action{Type: "app.launch", AppID: "firefox"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Message != "launched" {
		t.Fatalf("result = %#v", result)
	}
	if gotCommand != "sh" || len(gotArgs) != 2 || gotArgs[0] != "-lc" || gotArgs[1] != "firefox" {
		t.Fatalf("run = %q %#v", gotCommand, gotArgs)
	}

	if _, err := service.Invoke(Action{Type: "app.launch", AppID: "firefox"}); err == nil {
		t.Fatal("expected direct app launch rejection")
	}
	if _, err := service.Invoke(Action{Type: "app.launch.command", Text: "firefox"}); err == nil {
		t.Fatal("expected direct app launch command rejection")
	}
	if _, err := service.InvokePluginAction(PluginActionRequest{
		PluginID:     "ra-app-launcher",
		CapabilityID: "apps",
		Action:       Action{Type: "app.launch", AppID: "unknown"},
	}); err == nil {
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

	if _, err := service.InvokePluginAction(PluginActionRequest{
		PluginID:     "ra-app-launcher",
		CapabilityID: "apps",
		Action:       Action{Type: "app.launch", AppID: "firefox"},
	}); err == nil {
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
