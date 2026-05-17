package nohup

import (
	"bytes"
	"io"
	"os"
	"testing"
)

func TestNohupExecute(t *testing.T) {
	dir := t.TempDir()
	// Change to temp dir so nohup.out is created there
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	var buf bytes.Buffer
	code := run([]string{"echo", "hello"}, &buf)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}

	// Check nohup.out was created (since test stdout may not be a tty,
	// this only applies if stdout IS a tty)
	if data, err := os.ReadFile("nohup.out"); err == nil {
		t.Logf("nohup.out contains: %q", string(data))
	}
	os.Remove("nohup.out")
}

func TestNohupExitCode(t *testing.T) {
	var buf bytes.Buffer
	code := run([]string{"false"}, &buf)
	if code != 1 {
		t.Errorf("expected exit 1 from false, got %d", code)
	}
}

func TestNohupMissingCommand(t *testing.T) {
	code := run([]string{}, io.Discard)
	if code != 1 {
		t.Errorf("expected exit 1 for missing command, got %d", code)
	}
}

func TestNohupJson(t *testing.T) {
	var buf bytes.Buffer
	code := run([]string{"--json", "true"}, &buf)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	if !bytes.Contains(buf.Bytes(), []byte(`"command"`)) {
		t.Error("JSON output missing command field")
	}
}
