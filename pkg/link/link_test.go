package link

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func TestLinkCreatesHardLink(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	dst := filepath.Join(dir, "dst")
	os.WriteFile(src, []byte("data"), 0644)

	code := run([]string{src, dst}, nil, io.Discard)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}

	fi1, _ := os.Stat(src)
	fi2, _ := os.Stat(dst)
	stat1 := fi1.Sys().(*syscall.Stat_t)
	stat2 := fi2.Sys().(*syscall.Stat_t)
	if stat1.Ino != stat2.Ino {
		t.Error("inodes differ: not a hard link")
	}

	data, _ := os.ReadFile(dst)
	if string(data) != "data" {
		t.Errorf("expected 'data', got %q", string(data))
	}
}

func TestLinkNonexistentSource(t *testing.T) {
	dir := t.TempDir()
	code := run([]string{filepath.Join(dir, "nonexistent"), filepath.Join(dir, "dst")}, nil, io.Discard)
	if code != 1 {
		t.Errorf("expected exit 1 for nonexistent source, got %d", code)
	}
}

func TestLinkExistingTarget(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	dst := filepath.Join(dir, "dst")
	os.WriteFile(src, []byte("data"), 0644)
	os.WriteFile(dst, []byte("old"), 0644)

	code := run([]string{src, dst}, nil, io.Discard)
	if code != 1 {
		t.Errorf("expected exit 1 for existing target, got %d", code)
	}
}

func TestLinkMissingOperand(t *testing.T) {
	code := run([]string{"/x"}, nil, io.Discard)
	if code != 1 {
		t.Errorf("expected exit 1 for missing operand, got %d", code)
	}
}

func TestLinkJson(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	dst := filepath.Join(dir, "dst")
	os.WriteFile(src, []byte("data"), 0644)

	var buf bytes.Buffer
	code := run([]string{"--json", src, dst}, nil, &buf)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	if !bytes.Contains(buf.Bytes(), []byte(`"source"`)) {
		t.Error("JSON output missing source field")
	}
}
