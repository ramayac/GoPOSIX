package realpath

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRealpathResolve(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "realpath-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	subDir := filepath.Join(tmpDir, "sub")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a dummy file
	testFile := filepath.Join(subDir, "file.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a symlink to the file
	link1 := filepath.Join(tmpDir, "link1")
	if err := os.Symlink("sub/file.txt", link1); err != nil {
		t.Fatal(err)
	}

	// Create a symlink to non-existent file
	link2 := filepath.Join(tmpDir, "link2")
	if err := os.Symlink("sub/nonexistent.txt", link2); err != nil {
		t.Fatal(err)
	}

	// Create a symlink to non-existent directory and file
	link3 := filepath.Join(tmpDir, "link3")
	if err := os.Symlink("nonexistent_dir/file.txt", link3); err != nil {
		t.Fatal(err)
	}

	// Create a symlink with absolute path target
	linkAbs := filepath.Join(tmpDir, "linkabs")
	if err := os.Symlink(testFile, linkAbs); err != nil {
		t.Fatal(err)
	}

	// Test case 1: Existing file
	res, err := resolvePath(testFile, tmpDir)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if res != testFile {
		t.Errorf("expected %q, got %q", testFile, res)
	}

	// Test case 2: Symlink to existing file
	res, err = resolvePath(link1, tmpDir)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if res != testFile {
		t.Errorf("expected %q, got %q", testFile, res)
	}

	// Test case 3: Symlink to non-existent file (parent exists)
	res, err = resolvePath(link2, tmpDir)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	expectedLink2Target := filepath.Join(subDir, "nonexistent.txt")
	if res != expectedLink2Target {
		t.Errorf("expected %q, got %q", expectedLink2Target, res)
	}

	// Test case 4: Symlink to non-existent directory (should fail)
	_, err = resolvePath(link3, tmpDir)
	if err == nil {
		t.Error("expected error resolving symlink to non-existent directory, got nil")
	}

	// Test case 5: Symlink with absolute target
	res, err = resolvePath(linkAbs, tmpDir)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if res != testFile {
		t.Errorf("expected %q, got %q", testFile, res)
	}
}

func TestRealpathInfiniteSymlink(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "realpath-infinite")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	linkA := filepath.Join(tmpDir, "linkA")
	linkB := filepath.Join(tmpDir, "linkB")

	if err := os.Symlink("linkB", linkA); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("linkA", linkB); err != nil {
		t.Fatal(err)
	}

	_, err = resolvePath(linkA, tmpDir)
	if err == nil {
		t.Error("expected error due to infinite symlink loop, got nil")
	}
}

func TestRealpathCLI(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "realpath-cli-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	var stdout, stderr bytes.Buffer

	// Resolve existing tmpDir
	code := run([]string{tmpDir}, nil, &stdout, &stderr, tmpDir)
	if code != 0 {
		t.Errorf("expected exit 0, got %d, stderr: %q", code, stderr.String())
	}
	cleanedStdout := strings.TrimSpace(stdout.String())
	evalTmpDir, _ := filepath.EvalSymlinks(tmpDir)
	if cleanedStdout != evalTmpDir {
		t.Errorf("expected %q, got %q", evalTmpDir, cleanedStdout)
	}

	// Resolve non-existent file but existing parent dir
	stdout.Reset()
	stderr.Reset()
	nonExistentFile := filepath.Join(tmpDir, "doesnotexist.txt")
	code = run([]string{nonExistentFile}, nil, &stdout, &stderr, tmpDir)
	if code != 0 {
		t.Errorf("expected exit 0, got %d, stderr: %q", code, stderr.String())
	}
	cleanedStdout = strings.TrimSpace(stdout.String())
	if cleanedStdout != nonExistentFile {
		t.Errorf("expected %q, got %q", nonExistentFile, cleanedStdout)
	}

	// Resolve non-existent directory (should fail)
	stdout.Reset()
	stderr.Reset()
	nonExistentDir := filepath.Join(tmpDir, "doesnotexist/file.txt")
	code = run([]string{nonExistentDir}, nil, &stdout, &stderr, tmpDir)
	if code != 1 {
		t.Errorf("expected exit 1, got %d", code)
	}
	if !strings.Contains(stderr.String(), "No such file or directory") {
		t.Errorf("expected 'No such file or directory' in stderr, got: %q", stderr.String())
	}

	// Test -m flag (missing components okay)
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"-m", nonExistentDir}, nil, &stdout, &stderr, tmpDir)
	if code != 0 {
		t.Errorf("expected exit 0 with -m, got %d, stderr: %q", code, stderr.String())
	}
	cleanedStdout = strings.TrimSpace(stdout.String())
	if cleanedStdout != nonExistentDir {
		t.Errorf("expected %q, got %q", nonExistentDir, cleanedStdout)
	}

	// Test -e flag (last component must exist)
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"-e", nonExistentFile}, nil, &stdout, &stderr, tmpDir)
	if code != 1 {
		t.Errorf("expected exit 1 with -e on non-existent file, got %d", code)
	}

	// Test -s / --no-symlinks flag
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"-s", tmpDir}, nil, &stdout, &stderr, tmpDir)
	if code != 0 {
		t.Errorf("expected exit 0 with -s, got %d, stderr: %q", code, stderr.String())
	}

	// Test -s -e with non-existent file
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"-s", "-e", nonExistentFile}, nil, &stdout, &stderr, tmpDir)
	if code != 1 {
		t.Errorf("expected exit 1 with -s -e on non-existent file, got %d", code)
	}

	// Test --json flag
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"--json", tmpDir}, nil, &stdout, &stderr, tmpDir)
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
	if !strings.Contains(stdout.String(), `"resolved"`) {
		t.Errorf("expected JSON matches envelope, got: %q", stdout.String())
	}
}
