package stat

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunBasicFile(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "test.txt")
	os.WriteFile(f, []byte("hello"), 0644)

	result, err := Run(f)
	if err != nil {
		t.Fatal(err)
	}
	if result.Size != 5 {
		t.Errorf("size: got %d, want 5", result.Size)
	}
	if result.Path != f {
		t.Errorf("path: got %q, want %q", result.Path, f)
	}
	if result.Inode == 0 {
		t.Error("expected non-zero inode")
	}
	if result.Links == 0 {
		t.Error("expected non-zero links")
	}
}

func TestRunNonExistent(t *testing.T) {
	_, err := Run("/this/does/not/exist")
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestRunDirectory(t *testing.T) {
	dir := t.TempDir()
	result, err := Run(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsDir {
		t.Error("expected IsDir=true for directory")
	}
}

// --- CLI tests ---

func TestCLI_Basic(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "x.txt")
	os.WriteFile(f, []byte("data"), 0644)
	var out bytes.Buffer
	code := run([]string{f}, nil, &out)
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
}

func TestCLI_JSON(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "y.txt")
	os.WriteFile(f, []byte("json"), 0644)
	var out bytes.Buffer
	code := run([]string{"--json", f}, nil, &out)
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	if !strings.Contains(out.String(), "\"path\"") {
		t.Errorf("expected JSON, got: %s", out.String())
	}
}

func TestCLI_MissingFile(t *testing.T) {
	var out bytes.Buffer
	code := run([]string{}, nil, &out)
	if code != 1 {
		t.Errorf("expected exit 1, got %d", code)
	}
}

func TestCLI_NotExist(t *testing.T) {
	var out bytes.Buffer
	code := run([]string{"/nonexistent/zzz"}, nil, &out)
	if code != 1 {
		t.Errorf("expected exit 1 for nonexistent, got %d", code)
	}
}

func TestCLI_BadFlag(t *testing.T) {
	var out bytes.Buffer
	code := run([]string{"--nonexistent"}, nil, &out)
	if code != 2 {
		t.Errorf("expected exit 2, got %d", code)
	}
}

func TestCLI_LongFlag(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "long.txt")
	os.WriteFile(f, []byte("x"), 0644)
	var out bytes.Buffer
	code := run([]string{"--json", f}, nil, &out)
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	if !strings.Contains(out.String(), "\"path\"") {
		t.Errorf("expected JSON via --json, got: %s", out.String())
	}
}
