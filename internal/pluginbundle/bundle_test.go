package pluginbundle

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
)

func TestBuildAndReadBundle(t *testing.T) {
	manifest := Manifest{
		ID:          "codec-tools",
		Name:        "Codec Tools",
		Version:     "0.1.0",
		Permissions: []string{"clipboard:read", "clipboard:write"},
	}
	capabilities := []Capability{{
		ID:       "base64",
		Title:    "Base64 Convert",
		Icon:     "/icons/base64.svg",
		UI:       "/capabilities/base64/index.html",
		Keywords: []string{"base64", "b64"},
	}}
	assets := map[string][]byte{
		"/capabilities/base64/index.html": []byte("<main>base64</main>"),
		"/icons/base64.svg":               []byte("<svg></svg>"),
	}

	raw, err := Build(manifest, capabilities, assets)
	if err != nil {
		t.Fatal(err)
	}

	bundle, err := Read(raw)
	if err != nil {
		t.Fatal(err)
	}
	if bundle.Manifest.ID != "codec-tools" {
		t.Fatalf("manifest ID = %q", bundle.Manifest.ID)
	}
	if got := bundle.Capabilities[0].ID; got != "base64" {
		t.Fatalf("capability ID = %q", got)
	}
	if string(bundle.Assets["/capabilities/base64/index.html"]) != "<main>base64</main>" {
		t.Fatalf("assets = %#v", bundle.Assets)
	}
	if !json.Valid(bundle.RawManifest) {
		t.Fatalf("RawManifest is not JSON: %q", bundle.RawManifest)
	}
}

func TestBuildWritesAssetsAsPathMap(t *testing.T) {
	raw, err := Build(
		Manifest{ID: "codec-tools", Name: "Codec Tools", Version: "0.1.0"},
		[]Capability{{ID: "base64", Title: "Base64 Convert", UI: "/base64/index.html"}},
		map[string][]byte{"/base64/index.html": []byte("<main>base64</main>")},
	)
	if err != nil {
		t.Fatal(err)
	}

	sections, err := readCustomSections(raw)
	if err != nil {
		t.Fatal(err)
	}
	var assets map[string]string
	if err := json.Unmarshal(sections[assetsSection], &assets); err != nil {
		t.Fatal(err)
	}
	if got := assets["/base64/index.html"]; got != base64.StdEncoding.EncodeToString([]byte("<main>base64</main>")) {
		t.Fatalf("asset payload = %q", got)
	}
}

func TestReadRejectsMissingManifest(t *testing.T) {
	raw, err := BuildRaw(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Read(raw); err == nil {
		t.Fatal("expected missing manifest error")
	}
}

func TestBuildRejectsInvalidAssetPath(t *testing.T) {
	_, err := Build(Manifest{ID: "x", Name: "X", Version: "0.1.0"}, nil, map[string][]byte{
		"../index.html": []byte("bad"),
	})
	if err == nil {
		t.Fatal("expected invalid asset path error")
	}
}

func TestBuildRejectsMissingCapabilityUIAsset(t *testing.T) {
	_, err := Build(Manifest{ID: "codec-tools", Name: "Codec Tools", Version: "0.1.0"}, []Capability{{
		ID:    "base64",
		Title: "Base64 Convert",
		UI:    "/base64/index.html",
	}}, map[string][]byte{
		"/index.html": []byte("<main></main>"),
	})
	if err == nil {
		t.Fatal("expected missing ui asset error")
	}
	if !strings.Contains(err.Error(), "missing ui asset") {
		t.Fatalf("error = %q", err)
	}
}

func TestBuildRejectsNonHTMLCapabilityUI(t *testing.T) {
	_, err := Build(Manifest{ID: "codec-tools", Name: "Codec Tools", Version: "0.1.0"}, []Capability{{
		ID:    "base64",
		Title: "Base64 Convert",
		UI:    "/base64/app.js",
	}}, map[string][]byte{
		"/base64/app.js": []byte("console.log('ui')"),
	})
	if err == nil {
		t.Fatal("expected non-html ui error")
	}
	if !strings.Contains(err.Error(), "ui asset must be .html") {
		t.Fatalf("error = %q", err)
	}
}

func TestBuildRejectsRootCapabilityUI(t *testing.T) {
	_, err := Build(Manifest{ID: "codec-tools", Name: "Codec Tools", Version: "0.1.0"}, []Capability{{
		ID:    "base64",
		Title: "Base64 Convert",
		UI:    "/index.html",
	}}, map[string][]byte{
		"/index.html": []byte("<main></main>"),
	})
	if err == nil {
		t.Fatal("expected root ui error")
	}
	if !strings.Contains(err.Error(), "ui asset must live under a capability directory") {
		t.Fatalf("error = %q", err)
	}
}

func TestBuildRejectsHTMLCapabilityIcon(t *testing.T) {
	_, err := Build(Manifest{ID: "codec-tools", Name: "Codec Tools", Version: "0.1.0"}, []Capability{{
		ID:    "base64",
		Title: "Base64 Convert",
		UI:    "/base64/index.html",
		Icon:  "/base64/icon.html",
	}}, map[string][]byte{
		"/base64/index.html": []byte("<main></main>"),
		"/base64/icon.html":  []byte("<main>icon</main>"),
	})
	if err == nil {
		t.Fatal("expected html icon error")
	}
	if !strings.Contains(err.Error(), "icon asset must not be .html") {
		t.Fatalf("error = %q", err)
	}
}

func TestBuildRejectsDuplicateCapabilityIDs(t *testing.T) {
	_, err := Build(Manifest{ID: "codec-tools", Name: "Codec Tools", Version: "0.1.0"}, []Capability{
		{ID: "base64", Title: "Base64 Convert", UI: "/base64/index.html"},
		{ID: "base64", Title: "Base64 Other", UI: "/other/index.html"},
	}, map[string][]byte{
		"/base64/index.html": []byte("<main></main>"),
		"/other/index.html":  []byte("<main></main>"),
	})
	if err == nil {
		t.Fatal("expected duplicate capability id error")
	}
	if !strings.Contains(err.Error(), "duplicate capability id") {
		t.Fatalf("error = %q", err)
	}
}

func TestBuildRejectsDuplicateCapabilityUIDirectories(t *testing.T) {
	_, err := Build(Manifest{ID: "codec-tools", Name: "Codec Tools", Version: "0.1.0"}, []Capability{
		{ID: "base64", Title: "Base64 Convert", UI: "/tools/base64.html"},
		{ID: "json", Title: "JSON Convert", UI: "/tools/json.html"},
	}, map[string][]byte{
		"/tools/base64.html": []byte("<main></main>"),
		"/tools/json.html":   []byte("<main></main>"),
	})
	if err == nil {
		t.Fatal("expected duplicate ui directory error")
	}
	if !strings.Contains(err.Error(), "duplicate capability ui directory") {
		t.Fatalf("error = %q", err)
	}
}

func TestReadRejectsDuplicateCustomSections(t *testing.T) {
	rawManifest, err := json.Marshal(Manifest{ID: "codec-tools", Name: "Codec Tools", Version: "0.1.0"})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := BuildRaw(map[string][]byte{manifestSection: rawManifest}, [][]byte{encodeCustomSection(manifestSection, rawManifest)})
	if err != nil {
		t.Fatal(err)
	}

	_, err = Read(raw)
	if err == nil {
		t.Fatal("expected duplicate custom section error")
	}
	if !strings.Contains(err.Error(), "duplicate wasm custom section") {
		t.Fatalf("error = %q", err)
	}
}

func TestReadRejectsMissingCapabilityUIAsset(t *testing.T) {
	rawManifest, err := json.Marshal(Manifest{ID: "codec-tools", Name: "Codec Tools", Version: "0.1.0"})
	if err != nil {
		t.Fatal(err)
	}
	rawCapabilities, err := json.Marshal([]Capability{{
		ID:    "base64",
		Title: "Base64 Convert",
		UI:    "/base64/index.html",
	}})
	if err != nil {
		t.Fatal(err)
	}
	rawAssets, err := json.Marshal(map[string][]byte{"/index.html": []byte("<main></main>")})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := BuildRaw(map[string][]byte{
		manifestSection:     rawManifest,
		capabilitiesSection: rawCapabilities,
		assetsSection:       rawAssets,
	}, nil)
	if err != nil {
		t.Fatal(err)
	}

	_, err = Read(raw)
	if err == nil {
		t.Fatal("expected missing ui asset error")
	}
	if !strings.Contains(err.Error(), "missing ui asset") {
		t.Fatalf("error = %q", err)
	}
}
