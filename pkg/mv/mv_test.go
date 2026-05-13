package mv

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunRenameSingleFile(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	dst := filepath.Join(dir, "dst")
	os.WriteFile(src, []byte("hello"), 0644)

	result, err := Run([]string{src}, dst)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Moved) != 1 {
		t.Fatalf("expected 1 move, got %d", len(result.Moved))
	}
	if result.Moved[0].From != src || result.Moved[0].To != dst {
		t.Errorf("bad record: %+v", result.Moved[0])
	}
	// Source should no longer exist
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Error("source should not exist after move")
	}
	data, _ := os.ReadFile(dst)
	if string(data) != "hello" {
		t.Errorf("expected 'hello', got %q", string(data))
	}
}

func TestRunMoveIntoDirectory(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	os.WriteFile(src, []byte("data"), 0644)
	dstdir := filepath.Join(dir, "dstdir")
	os.Mkdir(dstdir, 0755)

	result, err := Run([]string{src}, dstdir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expectedDst := filepath.Join(dstdir, "src")
	if result.Moved[0].To != expectedDst {
		t.Errorf("expected dst %s, got %s", expectedDst, result.Moved[0].To)
	}
	if _, err := os.Stat(expectedDst); err != nil {
		t.Error("file should exist in destination dir")
	}
}

func TestRunMoveMultipleFiles(t *testing.T) {
	dir := t.TempDir()
	src1 := filepath.Join(dir, "a")
	src2 := filepath.Join(dir, "b")
	os.WriteFile(src1, []byte("1"), 0644)
	os.WriteFile(src2, []byte("2"), 0644)
	dstdir := filepath.Join(dir, "dstdir")
	os.Mkdir(dstdir, 0755)

	result, err := Run([]string{src1, src2}, dstdir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Moved) != 2 {
		t.Errorf("expected 2 moves, got %d", len(result.Moved))
	}
}

func TestRunMoveOverwrite(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	dst := filepath.Join(dir, "dst")
	os.WriteFile(src, []byte("new"), 0644)
	os.WriteFile(dst, []byte("old"), 0644)

	result, err := Run([]string{src}, dst)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Moved[0].From != src || result.Moved[0].To != dst {
		t.Errorf("bad record: %+v", result.Moved[0])
	}
	data, _ := os.ReadFile(dst)
	if string(data) != "new" {
		t.Errorf("expected 'new' after overwrite, got %q", string(data))
	}
}
