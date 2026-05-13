package cp

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunCopySingleFile(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	dst := filepath.Join(dir, "dst")
	os.WriteFile(src, []byte("hello"), 0644)

	result, err := Run([]string{src}, dst, false, false, SymlinkFollow)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Copied) != 1 {
		t.Fatalf("expected 1 copy, got %d", len(result.Copied))
	}
	if result.Copied[0].From != src || result.Copied[0].To != dst {
		t.Errorf("bad record: from=%s to=%s", result.Copied[0].From, result.Copied[0].To)
	}
	data, _ := os.ReadFile(dst)
	if string(data) != "hello" {
		t.Errorf("expected 'hello', got %q", string(data))
	}
}

func TestRunCopyIntoDirectory(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	os.WriteFile(src, []byte("hello"), 0644)
	dstdir := filepath.Join(dir, "dstdir")
	os.Mkdir(dstdir, 0755)

	result, err := Run([]string{src}, dstdir, false, false, SymlinkFollow)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expectedDst := filepath.Join(dstdir, "src")
	if result.Copied[0].To != expectedDst {
		t.Errorf("expected dst %s, got %s", expectedDst, result.Copied[0].To)
	}
}

func TestRunCopyDirectoryRecursive(t *testing.T) {
	dir := t.TempDir()
	srcdir := filepath.Join(dir, "srcdir")
	os.Mkdir(srcdir, 0755)
	os.WriteFile(filepath.Join(srcdir, "file.txt"), []byte("data"), 0644)

	dstdir := filepath.Join(dir, "dstdir")
	result, err := Run([]string{srcdir}, dstdir, true, false, SymlinkFollow)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Copied) < 1 {
		t.Error("expected at least 1 copy (file inside dir)")
	}
	// Verify the file was copied inside the destination
	if _, err := os.Stat(filepath.Join(dstdir, "file.txt")); err != nil {
		t.Errorf("expected file to exist inside destination: %v", err)
	}
}

func TestRunCopyDirectoryWithoutRecursiveFails(t *testing.T) {
	dir := t.TempDir()
	srcdir := filepath.Join(dir, "srcdir")
	os.Mkdir(srcdir, 0755)

	_, err := Run([]string{srcdir}, filepath.Join(dir, "dst"), false, false, SymlinkFollow)
	if err == nil {
		t.Error("expected error when copying directory without -r")
	}
}

func TestRunCopySymlinkPreserve(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	link := filepath.Join(dir, "link")
	os.WriteFile(src, []byte("data"), 0644)
	os.Symlink(src, link)

	dst := filepath.Join(dir, "dst")
	result, err := Run([]string{link}, dst, false, false, SymlinkPreserve)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Copied[0].From != link || result.Copied[0].To != dst {
		t.Errorf("bad record: from=%s to=%s", result.Copied[0].From, result.Copied[0].To)
	}
	fi, _ := os.Lstat(dst)
	if fi.Mode()&os.ModeSymlink == 0 {
		t.Error("expected symlink to be preserved")
	}
}
