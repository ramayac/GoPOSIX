package cp

import (
	"bytes"
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

// --- BusyBox test suite hardening ---

func TestBusyBox_CP_ArchivePreservesSymlinks(t *testing.T) {
	// cp -a bar baz: symlink copy with archive flag
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	link := filepath.Join(dir, "link")
	dst := filepath.Join(dir, "dst")
	os.WriteFile(src, []byte("hello"), 0644)
	os.Symlink("src", link)

	// Run via CLI to test -a flag
	var buf bytes.Buffer
	code := run([]string{"-a", link, dst}, nil, &buf, &buf, "")
	if code != 0 {
		t.Fatalf("cp -a failed: exit %d", code)
	}
	fi, _ := os.Lstat(dst)
	if fi.Mode()&os.ModeSymlink == 0 {
		t.Error("cp -a should preserve symlinks")
	}
	target, _ := os.Readlink(dst)
	if target != "src" {
		t.Errorf("symlink target: %q, want src", target)
	}
}

func TestBusyBox_CP_ArchiveMultipleFilesAndDirs(t *testing.T) {
	// cp -a file1 file2 link1 dir1 there/
	dir := t.TempDir()
	srcDir := filepath.Join(dir, "src")
	dstDir := filepath.Join(dir, "dst")
	os.MkdirAll(srcDir, 0755)
	os.MkdirAll(dstDir, 0755)

	os.WriteFile(filepath.Join(srcDir, "f1"), []byte("one"), 0644)
	os.WriteFile(filepath.Join(srcDir, "f2"), []byte("two"), 0644)
	os.Symlink("f2", filepath.Join(srcDir, "link1"))
	os.MkdirAll(filepath.Join(srcDir, "subdir"), 0755)

	f1 := filepath.Join(srcDir, "f1")
	f2 := filepath.Join(srcDir, "f2")
	l1 := filepath.Join(srcDir, "link1")
	sd := filepath.Join(srcDir, "subdir")

	var buf bytes.Buffer
	code := run([]string{"-a", f1, f2, l1, sd, dstDir}, nil, &buf, &buf, "")
	if code != 0 {
		t.Fatalf("cp -a failed: exit %d", code)
	}

	// Check files were copied
	if _, err := os.Stat(filepath.Join(dstDir, "f1")); err != nil {
		t.Error("f1 not copied")
	}
	if _, err := os.Stat(filepath.Join(dstDir, "f2")); err != nil {
		t.Error("f2 not copied")
	}

	// Check symlink preserved
	fi, _ := os.Lstat(filepath.Join(dstDir, "link1"))
	if fi.Mode()&os.ModeSymlink == 0 {
		t.Error("link1 should be a symlink")
	}
	target, _ := os.Readlink(filepath.Join(dstDir, "link1"))
	if target != "f2" {
		t.Errorf("link1 target: %q, want f2", target)
	}

	// Check directory copied
	if _, err := os.Stat(filepath.Join(dstDir, "subdir")); err != nil {
		t.Error("subdir not copied")
	}
}

func TestBusyBox_CP_Parents(t *testing.T) {
	// cp --parents foo/bar/baz/file dir/ → dir/foo/bar/baz/file
	dir := t.TempDir()
	srcBase := filepath.Join(dir, "src")
	dstBase := filepath.Join(dir, "dst")
	deep := filepath.Join(srcBase, "foo", "bar", "baz")
	os.MkdirAll(deep, 0755)
	deepFile := filepath.Join(deep, "file")
	os.WriteFile(deepFile, []byte("deep"), 0644)
	os.MkdirAll(dstBase, 0755)

	// Use a relative path for parents to work correctly
	cwd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(cwd)

	var buf bytes.Buffer
	relSrc := "src/foo/bar/baz/file"
	code := run([]string{"--parents", relSrc, "dst"}, nil, &buf, &buf, "")
	if code != 0 {
		t.Fatalf("cp --parents failed: exit %d", code)
	}

	expected := filepath.Join(dir, "dst", relSrc)
	if _, err := os.Stat(expected); err != nil {
		t.Errorf("expected %s to exist: %v", expected, err)
	}
}

func TestBusyBox_CP_HardLinkPreservation(t *testing.T) {
	// cp -d foo bar baz: hard links should be preserved
	dir := t.TempDir()
	foo := filepath.Join(dir, "foo")
	bar := filepath.Join(dir, "bar")
	baz := filepath.Join(dir, "baz")

	os.WriteFile(foo, []byte("linkme"), 0644)
	os.Link(foo, bar)
	os.MkdirAll(baz, 0755)

	var buf bytes.Buffer
	code := run([]string{"-d", foo, bar, baz}, nil, &buf, &buf, "")
	if code != 0 {
		t.Fatalf("cp -d failed: exit %d", code)
	}

	fi1, _ := os.Stat(filepath.Join(baz, "foo"))
	fi2, _ := os.Stat(filepath.Join(baz, "bar"))
	// Same inode = hard linked
	if !os.SameFile(fi1, fi2) {
		t.Error("baz/foo and baz/bar should be hard links (same file)")
	}
}

func TestBusyBox_CP_UnreadableFile(t *testing.T) {
	// cp unreadable_file dst → should fail, dst should NOT be created
	dir := t.TempDir()
	src := filepath.Join(dir, "unreadable")
	dst := filepath.Join(dir, "dst")

	os.WriteFile(src, []byte("secret"), 0000)

	var buf bytes.Buffer
	code := run([]string{src, dst}, nil, &buf, &buf, "")
	// Should fail (exit non-zero)
	if code == 0 {
		t.Fatal("cp of unreadable file should fail")
	}
	// Destination should NOT exist
	if _, err := os.Stat(dst); err == nil {
		t.Error("destination should not have been created")
	}
}

func TestCLI_RecursiveDir(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "srcdir")
	dst := filepath.Join(dir, "dstdir")
	os.MkdirAll(src, 0755)
	os.WriteFile(filepath.Join(src, "file.txt"), []byte("content"), 0644)
	var out bytes.Buffer
	// cp -r srcdir dstdir
	code := run([]string{"-r", src, dst}, nil, &out, &out, "")
	if code != 0 {
		t.Fatalf("exit %d, want 0. stderr: %s", code, out.String())
	}
	// Check nested file copied.
	if _, err := os.Stat(filepath.Join(dst, "file.txt")); err != nil {
		t.Error("nested file not copied")
	}
}
