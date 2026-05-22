package ln

import (
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
