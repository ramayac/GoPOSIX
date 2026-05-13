package rmdir

import (
	"os"
	"path/filepath"
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
