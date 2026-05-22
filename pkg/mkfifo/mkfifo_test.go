package mkfifo

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestMkfifoCreate(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "testpipe")

	code := run([]string{path}, nil, io.Discard, io.Discard, "")
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}

	fi, err := os.Stat(path)
	if err != nil {
		t.Fatalf("fifo not created: %v", err)
	}
	if fi.Mode()&os.ModeNamedPipe == 0 {
		t.Error("expected named pipe, got regular file")
	}
}

func TestMkfifoCustomMode(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "testpipe")

	code := run([]string{"-m", "0644", path}, nil, io.Discard, io.Discard, "")
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}

	fi, err := os.Stat(path)
	if err != nil {
		t.Fatalf("fifo not created: %v", err)
	}
	if fi.Mode().Perm() != 0644 {
		t.Errorf("expected mode 0644, got 0%o", fi.Mode().Perm())
	}
}

func TestMkfifoExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "existing")
	os.WriteFile(path, []byte("data"), 0644)

	code := run([]string{path}, nil, io.Discard, io.Discard, "")
	if code != 1 {
		t.Errorf("expected exit 1 for existing file, got %d", code)
	}
}

func TestMkfifoMissingOperand(t *testing.T) {
	code := run([]string{}, nil, io.Discard, io.Discard, "")
	if code != 1 {
		t.Errorf("expected exit 1 for missing operand, got %d", code)
	}
}

func TestMkfifoInvalidMode(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "testpipe")

	code := run([]string{"-m", "999", path}, nil, io.Discard, io.Discard, "")
	if code != 1 {
		t.Errorf("expected exit 1 for invalid mode, got %d", code)
	}
}

func TestMkfifoJson(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "jsonpipe")

	var buf bytes.Buffer
	code := run([]string{"--json", "-m", "0600", path}, nil, &buf, &buf, "")
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	if !bytes.Contains(buf.Bytes(), []byte(`"path"`)) {
		t.Error("JSON output missing path field")
	}
}
