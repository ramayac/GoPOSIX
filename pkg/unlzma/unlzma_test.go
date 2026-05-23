package unlzma

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/ulikunitz/xz/lzma"
)

// Helper function to compress input to LZMA bytes programmatically.
func compressLZMA(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	w, err := lzma.NewWriter(&buf)
	if err != nil {
		return nil, err
	}
	if _, err := w.Write(data); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func TestUnlzmaStdin(t *testing.T) {
	compressed, err := compressLZMA([]byte("HELLO LZMA\n"))
	if err != nil {
		t.Fatal(err)
	}

	stdin := bytes.NewReader(compressed)
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{}, stdin, &stdout, &stderr, "")
	if code != 0 {
		t.Errorf("Expected exit code 0, got %d. Stderr: %s", code, stderr.String())
	}

	if stdout.String() != "HELLO LZMA\n" {
		t.Errorf("Expected 'HELLO LZMA\\n', got %q", stdout.String())
	}
}

func TestUnlzmaHelp(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"--help"}, nil, &stdout, &stderr, "")
	if code != 0 {
		t.Errorf("Expected code 0, got %d", code)
	}
	if !bytes.Contains(stdout.Bytes(), []byte("Usage: unlzma")) {
		t.Errorf("Expected help output, got: %s", stdout.String())
	}
}

func TestUnlzmaFlagError(t *testing.T) {
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

func TestUnlzmaFiles(t *testing.T) {
	compressed, err := compressLZMA([]byte("FILE DATA\n"))
	if err != nil {
		t.Fatal(err)
	}

	tempDir, err := os.MkdirTemp("", "unlzma-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	srcPath := filepath.Join(tempDir, "test.lzma")
	if err := os.WriteFile(srcPath, compressed, 0644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	// Case 1: Keep mode
	code := run([]string{"-k", "test.lzma"}, nil, &stdout, &stderr, tempDir)
	if code != 0 {
		t.Errorf("Expected code 0, got %d. Stderr: %s", code, stderr.String())
	}

	destPath := filepath.Join(tempDir, "test")
	content, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "FILE DATA\n" {
		t.Errorf("Expected 'FILE DATA\\n', got %q", string(content))
	}

	// Ensure source file remains (keepMode)
	if _, err := os.Stat(srcPath); err != nil {
		t.Error("Expected source file to remain under -k flag")
	}

	// Case 2: Source deletion (default)
	os.Remove(destPath)
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"test.lzma"}, nil, &stdout, &stderr, tempDir)
	if code != 0 {
		t.Errorf("Expected code 0, got %d. Stderr: %s", code, stderr.String())
	}
	if _, err := os.Stat(srcPath); !os.IsNotExist(err) {
		t.Error("Expected source file to be deleted")
	}

	// Case 3: Suffix check errors
	badSuffix := filepath.Join(tempDir, "test.zz")
	if err := os.WriteFile(badSuffix, compressed, 0644); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"test.zz"}, nil, &stdout, &stderr, tempDir)
	if code == 0 {
		t.Error("Expected failure for bad suffix")
	}

	// Case 4: Is a directory check
	subDir := filepath.Join(tempDir, "subdir.lzma")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"subdir.lzma"}, nil, &stdout, &stderr, tempDir)
	if code == 0 {
		t.Error("Expected failure when extracting a directory")
	}

	// Case 5: Destination already exists
	srcPath2 := filepath.Join(tempDir, "test2.lzma")
	if err := os.WriteFile(srcPath2, compressed, 0644); err != nil {
		t.Fatal(err)
	}
	destPath2 := filepath.Join(tempDir, "test2")
	if err := os.WriteFile(destPath2, []byte("exist"), 0644); err != nil {
		t.Fatal(err)
	}

	stdout.Reset()
	stderr.Reset()
	code = run([]string{"test2.lzma"}, nil, &stdout, &stderr, tempDir)
	if code == 0 {
		t.Error("Expected failure since destination already exists")
	}

	// Case 6: Force overwrite
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"-f", "test2.lzma"}, nil, &stdout, &stderr, tempDir)
	if code != 0 {
		t.Errorf("Expected code 0 with force option, got %d. Stderr: %s", code, stderr.String())
	}

	// Case 7: Nonexistent file
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"nonexistent.lzma"}, nil, &stdout, &stderr, tempDir)
	if code == 0 {
		t.Error("Expected failure for nonexistent file")
	}
}

func TestUnlzmaCorrupted(t *testing.T) {
	stdin := bytes.NewReader([]byte("not-a-valid-lzma-header"))
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
