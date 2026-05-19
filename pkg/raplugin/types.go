package raplugin

import (
	"encoding/json"
	"strings"
)

type Manifest struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Version      string   `json:"version"`
	Permissions  []string `json:"permissions,omitempty"`
	MinRAVersion string   `json:"minRaVersion,omitempty"`
}

type Capability struct {
	ID       string   `json:"id"`
	Title    string   `json:"title"`
	Icon     string   `json:"icon,omitempty"`
	UI       string   `json:"ui"`
	Keywords []string `json:"keywords,omitempty"`
}

type Plugin struct {
	Manifest     Manifest
	Capabilities []Capability
	Assets       map[string][]byte
	Search       func(SearchRequest) []SearchResult
}

type App struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Comment string `json:"comment,omitempty"`
	Command string `json:"command"`
}

type SearchRequest struct {
	Query string `json:"query"`
	Limit int    `json:"limit,omitempty"`
	Apps  []App  `json:"apps,omitempty"`
}

type SearchResult struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Subtitle string `json:"subtitle,omitempty"`
	Kind     string `json:"kind"`
	Action   Action `json:"action"`
}

type Action struct {
	Type         string `json:"type"`
	AppID        string `json:"appId,omitempty"`
	Command      string `json:"command,omitempty"`
	Text         string `json:"text,omitempty"`
	PluginID     string `json:"pluginId,omitempty"`
	CapabilityID string `json:"capabilityId,omitempty"`
	UI           string `json:"ui,omitempty"`
	Query        string `json:"query,omitempty"`
}

var current = Plugin{}
var manifestJSON []byte
var capabilitiesJSON []byte
var assetsJSON []byte

func Register(plugin Plugin) {
	if plugin.Assets == nil {
		plugin.Assets = map[string][]byte{}
	}
	current = plugin
	manifestJSON = mustJSON(plugin.Manifest)
	capabilitiesJSON = mustJSON(plugin.Capabilities)
	assetsJSON = mustJSON(plugin.Assets)
}

func DefaultSearch(request SearchRequest) []SearchResult {
	trimmed := strings.TrimSpace(request.Query)
	results := make([]SearchResult, 0, len(current.Capabilities))
	for _, capability := range current.Capabilities {
		text := strings.ToLower(current.Manifest.Name + " " + capability.Title + " " + strings.Join(capability.Keywords, " "))
		if trimmed != "" && !Matches(text, trimmed) {
			continue
		}
		results = append(results, SearchResult{
			ID:       "capability:" + current.Manifest.ID + ":" + capability.ID,
			Title:    capability.Title,
			Subtitle: current.Manifest.Name,
			Kind:     "capability",
			Action: Action{
				Type:         "capability.open",
				PluginID:     current.Manifest.ID,
				CapabilityID: capability.ID,
				UI:           capability.UI,
				Query:        trimmed,
			},
		})
	}
	return limitResults(results, request.Limit)
}

func Matches(text string, query string) bool {
	tokens := strings.Fields(strings.ToLower(strings.TrimSpace(query)))
	if len(tokens) == 0 {
		return true
	}
	lowerText := strings.ToLower(text)
	words := strings.Fields(lowerText)
	for _, token := range tokens {
		if strings.Contains(lowerText, token) {
			return true
		}
		for _, word := range words {
			if word != "" && strings.Contains(token, word) {
				return true
			}
		}
	}
	return false
}

func SearchData(input []byte) []byte {
	var request SearchRequest
	if err := json.Unmarshal(input, &request); err != nil {
		return []byte("[]")
	}
	var results []SearchResult
	if current.Search != nil {
		results = current.Search(request)
	} else {
		results = DefaultSearch(request)
	}
	return mustJSON(limitResults(results, request.Limit))
}

func ManifestData() []byte {
	return manifestJSON
}

func CapabilitiesData() []byte {
	return capabilitiesJSON
}

func AssetsData() []byte {
	return assetsJSON
}

func limitResults(results []SearchResult, limit int) []SearchResult {
	if limit > 0 && len(results) > limit {
		return results[:limit]
	}
	return results
}

func mustJSON(value any) []byte {
	raw, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return raw
}
