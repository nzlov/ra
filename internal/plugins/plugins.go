package plugins

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type Registry struct {
	Root    string
	Roots   []string
	Plugins []Plugin
	Errors  []LoadError
}

type Plugin struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Type        string     `json:"type"`
	Entry       string     `json:"entry"`
	Permissions []string   `json:"permissions,omitempty"`
	Commands    []Command  `json:"commands,omitempty"`
	Dir         string     `json:"-"`
	EntryPath   string     `json:"-"`
}

type Command struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Subtitle string `json:"subtitle,omitempty"`
	Export   string `json:"export,omitempty"`
}

type LoadError struct {
	Path  string
	Error string
}

type Result struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Subtitle string `json:"subtitle"`
	Kind     string `json:"kind"`
	Action   Action `json:"action"`
}

type Action struct {
	Type      string `json:"type"`
	PluginID  string `json:"pluginId"`
	CommandID string `json:"commandId"`
	EntryPath string `json:"entryPath"`
	Export    string `json:"export,omitempty"`
}

var validID = regexp.MustCompile(`^[a-z0-9][a-z0-9-_.]*$`)

func LoadRegistry(root string) (Registry, error) {
	return LoadRegistries([]string{root})
}

func LoadRegistries(roots []string) (Registry, error) {
	registry := Registry{Roots: append([]string(nil), roots...)}
	if len(roots) == 1 {
		registry.Root = roots[0]
	}
	seen := map[string]struct{}{}
	for _, root := range roots {
		if root == "" {
			continue
		}
		if _, ok := seen[root]; ok {
			continue
		}
		seen[root] = struct{}{}
		if err := loadRoot(root, &registry); err != nil {
			return registry, err
		}
	}

	sort.SliceStable(registry.Plugins, func(i, j int) bool {
		return registry.Plugins[i].Name < registry.Plugins[j].Name
	})
	return registry, nil
}

func loadRoot(root string, registry *Registry) error {
	items, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, item := range items {
		if !item.IsDir() {
			continue
		}
		pluginDir := filepath.Join(root, item.Name())
		plugin, err := loadPlugin(pluginDir)
		if err != nil {
			registry.Errors = append(registry.Errors, LoadError{Path: pluginDir, Error: err.Error()})
			continue
		}
		registry.Plugins = append(registry.Plugins, plugin)
	}
	return nil
}

func (r Registry) Search(query string, limit int) []Result {
	query = strings.ToLower(strings.TrimSpace(query))
	var results []Result
	for _, plugin := range r.Plugins {
		for _, command := range plugin.Commands {
			text := strings.ToLower(plugin.Name + " " + command.Title + " " + command.Subtitle)
			if query != "" && !strings.Contains(text, query) {
				continue
			}
			actionType := "plugin.open"
			if plugin.Type == "command" {
				actionType = "plugin.run"
			}
			results = append(results, Result{
				ID:       "plugin:" + plugin.ID + ":" + command.ID,
				Title:    command.Title,
				Subtitle: command.Subtitle,
				Kind:     "plugin",
				Action: Action{
					Type:      actionType,
					PluginID:  plugin.ID,
					CommandID: command.ID,
					EntryPath: plugin.EntryPath,
					Export:    command.Export,
				},
			})
		}
	}
	if limit > 0 && len(results) > limit {
		return results[:limit]
	}
	return results
}

func loadPlugin(dir string) (Plugin, error) {
	raw, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
	if err != nil {
		return Plugin{}, err
	}
	var plugin Plugin
	if err := json.Unmarshal(raw, &plugin); err != nil {
		return Plugin{}, err
	}
	if err := validate(plugin); err != nil {
		return Plugin{}, err
	}
	plugin.Dir = dir
	plugin.EntryPath = filepath.Join(dir, plugin.Entry)
	if _, err := os.Stat(plugin.EntryPath); err != nil {
		return Plugin{}, fmt.Errorf("entry %q is not readable: %w", plugin.Entry, err)
	}
	return plugin, nil
}

func validate(plugin Plugin) error {
	if !validID.MatchString(plugin.ID) {
		return fmt.Errorf("invalid plugin id %q", plugin.ID)
	}
	if strings.TrimSpace(plugin.Name) == "" {
		return fmt.Errorf("plugin %q has empty name", plugin.ID)
	}
	if plugin.Type != "webview" && plugin.Type != "command" {
		return fmt.Errorf("plugin %q has unsupported type %q", plugin.ID, plugin.Type)
	}
	if strings.TrimSpace(plugin.Entry) == "" || filepath.IsAbs(plugin.Entry) || strings.Contains(plugin.Entry, "..") {
		return fmt.Errorf("plugin %q has invalid entry %q", plugin.ID, plugin.Entry)
	}
	for _, command := range plugin.Commands {
		if !validID.MatchString(command.ID) {
			return fmt.Errorf("plugin %q has invalid command id %q", plugin.ID, command.ID)
		}
		if strings.TrimSpace(command.Title) == "" {
			return fmt.Errorf("plugin %q command %q has empty title", plugin.ID, command.ID)
		}
		if plugin.Type == "command" && strings.TrimSpace(command.Export) == "" {
			return fmt.Errorf("plugin %q command %q has empty export", plugin.ID, command.ID)
		}
	}
	return nil
}
