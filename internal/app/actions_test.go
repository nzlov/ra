package app

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestActionExecutorWritesClipboardThroughConfiguredCommand(t *testing.T) {
	var gotCommand string
	var gotStdin string
	executor := ActionExecutor{
		ClipboardCommand: []string{"wl-copy"},
		RunCommand: func(command string, args []string, stdin string) error {
			gotCommand = command
			gotStdin = stdin
			return nil
		},
	}

	result, err := executor.Invoke(Action{Type: "clipboard.write", Text: "42"})
	if err != nil {
		t.Fatal(err)
	}
	if gotCommand != "wl-copy" {
		t.Fatalf("command = %q", gotCommand)
	}
	if gotStdin != "42" {
		t.Fatalf("stdin = %q", gotStdin)
	}
	if result.Message != "copied" {
		t.Fatalf("message = %q", result.Message)
	}
}

func TestActionExecutorReturnsPluginFileURL(t *testing.T) {
	entry := filepath.Join(t.TempDir(), "index.html")
	executor := ActionExecutor{}

	result, err := executor.Invoke(Action{Type: "plugin.open", EntryPath: entry})
	if err != nil {
		t.Fatal(err)
	}
	if result.Type != "plugin.open" {
		t.Fatalf("type = %q", result.Type)
	}
	if result.URL == "" {
		t.Fatal("expected plugin URL")
	}
	if result.EntryPath != entry {
		t.Fatalf("entry path = %q", result.EntryPath)
	}
}

func TestActionExecutorRunsWASMCommandAndCopiesResult(t *testing.T) {
	entry := filepath.Join(t.TempDir(), "answer.wasm")
	if err := os.WriteFile(entry, answerWASM(), 0o644); err != nil {
		t.Fatal(err)
	}
	var copied string
	executor := ActionExecutor{
		ClipboardCommand: []string{"wl-copy"},
		RunCommand: func(command string, args []string, stdin string) error {
			copied = stdin
			return nil
		},
	}

	result, err := executor.Invoke(Action{Type: "plugin.run", EntryPath: entry, Export: "answer"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Message != "42" {
		t.Fatalf("message = %q", result.Message)
	}
	if copied != "42" {
		t.Fatalf("copied = %q", copied)
	}
}

func answerWASM() []byte {
	return []byte{
		0x00, 0x61, 0x73, 0x6d,
		0x01, 0x00, 0x00, 0x00,
		0x01, 0x05, 0x01, 0x60, 0x00, 0x01, 0x7f,
		0x03, 0x02, 0x01, 0x00,
		0x07, 0x0a, 0x01, 0x06, 0x61, 0x6e, 0x73, 0x77, 0x65, 0x72, 0x00, 0x00,
		0x0a, 0x06, 0x01, 0x04, 0x00, 0x41, 0x2a, 0x0b,
	}
}

func TestActionExecutorRejectsPluginRunWithoutExport(t *testing.T) {
	executor := ActionExecutor{}
	if _, err := executor.Invoke(Action{Type: "plugin.run", EntryPath: "/tmp/missing.wasm"}); err == nil {
		t.Fatal("expected error")
	} else if got := fmt.Sprint(err); got != "missing wasm export" {
		t.Fatalf("error = %q", got)
	}
}
