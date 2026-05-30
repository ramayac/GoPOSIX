package ln

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestHardLink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	link := filepath.Join(dir, "link")
	os.WriteFile(target, []byte("data"), 0644)

	code := run([]string{target, link}, nil, os.Stdout, os.Stdout, "")
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	fi, err := os.Stat(link)
	if err != nil {
		t.Fatalf("link not created: %v", err)
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		t.Error("expected hard link, got symlink")
	}
}

func TestSymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	link := filepath.Join(dir, "link")
	os.WriteFile(target, []byte("data"), 0644)

	code := run([]string{"-s", target, link}, nil, os.Stdout, os.Stdout, "")
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	fi, err := os.Lstat(link)
	if err != nil {
		t.Fatalf("link not created: %v", err)
	}
	if fi.Mode()&os.ModeSymlink == 0 {
		t.Error("expected symlink, got hard link")
	}
}

func TestForceOverwrite(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	link := filepath.Join(dir, "link")
	os.WriteFile(target, []byte("data"), 0644)
	os.WriteFile(link, []byte("old"), 0644)

	code := run([]string{"-f", target, link}, nil, os.Stdout, os.Stdout, "")
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
}

func TestMissingOperand(t *testing.T) {
	code := run([]string{"/x"}, nil, os.Stdout, os.Stdout, "")
	if code != 1 {
		t.Errorf("expected exit 1 for missing operand, got %d", code)
	}
}

func TestLnForceFlag(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	os.WriteFile(src, []byte("content"), 0644)
	dst := filepath.Join(dir, "dst.txt")
	os.WriteFile(dst, []byte("old"), 0644)
	var buf bytes.Buffer
	code := run([]string{"-f", src, dst}, nil, &buf, &buf, "")
	if code != 0 {
		t.Errorf("ln -f exit %d, want 0", code)
	}
	if _, err := os.Stat(dst); err != nil {
		t.Error("dst should exist after ln -f")
	}
}
func TestLnTargetIsDir(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "file.txt")
	os.WriteFile(src, []byte("x"), 0644)
	sub := filepath.Join(dir, "subdir")
	os.MkdirAll(sub, 0755)
	var buf bytes.Buffer
	code := run([]string{src, sub}, nil, &buf, &buf, "")
	if code != 0 {
		t.Errorf("ln to dir exit %d, want 0", code)
	}
	linked := filepath.Join(sub, "file.txt")
	if _, err := os.Stat(linked); err != nil {
		t.Error("link should be created inside directory")
	}
}
