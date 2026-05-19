package app

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/nzlov/ra/internal/wasmplugin"
)

type ActionExecutor struct {
	ClipboardCommand []string
	RunCommand       func(command string, args []string, stdin string) error
}

func NewActionExecutor() ActionExecutor {
	return ActionExecutor{ClipboardCommand: defaultClipboardCommand()}
}

func (e ActionExecutor) Invoke(action Action) (InvokeResult, error) {
	switch action.Type {
	case "app.launch":
		if action.Command == "" {
			return InvokeResult{}, errors.New("missing launch command")
		}
		if err := e.run("sh", []string{"-lc", action.Command}, ""); err != nil {
			return InvokeResult{}, err
		}
		return InvokeResult{Type: action.Type, Message: "launched"}, nil
	case "clipboard.write":
		if len(e.ClipboardCommand) == 0 {
			return InvokeResult{}, errors.New("no clipboard command configured")
		}
		if err := e.run(e.ClipboardCommand[0], e.ClipboardCommand[1:], action.Text); err != nil {
			return InvokeResult{}, err
		}
		return InvokeResult{Type: action.Type, Message: "copied"}, nil
	case "plugin.open":
		if action.EntryPath == "" {
			return InvokeResult{}, errors.New("missing plugin entry path")
		}
		return InvokeResult{
			Type:      action.Type,
			Message:   filepath.Base(action.EntryPath),
			EntryPath: action.EntryPath,
			URL:       fileURL(action.EntryPath),
		}, nil
	case "plugin.run":
		if action.EntryPath == "" {
			return InvokeResult{}, errors.New("missing plugin entry path")
		}
		if action.Export == "" {
			return InvokeResult{}, errors.New("missing wasm export")
		}
		value, err := wasmplugin.NewRunner().CallI32(context.Background(), action.EntryPath, action.Export)
		if err != nil {
			return InvokeResult{}, err
		}
		message := fmt.Sprint(value)
		if len(e.ClipboardCommand) > 0 {
			if err := e.run(e.ClipboardCommand[0], e.ClipboardCommand[1:], message); err != nil {
				return InvokeResult{}, err
			}
		}
		return InvokeResult{Type: action.Type, Message: message, EntryPath: action.EntryPath}, nil
	default:
		return InvokeResult{}, errors.New("unsupported action type")
	}
}

func (e ActionExecutor) run(command string, args []string, stdin string) error {
	if e.RunCommand != nil {
		return e.RunCommand(command, args, stdin)
	}
	cmd := exec.Command(command, args...)
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}
	return cmd.Start()
}

func defaultClipboardCommand() []string {
	if runtime.GOOS == "linux" {
		if _, err := exec.LookPath("wl-copy"); err == nil {
			return []string{"wl-copy"}
		}
		if _, err := exec.LookPath("xclip"); err == nil {
			return []string{"xclip", "-selection", "clipboard"}
		}
	}
	return nil
}

func fileURL(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}
	return (&url.URL{Scheme: "file", Path: filepath.ToSlash(abs)}).String()
}
