package common

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// redirectStdout temporarily replaces os.Stdout with a pipe and returns the
// captured output after calling fn.
func redirectStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stdout
	os.Stdout = w

	fn()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

func TestRenderJSONStructure(t *testing.T) {
	type payload struct {
		Text string `json:"text"`
	}
	out := redirectStdout(t, func() {
		Render("echo", payload{Text: "hello"}, true, os.Stdout, func() {})
	})

	var env JSONEnvelope
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &env); err != nil {
		t.Fatalf("invalid JSON: %v\nraw: %s", err, out)
	}
	if env.Command != "echo" {
		t.Errorf("command: got %q, want %q", env.Command, "echo")
	}
	if env.ExitCode != 0 {
		t.Errorf("exitCode: got %d, want 0", env.ExitCode)
	}
	if env.Error != nil {
		t.Errorf("error: expected nil, got %+v", env.Error)
	}
	if env.Data == nil {
		t.Error("data: expected non-nil")
	}
	if env.Version == "" {
		t.Error("version: expected non-empty")
	}
}

func TestRenderJSONAllSixKeys(t *testing.T) {
	out := redirectStdout(t, func() {
		Render("ls", nil, true, os.Stdout, func() {})
	})

	// Check raw JSON has all six keys present.
	for _, key := range []string{"command", "version", "schemaVersion", "exitCode", "data", "error"} {
		if !strings.Contains(out, `"`+key+`"`) {
			t.Errorf("JSON missing key %q in output: %s", key, out)
		}
	}
}

func TestRenderTextMode(t *testing.T) {
	called := false
	out := redirectStdout(t, func() {
		Render("echo", nil, false, os.Stdout, func() { called = true })
	})
	if !called {
		t.Error("textFn not called in text mode")
	}
	if strings.TrimSpace(out) != "" {
		t.Errorf("expected no JSON output in text mode, got: %s", out)
	}
}

func TestRenderErrorJSON(t *testing.T) {
	out := redirectStdout(t, func() {
		RenderError("ls", 2, "ENOENT", "no such file or directory: /nope", true, os.Stdout)
	})

	var env JSONEnvelope
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &env); err != nil {
		t.Fatalf("invalid JSON: %v\nraw: %s", err, out)
	}
	if env.ExitCode != 2 {
		t.Errorf("exitCode: got %d, want 2", env.ExitCode)
	}
	if env.Data != nil {
		t.Errorf("data: expected nil on error, got %+v", env.Data)
	}
	if env.Error == nil {
		t.Fatal("error: expected non-nil")
	}
	if env.Error.Code != "ENOENT" {
		t.Errorf("error.code: got %q, want ENOENT", env.Error.Code)
	}
}

func TestRenderErrorTextModeNoop(t *testing.T) {
	out := redirectStdout(t, func() {
		RenderError("ls", 2, "ENOENT", "oops", false, os.Stdout)
	})
	if strings.TrimSpace(out) != "" {
		t.Errorf("expected no output in text mode, got: %s", out)
	}
}
