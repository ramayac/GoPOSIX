package mv

import (
	"bytes"
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

// --- BusyBox test suite hardening ---

func TestBusyBox_Mv_FilesToDir(t *testing.T) {
	dir := t.TempDir()
	srcDir := filepath.Join(dir, "src")
	dstDir := filepath.Join(dir, "dst")
	os.MkdirAll(srcDir, 0755)
	os.MkdirAll(dstDir, 0755)

	f1 := filepath.Join(srcDir, "f1")
	f2 := filepath.Join(srcDir, "f2")
	link := filepath.Join(srcDir, "link")
	subDir := filepath.Join(srcDir, "sub")
	subFile := filepath.Join(subDir, "subfile")

	os.WriteFile(f1, []byte("one"), 0644)
	os.WriteFile(f2, []byte("two"), 0644)
	os.Symlink("f2", link)
	os.MkdirAll(subDir, 0755)
	os.WriteFile(subFile, []byte("sub"), 0644)

	_, err := Run([]string{f1, f2, link, subDir}, dstDir)
	if err != nil {
		t.Fatalf("mv files to dir failed: %v", err)
	}
	for _, path := range []string{f1, f2, link, subDir} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Errorf("source should not exist: %s", path)
		}
	}
	if _, err := os.Stat(filepath.Join(dstDir, "f1")); err != nil {
		t.Error("dst/f1 should exist")
	}
	if _, err := os.Stat(filepath.Join(dstDir, "sub", "subfile")); err != nil {
		t.Error("dst/sub/subfile should exist")
	}
	fi, err := os.Lstat(filepath.Join(dstDir, "link"))
	if err != nil || fi.Mode()&os.ModeSymlink == 0 {
		t.Error("dst/link should be a symlink")
	}
}

func TestBusyBox_Mv_TargetFlag(t *testing.T) {
	dir := t.TempDir()
	srcDir := filepath.Join(dir, "src")
	dstDir := filepath.Join(dir, "dst")
	os.MkdirAll(srcDir, 0755)
	os.MkdirAll(dstDir, 0755)
	f1 := filepath.Join(srcDir, "f1")
	os.WriteFile(f1, []byte("data"), 0644)

	var buf bytes.Buffer
	code := run([]string{"-t", dstDir, f1}, nil, &buf, &buf, "")
	if code != 0 {
		t.Fatalf("mv -t failed with exit %d", code)
	}
	if _, err := os.Stat(filepath.Join(dstDir, "f1")); err != nil {
		t.Error("dst/f1 should exist after mv -t")
	}
}

func TestBusyBox_Mv_MovesUnreadableFile(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "unreadable")
	dst := filepath.Join(dir, "moved")
	os.WriteFile(src, []byte("secret"), 0200)
	_, err := Run([]string{src}, dst)
	if err != nil {
		t.Fatalf("mv unreadable file failed: %v", err)
	}
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Error("source should not exist after move")
	}
	if _, err := os.Stat(dst); err != nil {
		t.Error("destination should exist after move")
	}
}

func TestBusyBox_Mv_RefusesDirToSubdir(t *testing.T) {
	dir := t.TempDir()
	parent := filepath.Join(dir, "parent")
	sub := filepath.Join(parent, "sub")
	os.MkdirAll(sub, 0755)
	_, err := Run([]string{parent}, sub)
	if err == nil {
		t.Error("mv dir into self should fail")
	}
}
