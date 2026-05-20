package rmdir

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunRemoveSingleEmptyDir(t *testing.T) {
	dir := t.TempDir()
	empty := filepath.Join(dir, "empty")
	os.Mkdir(empty, 0755)

	result, err := Run([]string{empty}, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Removed) != 1 || result.Removed[0] != empty {
		t.Errorf("expected [%s], got %v", empty, result.Removed)
	}
	if _, err := os.Stat(empty); !os.IsNotExist(err) {
		t.Error("directory should have been removed")
	}
}

func TestRunRemoveParents(t *testing.T) {
	dir := t.TempDir()
	child := filepath.Join(dir, "a", "b", "c")
	os.MkdirAll(child, 0755)

	result, err := Run([]string{child}, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should remove c, b, a (and possibly the temp dir if empty)
	if len(result.Removed) < 1 || result.Removed[0] != child {
		t.Errorf("expected first removed to be %s, got %v", child, result.Removed)
	}
}

func TestRunRemoveMultipleEmptyDirs(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a")
	b := filepath.Join(dir, "b")
	os.Mkdir(a, 0755)
	os.Mkdir(b, 0755)

	result, err := Run([]string{a, b}, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Removed) != 2 {
		t.Errorf("expected 2 removed, got %d", len(result.Removed))
	}
}

func TestRunNonEmptyRejection(t *testing.T) {
	dir := t.TempDir()
	nonempty := filepath.Join(dir, "nonempty")
	os.Mkdir(nonempty, 0755)
	os.WriteFile(filepath.Join(nonempty, "file"), []byte("x"), 0644)

	_, err := Run([]string{nonempty}, false)
	if err == nil {
		t.Error("expected error for non-empty directory")
	}
}
func TestCLI_Basic(t *testing.T) {
	d := filepath.Join(t.TempDir(), "rmme")
	os.Mkdir(d, 0755)
	var out bytes.Buffer
	code := run([]string{d}, nil, &out)
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if _, err := os.Stat(d); !os.IsNotExist(err) {
		t.Error("dir not removed")
	}
}
func TestCLI_Parents(t *testing.T) {
	base := t.TempDir()
	d := filepath.Join(base, "x", "y")
	os.MkdirAll(d, 0755)
	var out bytes.Buffer
	code := run([]string{"-p", d}, nil, &out)
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if _, err := os.Stat(filepath.Join(base, "x")); !os.IsNotExist(err) {
		t.Error("parent not removed")
	}
}
func TestCLI_NotEmpty(t *testing.T) {
	d := t.TempDir()
	os.Mkdir(filepath.Join(d, "sub"), 0755)
	var out bytes.Buffer
	code := run([]string{d}, nil, &out)
	if code != 1 {
		t.Errorf("exit %d, want 1 for non-empty dir", code)
	}
}
func TestCLI_JSON(t *testing.T) {
	d := filepath.Join(t.TempDir(), "jd")
	os.Mkdir(d, 0755)
	var out bytes.Buffer
	code := run([]string{"--json", d}, nil, &out)
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(out.String(), "\"removed\"") {
		t.Errorf("no JSON: %s", out.String())
	}
}
func TestCLI_BadFlag(t *testing.T) {
	var out bytes.Buffer
	code := run([]string{"--nonexistent"}, nil, &out)
	if code != 2 {
		t.Errorf("exit %d, want 2", code)
	}
}
