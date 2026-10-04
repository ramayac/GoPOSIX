package du

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDuRun(t *testing.T) {
	var out bytes.Buffer
	rc := run([]string{"."}, nil, &out, &out, "")
	if rc != 0 {
		t.Errorf("expected 0, got %d", rc)
	}
	if !strings.Contains(out.String(), ".") {
		t.Error("expected output with .")
	}
}

func TestDuJSON(t *testing.T) {
	var out bytes.Buffer
	rc := run([]string{"--json", "."}, nil, &out, &out, "")
	if rc != 0 {
		t.Errorf("expected 0, got %d", rc)
	}
	if !strings.Contains(out.String(), "command") {
		t.Errorf("expected JSON, got %s", out.String())
	}
}

// --- BusyBox test suite hardening ---

func TestBusyBox_Du_HumanReadable(t *testing.T) {
	// du -h should output human-readable sizes (e.g., "1.0M\tfile")
	// We test that the function produces output with a unit suffix.
	if s := humanSize(1024 * 1024); s != "1.0M" {
		t.Errorf("humanSize(1M) = %q, want %q", s, "1.0M")
	}
	if s := humanSize(1536 * 1024); s != "1.5M" {
		t.Errorf("humanSize(1.5M) = %q, want %q", s, "1.5M")
	}
	if s := humanSize(512); s != "512B" {
		t.Errorf("humanSize(512) = %q, want %q", s, "512B")
	}
	if s := humanSize(2048); s != "2.0K" {
		t.Errorf("humanSize(2K) = %q, want %q", s, "2.0K")
	}
}

func TestBusyBox_Du_KFlag(t *testing.T) {
	// du -k should accept the -k flag and produce output.
	var out bytes.Buffer
	rc := run([]string{"-k", "."}, nil, &out, &out, "")
	if rc != 0 {
		t.Errorf("expected 0, got %d", rc)
	}
	if out.Len() == 0 {
		t.Error("expected output")
	}
}

func TestBusyBox_Du_MFlag(t *testing.T) {
	// du -m should accept the -m flag.
	var out bytes.Buffer
	rc := run([]string{"-m", "."}, nil, &out, &out, "")
	if rc != 0 {
		t.Errorf("expected 0, got %d", rc)
	}
	if out.Len() == 0 {
		t.Error("expected output")
	}
}

func TestBusyBox_Du_LFlag(t *testing.T) {
	// du -l should accept the -l flag (count hard links).
	var out bytes.Buffer
	rc := run([]string{"-l", "."}, nil, &out, &out, "")
	if rc != 0 {
		t.Errorf("expected 0, got %d", rc)
	}
	if out.Len() == 0 {
		t.Error("expected output")
	}
}

func TestDuSummarize(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("abc"), 0644); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if code := run([]string{"-s", dir}, nil, &out, &out, ""); code != 0 {
		t.Errorf("expected 0, got %d", code)
	}
	if !strings.Contains(out.String(), dir) {
		t.Errorf("expected %q in output, got %q", dir, out.String())
	}
}

func TestDuMissingDir(t *testing.T) {
	var out bytes.Buffer
	if code := run([]string{"/nonexistent-xyz-dir"}, nil, &out, &out, ""); code != 1 {
		t.Errorf("expected 1, got %d", code)
	}
}

func TestDuMultipleRoots(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "sub")
	if err := os.Mkdir(sub, 0755); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if code := run([]string{dir, sub}, nil, &out, &out, ""); code != 0 {
		t.Errorf("expected 0, got %d", code)
	}
	if out.Len() == 0 {
		t.Error("expected output")
	}
}

func TestCLIBadFlag(t *testing.T) {
	var out, errBuf bytes.Buffer
	code := run([]string{"--no-such-flag"}, nil, &out, &errBuf, "")
	if code != 1 {
		t.Errorf("expected exit 1, got %d", code)
	}
	if !strings.Contains(errBuf.String(), "du:") {
		t.Errorf("expected stderr message, got %q", errBuf.String())
	}
}
