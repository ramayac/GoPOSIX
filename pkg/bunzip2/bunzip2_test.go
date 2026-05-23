package bunzip2

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"
)

// A pre-bzipped "HELLO\n" byte sequence.
var helloBz2 = []byte{
	0x42, 0x5a, 0x68, 0x39, 0x31, 0x41, 0x59, 0x26, 0x53, 0x59, 0x5b, 0xb8, 0xe8, 0xa3, 0x00, 0x00,
	0x01, 0x44, 0x00, 0x00, 0x10, 0x02, 0x44, 0xa0, 0x00, 0x30, 0xcd, 0x00, 0xc3, 0x46, 0x29, 0x97,
	0x17, 0x72, 0x45, 0x38, 0x50, 0x90, 0x5b, 0xb8, 0xe8, 0xa3,
}

func TestBunzip2Stdin(t *testing.T) {
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

func TestBunzip2Help(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"--help"}, nil, &stdout, &stderr, "")
	if code != 0 {
		t.Errorf("Expected code 0, got %d", code)
	}
	if !bytes.Contains(stdout.Bytes(), []byte("Usage: bunzip2")) {
		t.Errorf("Expected help output, got: %s", stdout.String())
	}
}

func TestBunzip2FlagError(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"--invalid-flag"}, nil, &stdout, &stderr, "")
	if code == 0 {
		t.Error("Expected non-zero exit code on invalid flag")
	}

	// JSON mode error
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"--invalid-flag", "--json"}, nil, &stdout, &stderr, "")
	if code == 0 {
		t.Error("Expected non-zero exit code on invalid flag with --json")
	}
}

func TestBunzip2Files(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "bunzip2-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Case 1: Standard file extraction
	srcPath := filepath.Join(tempDir, "test.bz2")
	if err := os.WriteFile(srcPath, helloBz2, 0644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"-k", "test.bz2"}, nil, &stdout, &stderr, tempDir)
	if code != 0 {
		t.Errorf("Expected code 0, got %d. Stderr: %s", code, stderr.String())
	}

	// Check output
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
	code = run([]string{"test.bz2"}, nil, &stdout, &stderr, tempDir)
	if code != 0 {
		t.Errorf("Expected code 0, got %d. Stderr: %s", code, stderr.String())
	}
	// Source should be deleted
	if _, err := os.Stat(srcPath); !os.IsNotExist(err) {
		t.Error("Expected source file to be deleted")
	}

	// Case 3: Suffix check errors
	badSuffix := filepath.Join(tempDir, "test.zz")
	if err := os.WriteFile(badSuffix, helloBz2, 0644); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"test.zz"}, nil, &stdout, &stderr, tempDir)
	if code == 0 {
		t.Error("Expected failure for bad suffix")
	}

	// Case 4: Is a directory check
	subDir := filepath.Join(tempDir, "subdir.bz2")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"subdir.bz2"}, nil, &stdout, &stderr, tempDir)
	if code == 0 {
		t.Error("Expected failure when extracting a directory")
	}

	// Case 5: Destination already exists
	srcPath2 := filepath.Join(tempDir, "test2.bz2")
	if err := os.WriteFile(srcPath2, helloBz2, 0644); err != nil {
		t.Fatal(err)
	}
	destPath2 := filepath.Join(tempDir, "test2")
	if err := os.WriteFile(destPath2, []byte("exist"), 0644); err != nil {
		t.Fatal(err)
	}

	stdout.Reset()
	stderr.Reset()
	code = run([]string{"test2.bz2"}, nil, &stdout, &stderr, tempDir)
	if code == 0 {
		t.Error("Expected failure since destination already exists")
	}

	// Case 6: Force overwrite
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"-f", "test2.bz2"}, nil, &stdout, &stderr, tempDir)
	if code != 0 {
		t.Errorf("Expected code 0 with force option, got %d. Stderr: %s", code, stderr.String())
	}

	// Case 7: TBZ2 suffix
	tbz2Path := filepath.Join(tempDir, "archive.tbz2")
	if err := os.WriteFile(tbz2Path, helloBz2, 0644); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"archive.tbz2"}, nil, &stdout, &stderr, tempDir)
	if code != 0 {
		t.Errorf("Expected code 0, got %d. Stderr: %s", code, stderr.String())
	}
	tbz2Dest := filepath.Join(tempDir, "archive.tar")
	if _, err := os.Stat(tbz2Dest); err != nil {
		t.Error("Expected tbz2 to extract as a .tar file")
	}
}

func TestBunzip2Corrupted(t *testing.T) {
	stdin := bytes.NewReader([]byte("not-a-valid-bzip-header"))
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
