package builtinplugins

import (
	"embed"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/nzlov/ra/internal/pluginbundle"
	pluginregistry "github.com/nzlov/ra/internal/plugins"
)

//go:embed ra-app-launcher/** ra-calculator/** ra-plugin-manager/**
var files embed.FS

var builtinIDs = []string{"ra-app-launcher", "ra-calculator", "ra-plugin-manager"}

func List() []pluginregistry.BuiltinPlugin {
	items := make([]pluginregistry.BuiltinPlugin, 0, len(builtinIDs))
	for _, id := range builtinIDs {
		raw, err := build(id)
		name := id
		if err != nil {
			raw = nil
			name = id + ": " + err.Error()
		}
		items = append(items, pluginregistry.BuiltinPlugin{Name: name, Raw: raw})
	}
	return items
}

func build(id string) ([]byte, error) {
	var manifest pluginbundle.Manifest
	if err := readJSON(filepath.ToSlash(filepath.Join(id, "manifest.json")), &manifest); err != nil {
		return nil, err
	}
	var capabilities []pluginbundle.Capability
	if err := readJSON(filepath.ToSlash(filepath.Join(id, "capabilities.json")), &capabilities); err != nil {
		return nil, err
	}
	assets := map[string][]byte{}
	if err := collectAssets(id, id, assets); err != nil {
		return nil, err
	}
	return pluginbundle.Build(manifest, capabilities, assets)
}

func readJSON(name string, target any) error {
	raw, err := files.ReadFile(name)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(raw, target); err != nil {
		return fmt.Errorf("read %s: %w", name, err)
	}
	return nil
}

func collectAssets(root string, dir string, assets map[string][]byte) error {
	entries, err := files.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		name := filepath.ToSlash(filepath.Join(dir, entry.Name()))
		if entry.IsDir() {
			if err := collectAssets(root, name, assets); err != nil {
				return err
			}
			continue
		}
		if strings.HasSuffix(name, "/manifest.json") || strings.HasSuffix(name, "/capabilities.json") {
			continue
		}
		raw, err := files.ReadFile(name)
		if err != nil {
			return err
		}
		rel := strings.TrimPrefix(name, root)
		assets[rel] = raw
	}
	return nil
}
