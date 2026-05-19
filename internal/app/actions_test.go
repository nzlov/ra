package app

import (
	"fmt"
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

func TestActionExecutorReturnsCapabilityOpenResult(t *testing.T) {
	executor := ActionExecutor{}

	result, err := executor.Invoke(Action{
		Type:         "capability.open",
		PluginID:     "codec-tools",
		CapabilityID: "base64",
		UI:           "/base64/index.html",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Type != "capability.open" {
		t.Fatalf("type = %q", result.Type)
	}
	if result.Message != "codec-tools.base64" {
		t.Fatalf("message = %q", result.Message)
	}
}

func TestActionExecutorRejectsCapabilityOpenWithoutIDs(t *testing.T) {
	executor := ActionExecutor{}
	if _, err := executor.Invoke(Action{Type: "capability.open", PluginID: "codec-tools"}); err == nil {
		t.Fatal("expected error")
	} else if got := fmt.Sprint(err); got != "missing capability id" {
		t.Fatalf("error = %q", got)
	}
}
