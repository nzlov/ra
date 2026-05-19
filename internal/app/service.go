package app

import (
	"os"

	"github.com/nzlov/ra/internal/calculator"
	"github.com/nzlov/ra/internal/desktop"
	"github.com/nzlov/ra/internal/plugins"
)

type Config struct {
	PluginRoot  string
	PluginRoots []string
	DesktopDirs []string
	Limit       int
}

type LauncherService struct {
	config         Config
	desktopEntries []desktop.Entry
	pluginRegistry plugins.Registry
	actions        ActionExecutor
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
	AppID     string `json:"appId,omitempty"`
	Command   string `json:"command,omitempty"`
	Text      string `json:"text,omitempty"`
	PluginID  string `json:"pluginId,omitempty"`
	CommandID string `json:"commandId,omitempty"`
	EntryPath string `json:"entryPath,omitempty"`
	Export    string `json:"export,omitempty"`
}

type InvokeResult struct {
	Type      string `json:"type"`
	Message   string `json:"message"`
	EntryPath string `json:"entryPath,omitempty"`
	URL       string `json:"url,omitempty"`
}

type Status struct {
	AppCount         int      `json:"appCount"`
	PluginCount      int      `json:"pluginCount"`
	PluginErrorCount int      `json:"pluginErrorCount"`
	PluginRoots      []string `json:"pluginRoots"`
}

func NewLauncherService(config Config) *LauncherService {
	if len(config.PluginRoots) == 0 {
		if config.PluginRoot != "" {
			config.PluginRoots = []string{config.PluginRoot}
		} else {
			home, _ := os.UserHomeDir()
			config.PluginRoots = defaultPluginRoots(home)
		}
	}
	if config.PluginRoot == "" && len(config.PluginRoots) > 0 {
		config.PluginRoot = config.PluginRoots[0]
	}
	if config.Limit == 0 {
		config.Limit = 20
	}
	if len(config.DesktopDirs) == 0 {
		home, _ := os.UserHomeDir()
		config.DesktopDirs = desktop.DefaultDirs(home)
	}
	return &LauncherService{config: config, actions: NewActionExecutor()}
}

func NewDefaultLauncherService() *LauncherService {
	service := NewLauncherService(Config{})
	_ = service.Refresh()
	return service
}

func (s *LauncherService) Refresh() error {
	entries, err := desktop.LoadDirs(s.config.DesktopDirs)
	if err != nil {
		return err
	}
	s.desktopEntries = entries
	return s.RefreshPlugins()
}

func (s *LauncherService) RefreshPlugins() error {
	registry, err := plugins.LoadRegistries(s.config.PluginRoots)
	if err != nil {
		return err
	}
	s.pluginRegistry = registry
	return nil
}

func (s *LauncherService) SetDesktopEntries(entries []desktop.Entry) {
	s.desktopEntries = entries
}

func (s *LauncherService) Search(query string) []Result {
	if calc, ok := calculator.Query(query); ok {
		return []Result{{
			ID:       "calculator:result",
			Title:    calc.Title,
			Subtitle: calc.Subtitle,
			Kind:     "calculator",
			Action: Action{
				Type: calc.Action.Type,
				Text: calc.Action.Text,
			},
		}}
	}

	results := make([]Result, 0, s.config.Limit)
	for _, entry := range desktop.Search(s.desktopEntries, query, s.config.Limit) {
		results = append(results, Result{
			ID:       "app:" + entry.ID,
			Title:    entry.Name,
			Subtitle: entry.Comment,
			Kind:     "app",
			Action: Action{
				Type:    "app.launch",
				AppID:   entry.ID,
				Command: entry.LaunchCommand(),
			},
		})
	}

	remaining := s.config.Limit - len(results)
	if remaining <= 0 {
		return results
	}
	for _, result := range s.pluginRegistry.Search(query, remaining) {
		results = append(results, Result{
			ID:       result.ID,
			Title:    result.Title,
			Subtitle: result.Subtitle,
			Kind:     result.Kind,
			Action: Action{
				Type:      result.Action.Type,
				PluginID:  result.Action.PluginID,
				CommandID: result.Action.CommandID,
				EntryPath: result.Action.EntryPath,
				Export:    result.Action.Export,
			},
		})
	}
	return results
}

func (s *LauncherService) Invoke(action Action) (InvokeResult, error) {
	return s.actions.Invoke(action)
}

func (s *LauncherService) Status() Status {
	return Status{
		AppCount:         len(s.desktopEntries),
		PluginCount:      len(s.pluginRegistry.Plugins),
		PluginErrorCount: len(s.pluginRegistry.Errors),
		PluginRoots:      append([]string(nil), s.config.PluginRoots...),
	}
}

func defaultPluginRoots(home string) []string {
	roots := []string{"plugins"}
	if home != "" {
		roots = append(roots, home+"/.local/share/ra/plugins")
	}
	return roots
}
