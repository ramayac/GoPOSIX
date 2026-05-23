package uncompress

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"
)

// Compressed "HELLO\n" using UNIX .Z format.
var helloZ = []byte{0x1f, 0x9d, 0x90, 0x48, 0x8a, 0x30, 0x61, 0xf2, 0x44, 0x01}

func TestUncompressStdin(t *testing.T) {
	stdin := bytes.NewReader(helloZ)
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{}, stdin, &stdout, &stderr, "")
	if code != 0 {
		t.Errorf("Expected exit code 0, got %d. Stderr: %s", code, stderr.String())
	}

	if stdout.String() != "HELLO\n" {
		t.Errorf("Expected 'HELLO\\n', got %q", stdout.String())
	}
}

func TestUncompressHelp(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"--help"}, nil, &stdout, &stderr, "")
	if code != 0 {
		t.Errorf("Expected code 0, got %d", code)
	}
	if !bytes.Contains(stdout.Bytes(), []byte("Usage: uncompress")) {
		t.Errorf("Expected help output, got: %s", stdout.String())
	}
}

func TestUncompressFlagError(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"--invalid-flag"}, nil, &stdout, &stderr, "")
	if code == 0 {
		t.Error("Expected non-zero exit code on invalid flag")
	}

	stdout.Reset()
	stderr.Reset()
	code = run([]string{"--invalid-flag", "--json"}, nil, &stdout, &stderr, "")
	if code == 0 {
		t.Error("Expected non-zero exit code on invalid flag with --json")
	}
}

func TestUncompressFiles(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "uncompress-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	srcPath := filepath.Join(tempDir, "test.z")
	if err := os.WriteFile(srcPath, helloZ, 0644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	// Case 1: Keep mode
	code := run([]string{"-k", "test.z"}, nil, &stdout, &stderr, tempDir)
	if code != 0 {
		t.Errorf("Expected code 0, got %d. Stderr: %s", code, stderr.String())
	}

	destPath := filepath.Join(tempDir, "test")
	content, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "HELLO\n" {
		t.Errorf("Expected 'HELLO\\n', got %q", string(content))
	}

	// Ensure source file remains (keepMode)
	if _, err := os.Stat(srcPath); err != nil {
		t.Error("Expected source file to remain under -k flag")
	}

	// Case 2: Source deletion (default)
	os.Remove(destPath)
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"test.z"}, nil, &stdout, &stderr, tempDir)
	if code != 0 {
		t.Errorf("Expected code 0, got %d. Stderr: %s", code, stderr.String())
	}
	if _, err := os.Stat(srcPath); !os.IsNotExist(err) {
		t.Error("Expected source file to be deleted")
	}

	// Case 3: Suffix check errors
	badSuffix := filepath.Join(tempDir, "test.zz")
	if err := os.WriteFile(badSuffix, helloZ, 0644); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"test.zz"}, nil, &stdout, &stderr, tempDir)
	if code == 0 {
		t.Error("Expected failure for bad suffix")
	}

	// Case 4: Is a directory check
	subDir := filepath.Join(tempDir, "subdir.z")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"subdir.z"}, nil, &stdout, &stderr, tempDir)
	if code == 0 {
		t.Error("Expected failure when extracting a directory")
	}

	// Case 5: Destination already exists
	srcPath2 := filepath.Join(tempDir, "test2.z")
	if err := os.WriteFile(srcPath2, helloZ, 0644); err != nil {
		t.Fatal(err)
	}
	destPath2 := filepath.Join(tempDir, "test2")
	if err := os.WriteFile(destPath2, []byte("exist"), 0644); err != nil {
		t.Fatal(err)
	}

	stdout.Reset()
	stderr.Reset()
	code = run([]string{"test2.z"}, nil, &stdout, &stderr, tempDir)
	if code == 0 {
		t.Error("Expected failure since destination already exists")
	}

	// Case 6: Force overwrite
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"-f", "test2.z"}, nil, &stdout, &stderr, tempDir)
	if code != 0 {
		t.Errorf("Expected code 0 with force option, got %d. Stderr: %s", code, stderr.String())
	}

	// Case 7: Nonexistent file
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"nonexistent.z"}, nil, &stdout, &stderr, tempDir)
	if code == 0 {
		t.Error("Expected failure for nonexistent file")
	}
}

func TestUncompressCorrupted(t *testing.T) {
	stdin := bytes.NewReader([]byte("not-a-valid-Z-header"))
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{}, stdin, &stdout, &stderr, "")
	if code == 0 {
		t.Error("Expected error code for corrupted data")
	}

	// Stdin with JSON
	stdin.Seek(0, io.SeekStart)
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"--json"}, stdin, &stdout, &stderr, "")
	if code == 0 {
		t.Error("Expected error code for corrupted data in JSON mode")
	}
}
