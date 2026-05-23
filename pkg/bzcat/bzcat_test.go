package bzcat

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

var helloBz2 = []byte{
	0x42, 0x5a, 0x68, 0x39, 0x31, 0x41, 0x59, 0x26, 0x53, 0x59, 0x5b, 0xb8, 0xe8, 0xa3, 0x00, 0x00,
	0x01, 0x44, 0x00, 0x00, 0x10, 0x02, 0x44, 0xa0, 0x00, 0x30, 0xcd, 0x00, 0xc3, 0x46, 0x29, 0x97,
	0x17, 0x72, 0x45, 0x38, 0x50, 0x90, 0x5b, 0xb8, 0xe8, 0xa3,
}

func TestBzcatStdin(t *testing.T) {
	stdin := bytes.NewReader(helloBz2)
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

func TestBzcatHelp(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"--help"}, nil, &stdout, &stderr, "")
	if code != 0 {
		t.Errorf("Expected code 0, got %d", code)
	}
	if !bytes.Contains(stdout.Bytes(), []byte("Usage: bzcat")) {
		t.Errorf("Expected help output, got: %s", stdout.String())
	}
}

func TestBzcatFlagError(t *testing.T) {
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

func TestBzcatFiles(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "bzcat-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create test file
	srcPath := filepath.Join(tempDir, "test.bz2")
	if err := os.WriteFile(srcPath, helloBz2, 0644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	// Case 1: Single file
	code := run([]string{"test.bz2"}, nil, &stdout, &stderr, tempDir)
	if code != 0 {
		t.Errorf("Expected code 0, got %d. Stderr: %s", code, stderr.String())
	}
	if stdout.String() != "HELLO\n" {
		t.Errorf("Expected 'HELLO\\n', got %q", stdout.String())
	}

	// Source file should NOT be deleted by bzcat!
	if _, err := os.Stat(srcPath); err != nil {
		t.Error("Expected source file to remain intact")
	}

	// Case 2: Missing file
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"nonexistent.bz2"}, nil, &stdout, &stderr, tempDir)
	if code == 0 {
		t.Error("Expected failure for nonexistent file")
	}

	// Case 3: Directory
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"."}, nil, &stdout, &stderr, tempDir)
	if code == 0 {
		t.Error("Expected failure for directory")
	}

	// Case 4: Multiple files (one valid, one missing)
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"test.bz2", "nonexistent.bz2"}, nil, &stdout, &stderr, tempDir)
	if code == 0 {
		t.Error("Expected failure when one of multiple files is missing")
	}
	if stdout.String() != "HELLO\n" {
		t.Errorf("Expected valid file to still decompress to stdout")
	}

	// Case 5: JSON mode output
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"--json", "test.bz2"}, nil, &stdout, &stderr, tempDir)
	if code != 0 {
		t.Errorf("Expected code 0 in JSON mode, got %d. Stderr: %s", code, stderr.String())
	}
}

func TestBzcatCorrupted(t *testing.T) {
	stdin := bytes.NewReader([]byte("corrupted"))
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{}, stdin, &stdout, &stderr, "")
	if code == 0 {
		t.Error("Expected error code for corrupted data")
	}

	// JSON mode error
	stdin.Seek(0, 0)
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"--json"}, stdin, &stdout, &stderr, "")
	if code == 0 {
		t.Error("Expected error code for corrupted data in JSON mode")
	}
}
