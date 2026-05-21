package raplugin

import "encoding/json"

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
	Match    Match    `json:"match,omitempty"`
}

type Match struct {
	Regex   string `json:"regex,omitempty"`
	Mode    string `json:"mode,omitempty"`
	Pattern string `json:"pattern,omitempty"`
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
}

type SearchRequest struct {
	Query string `json:"query"`
	Limit int    `json:"limit,omitempty"`
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
	Text         string `json:"text,omitempty"`
	CapabilityID string `json:"capabilityId,omitempty"`
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

func SearchData(input []byte) []byte {
	var request SearchRequest
	if err := json.Unmarshal(input, &request); err != nil {
		return []byte("[]")
	}
	var results []SearchResult
	if current.Search != nil {
		results = current.Search(request)
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
