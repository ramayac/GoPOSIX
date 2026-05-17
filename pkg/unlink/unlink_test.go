package unlink

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestUnlinkFile(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "testfile")
	os.WriteFile(f, []byte("data"), 0644)

	code := run([]string{f}, io.Discard)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	if _, err := os.Stat(f); !os.IsNotExist(err) {
		t.Error("file still exists after unlink")
	}
}

func TestUnlinkSymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	link := filepath.Join(dir, "link")
	os.WriteFile(target, []byte("data"), 0644)
	os.Symlink(target, link)

	code := run([]string{link}, io.Discard)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	if _, err := os.Lstat(link); !os.IsNotExist(err) {
		t.Error("symlink still exists after unlink")
	}
	if _, err := os.Stat(target); os.IsNotExist(err) {
		t.Error("target was removed — unlink should only remove the link")
	}
}

func TestUnlinkNonexistent(t *testing.T) {
	code := run([]string{"/tmp/goposix_nonexistent_test_file"}, io.Discard)
	if code != 1 {
		t.Errorf("expected exit 1 for nonexistent, got %d", code)
	}
}

func TestUnlinkDirectory(t *testing.T) {
	dir := t.TempDir()
	d := filepath.Join(dir, "testdir")
	os.Mkdir(d, 0755)

	code := run([]string{d}, io.Discard)
	if code != 1 {
		t.Errorf("expected exit 1 for directory, got %d", code)
	}
}

func TestUnlinkMissingOperand(t *testing.T) {
	code := run([]string{}, io.Discard)
	if code != 1 {
		t.Errorf("expected exit 1 for missing operand, got %d", code)
	}
}

func TestUnlinkJson(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "testfile")
	os.WriteFile(f, []byte("data"), 0644)

	var buf bytes.Buffer
	code := run([]string{"--json", f}, &buf)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	if !bytes.Contains(buf.Bytes(), []byte(`"removed"`)) {
		t.Error("JSON output missing removed field")
	}
}
