package desktop

import (
	"bufio"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Entry struct {
	ID         string
	Name       string
	Comment    string
	Exec       string
	Icon       string
	Categories []string
	Path       string
}

func DefaultDirs(home string) []string {
	dirs := []string{"/usr/share/applications"}
	if home != "" {
		dirs = append([]string{filepath.Join(home, ".local/share/applications")}, dirs...)
	}
	return dirs
}

func LoadDirs(dirs []string) ([]Entry, error) {
	var entries []Entry
	for _, dir := range dirs {
		items, err := os.ReadDir(dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}

		for _, item := range items {
			if item.IsDir() || !strings.HasSuffix(item.Name(), ".desktop") {
				continue
			}
			path := filepath.Join(dir, item.Name())
			raw, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			entry, ok := ParseEntry(item.Name(), raw)
			if !ok {
				continue
			}
			entry.Path = path
			entries = append(entries, entry)
		}
	}

	sort.SliceStable(entries, func(i, j int) bool {
		return strings.ToLower(entries[i].Name) < strings.ToLower(entries[j].Name)
	})
	return entries, nil
}

func ParseEntry(filename string, raw []byte) (Entry, bool) {
	values := map[string]string{}
	inDesktopEntry := false
	scanner := bufio.NewScanner(strings.NewReader(string(raw)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			inDesktopEntry = line == "[Desktop Entry]"
			continue
		}
		if !inDesktopEntry {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		values[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}

	if values["Type"] != "Application" || values["Name"] == "" || values["Exec"] == "" {
		return Entry{}, false
	}
	if isTrue(values["NoDisplay"]) || isTrue(values["Hidden"]) || isTrue(values["Terminal"]) {
		return Entry{}, false
	}

	id := strings.TrimSuffix(filename, ".desktop")
	return Entry{
		ID:         id,
		Name:       values["Name"],
		Comment:    values["Comment"],
		Exec:       values["Exec"],
		Icon:       values["Icon"],
		Categories: splitCategories(values["Categories"]),
	}, true
}

func Search(entries []Entry, query string, limit int) []Entry {
	query = strings.TrimSpace(strings.ToLower(query))
	if query == "" {
		return first(entries, limit)
	}

	type scored struct {
		entry Entry
		score int
	}
	var matches []scored
	for _, entry := range entries {
		score := scoreEntry(entry, query)
		if score == 0 {
			continue
		}
		matches = append(matches, scored{entry: entry, score: score})
	}
	sort.SliceStable(matches, func(i, j int) bool {
		if matches[i].score != matches[j].score {
			return matches[i].score > matches[j].score
		}
		return strings.ToLower(matches[i].entry.Name) < strings.ToLower(matches[j].entry.Name)
	})

	if limit <= 0 || limit > len(matches) {
		limit = len(matches)
	}
	results := make([]Entry, 0, limit)
	for _, match := range matches[:limit] {
		results = append(results, match.entry)
	}
	return results
}

func (e Entry) LaunchCommand() string {
	fields := strings.Fields(e.Exec)
	kept := make([]string, 0, len(fields))
	for _, field := range fields {
		if strings.HasPrefix(field, "%") {
			continue
		}
		kept = append(kept, field)
	}
	return strings.Join(kept, " ")
}

func scoreEntry(entry Entry, query string) int {
	name := strings.ToLower(entry.Name)
	comment := strings.ToLower(entry.Comment)
	id := strings.ToLower(entry.ID)
	switch {
	case strings.HasPrefix(name, query):
		return 100
	case strings.Contains(name, query):
		return 80
	case strings.HasPrefix(id, query):
		return 60
	case strings.Contains(id, query):
		return 40
	case strings.Contains(comment, query):
		return 20
	default:
		return 0
	}
}

func first(entries []Entry, limit int) []Entry {
	if limit <= 0 || limit > len(entries) {
		limit = len(entries)
	}
	return append([]Entry(nil), entries[:limit]...)
}

func splitCategories(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ";")
	categories := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			categories = append(categories, part)
		}
	}
	return categories
}

func isTrue(value string) bool {
	return strings.EqualFold(value, "true")
}
