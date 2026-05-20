package readlink

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunSymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target.txt")
	os.WriteFile(target, []byte("x"), 0644)
	link := filepath.Join(dir, "link")
	os.Symlink(target, link)

	result, err := Run(link, false)
	if err != nil {
		t.Fatal(err)
	}
	if result.Target != target {
		t.Errorf("got %q, want %q", result.Target, target)
	}
}

func TestRunCanonicalize(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "real.txt")
	os.WriteFile(target, []byte("x"), 0644)
	link := filepath.Join(dir, "link")
	os.Symlink(target, link)

	result, err := Run(link, true)
	if err != nil {
		t.Fatal(err)
	}
	if result.Target == "" {
		t.Error("expected non-empty canonical target")
	}
}

func TestRunNotSymlink(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "plain.txt")
	os.WriteFile(f, []byte("x"), 0644)
	_, err := Run(f, false)
	if err == nil {
		t.Error("expected error for non-symlink")
	}
}
func TestCLI_Basic(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	os.WriteFile(target, []byte("x"), 0644)
	link := filepath.Join(dir, "link")
	os.Symlink(target, link)
	var out bytes.Buffer
	code := run([]string{link}, &out)
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(out.String(), "target") {
		t.Errorf("expected target, got: %s", out.String())
	}
}
func TestCLI_Canonicalize(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	os.WriteFile(target, []byte("x"), 0644)
	link := filepath.Join(dir, "link")
	os.Symlink(target, link)
	var out bytes.Buffer
	code := run([]string{"-f", link}, &out)
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
}
func TestCLI_JSON(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "t")
	os.WriteFile(target, []byte("x"), 0644)
	link := filepath.Join(dir, "l")
	os.Symlink(target, link)
	var out bytes.Buffer
	code := run([]string{"--json", link}, &out)
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(out.String(), "\"target\"") {
		t.Errorf("no JSON: %s", out.String())
	}
}
func TestCLI_NoArgs(t *testing.T) {
	var out bytes.Buffer
	code := run([]string{}, &out)
	if code != 1 {
		t.Errorf("exit %d, want 1", code)
	}
}
func TestCLI_BadFlag(t *testing.T) {
	var out bytes.Buffer
	code := run([]string{"--nonexistent"}, &out)
	if code != 2 {
		t.Errorf("exit %d, want 2", code)
	}
}
