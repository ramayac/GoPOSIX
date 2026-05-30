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
	code := run([]string{link}, nil, &out, &out, "")
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
	code := run([]string{"-f", link}, nil, &out, &out, "")
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
	code := run([]string{"--json", link}, nil, &out, &out, "")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(out.String(), "\"target\"") {
		t.Errorf("no JSON: %s", out.String())
	}
}
func TestCLI_NoArgs(t *testing.T) {
	var out bytes.Buffer
	code := run([]string{}, nil, &out, &out, "")
	if code != 1 {
		t.Errorf("exit %d, want 1", code)
	}
}
func TestCLI_BadFlag(t *testing.T) {
	var out bytes.Buffer
	code := run([]string{"--nonexistent"}, nil, &out, &out, "")
	if code != 2 {
		t.Errorf("exit %d, want 2", code)
	}
}

func TestRunCanonicalize_LogicalWD(t *testing.T) {
	// Create base temp dir
	baseDir := t.TempDir()

	// Create physical directory and a target file in it
	physicalDir := filepath.Join(baseDir, "physical")
	if err := os.Mkdir(physicalDir, 0755); err != nil {
		t.Fatal(err)
	}
	targetFile := filepath.Join(physicalDir, "target.txt")
	if err := os.WriteFile(targetFile, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a link in the physical directory pointing to the target file
	linkFile := filepath.Join(physicalDir, "link.txt")
	if err := os.Symlink("target.txt", linkFile); err != nil {
		t.Fatal(err)
	}

	// Create a symlink pointing to the physical directory (the logical directory)
	logicalDir := filepath.Join(baseDir, "logical")
	if err := os.Symlink("physical", logicalDir); err != nil {
		t.Fatal(err)
	}

	// Cache active env and CWD to restore later
	origPWD := os.Getenv("PWD")
	origWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		os.Setenv("PWD", origPWD)
		os.Chdir(origWD)
	})

	// Change current working directory of the process to the physical directory
	if err := os.Chdir(physicalDir); err != nil {
		t.Fatal(err)
	}

	// Set the PWD environment variable to the logical directory
	os.Setenv("PWD", logicalDir)

	// Canonicalize "link.txt".
	// Since we are running under logicalDir PWD, the expected output should preserve
	// the logical path hierarchy: logicalDir/target.txt
	result, err := Run("link.txt", true)
	if err != nil {
		t.Fatal(err)
	}

	expectedTarget := filepath.Join(logicalDir, "target.txt")
	if filepath.Clean(result.Target) != filepath.Clean(expectedTarget) {
		t.Errorf("got target %q, want logical target %q", result.Target, expectedTarget)
	}
}

func TestRunCanonicalize_RecursiveSymlinks(t *testing.T) {
	baseDir := t.TempDir()

	physicalDir := filepath.Join(baseDir, "physical")
	if err := os.Mkdir(physicalDir, 0755); err != nil {
		t.Fatal(err)
	}
	targetFile := filepath.Join(physicalDir, "target.txt")
	if err := os.WriteFile(targetFile, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	// link2 -> target.txt
	link2 := filepath.Join(physicalDir, "link2.txt")
	if err := os.Symlink("target.txt", link2); err != nil {
		t.Fatal(err)
	}

	// link1 -> link2.txt
	link1 := filepath.Join(physicalDir, "link1.txt")
	if err := os.Symlink("link2.txt", link1); err != nil {
		t.Fatal(err)
	}

	logicalDir := filepath.Join(baseDir, "logical")
	if err := os.Symlink("physical", logicalDir); err != nil {
		t.Fatal(err)
	}

	origPWD := os.Getenv("PWD")
	origWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		os.Setenv("PWD", origPWD)
		os.Chdir(origWD)
	})

	if err := os.Chdir(physicalDir); err != nil {
		t.Fatal(err)
	}
	os.Setenv("PWD", logicalDir)

	result, err := Run("link1.txt", true)
	if err != nil {
		t.Fatal(err)
	}

	expectedTarget := filepath.Join(logicalDir, "target.txt")
	if filepath.Clean(result.Target) != filepath.Clean(expectedTarget) {
		t.Errorf("got target %q, want logical target %q", result.Target, expectedTarget)
	}
}

func TestReadlinkNonSymlink(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "regular.txt")
	os.WriteFile(f, []byte("content"), 0644)
	var buf bytes.Buffer
	code := run([]string{"-f", f}, nil, &buf, &buf, "")
	// -f canonicalize should work on non-symlinks too
	if code != 0 {
		t.Errorf("readlink -f regular: exit %d", code)
	}
}
func TestReadlinkMissingArg(t *testing.T) {
	var buf bytes.Buffer
	code := run([]string{}, nil, &buf, &buf, "")
	if code != 1 {
		t.Errorf("readlink no args: exit %d, want 1", code)
	}
}

func TestReadlinkCanonicalizeMissing(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "nonexistent")
	var buf bytes.Buffer
	code := run([]string{"-f", f}, nil, &buf, &buf, "")
	// -f should fail for non-existent paths
	if code != 1 {
		t.Errorf("readlink -f missing: exit %d, want 1", code)
	}
}
