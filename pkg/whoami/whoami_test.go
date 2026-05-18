package whoami

import (
	"bytes"
	"testing"
)

func TestRunReturnsUser(t *testing.T) {
	result, err := Run()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.User == "" {
		t.Error("expected non-empty username")
	}
	if result.UID < 0 {
		t.Errorf("expected UID >= 0, got %d", result.UID)
	}
}

func TestRunCLI(t *testing.T) {
	var buf bytes.Buffer
	code := run([]string{}, &buf)
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
	if buf.Len() == 0 {
		t.Error("expected output")
	}
}

func TestRunCLIJSON(t *testing.T) {
	var buf bytes.Buffer
	code := run([]string{"--json"}, &buf)
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
	if buf.Len() == 0 {
		t.Error("expected JSON output")
	}
	if !bytes.Contains(buf.Bytes(), []byte("\"user\"")) {
		t.Error("JSON output missing 'user' field")
	}
}

func TestRunCLI_UnknownFlag(t *testing.T) {
	var buf bytes.Buffer
	code := run([]string{"--no-such-flag"}, &buf)
	if code != 2 {
		t.Errorf("expected exit 2 for unknown flag, got %d", code)
	}
}
