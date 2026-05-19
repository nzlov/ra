package desktop

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseEntryKeepsLaunchableVisibleApps(t *testing.T) {
	entry, ok := ParseEntry("org.example.Editor.desktop", []byte(`
[Desktop Entry]
Type=Application
Name=Example Editor
Comment=Edit text files
Exec=example-editor %U --new-window
Icon=accessories-text-editor
Categories=Utility;TextEditor;
`))
	if !ok {
		t.Fatal("expected entry to be launchable")
	}

	if entry.ID != "org.example.Editor" {
		t.Fatalf("ID = %q", entry.ID)
	}
	if entry.Name != "Example Editor" {
		t.Fatalf("Name = %q", entry.Name)
	}
	if entry.Exec != "example-editor %U --new-window" {
		t.Fatalf("Exec = %q", entry.Exec)
	}
	if got := entry.LaunchCommand(); got != "example-editor --new-window" {
		t.Fatalf("LaunchCommand() = %q", got)
	}
}

func TestParseEntrySkipsHiddenOrNonApplications(t *testing.T) {
	cases := map[string]string{
		"hidden": `
[Desktop Entry]
Type=Application
Name=Hidden
NoDisplay=true
Exec=hidden
`,
		"terminal": `
[Desktop Entry]
Type=Application
Name=Terminal Only
Terminal=true
Exec=terminal-only
`,
		"link": `
[Desktop Entry]
Type=Link
Name=Website
URL=https://example.com
`,
	}

	for name, raw := range cases {
		t.Run(name, func(t *testing.T) {
			if _, ok := ParseEntry(name+".desktop", []byte(raw)); ok {
				t.Fatal("expected entry to be skipped")
			}
		})
	}
}

func TestLoadDirsReadsDesktopFiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "app.desktop"), []byte(`
[Desktop Entry]
Type=Application
Name=Alpha Tool
Exec=alpha-tool
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "note.txt"), []byte("ignored"), 0o644); err != nil {
		t.Fatal(err)
	}

	entries, err := LoadDirs([]string{dir})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("len(entries) = %d", len(entries))
	}
	if entries[0].Name != "Alpha Tool" {
		t.Fatalf("Name = %q", entries[0].Name)
	}
}

func TestSearchRanksPrefixBeforeContains(t *testing.T) {
	entries := []Entry{
		{ID: "beta", Name: "Beta Notes", Comment: "Alpha capable"},
		{ID: "alpha", Name: "Alpha Tool", Comment: "Utility"},
	}

	results := Search(entries, "alp", 10)
	if len(results) != 2 {
		t.Fatalf("len(results) = %d", len(results))
	}
	if results[0].ID != "alpha" {
		t.Fatalf("first result ID = %q", results[0].ID)
	}
}
