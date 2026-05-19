package wasmplugin

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunnerCallsExportedI32Function(t *testing.T) {
	path := filepath.Join(t.TempDir(), "answer.wasm")
	if err := os.WriteFile(path, answerWASM(), 0o644); err != nil {
		t.Fatal(err)
	}

	runner := NewRunner()
	result, err := runner.CallI32(t.Context(), path, "answer")
	if err != nil {
		t.Fatal(err)
	}
	if result != 42 {
		t.Fatalf("result = %d", result)
	}
}

func TestRunnerRejectsMissingExport(t *testing.T) {
	path := filepath.Join(t.TempDir(), "answer.wasm")
	if err := os.WriteFile(path, answerWASM(), 0o644); err != nil {
		t.Fatal(err)
	}

	runner := NewRunner()
	if _, err := runner.CallI32(t.Context(), path, "missing"); err == nil {
		t.Fatal("expected missing export error")
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
