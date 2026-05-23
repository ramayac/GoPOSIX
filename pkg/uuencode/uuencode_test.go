package uuencode

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestUuencodeHelp(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"-h"}, nil, &stdout, &stderr, "")
	if code != 0 {
		t.Errorf("Expected code 0, got %d", code)
	}
	if !bytes.Contains(stdout.Bytes(), []byte("Usage: uuencode")) {
		t.Errorf("Expected help output, got: %s", stdout.String())
	}
}

func TestUuencodeMissingArg(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{}, nil, &stdout, &stderr, "")
	if code == 0 {
		t.Error("Expected non-zero exit code for missing arguments")
	}

	stdout.Reset()
	stderr.Reset()
	code = run([]string{"--json"}, nil, &stdout, &stderr, "")
	if code == 0 {
		t.Error("Expected non-zero exit code in JSON mode for missing arguments")
	}
}

func TestUuencodeFlagError(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"--invalid-flag", "FILE"}, nil, &stdout, &stderr, "")
	if code == 0 {
		t.Error("Expected non-zero exit code on invalid flag")
	}
}

func TestUuencodeTraditional(t *testing.T) {
	stdin := bytes.NewReader([]byte("A"))
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"FILE"}, stdin, &stdout, &stderr, "")
	if code != 0 {
		t.Fatalf("Expected code 0, got %d. Stderr: %s", code, stderr.String())
	}

	expected := "begin 644 FILE\n!00``\n`\nend\n"
	if stdout.String() != expected {
		t.Errorf("Expected %q, got %q", expected, stdout.String())
	}
}

func TestUuencodeBase64(t *testing.T) {
	stdin := bytes.NewReader([]byte("AB"))
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"-m", "FILE"}, stdin, &stdout, &stderr, "")
	if code != 0 {
		t.Fatalf("Expected code 0, got %d. Stderr: %s", code, stderr.String())
	}

	expected := "begin-base64 644 FILE\nQUI=\n====\n"
	if stdout.String() != expected {
		t.Errorf("Expected %q, got %q", expected, stdout.String())
	}
}

func TestUuencodeFromFile(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "uuencode-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	filePath := filepath.Join(tempDir, "input.txt")
	if err := os.WriteFile(filePath, []byte("ABC"), 0644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"input.txt", "REMOTE"}, nil, &stdout, &stderr, tempDir)
	if code != 0 {
		t.Fatalf("Expected code 0, got %d. Stderr: %s", code, stderr.String())
	}

	// We wrote the file with 0644 mode, so the output should reflect that
	expectedPrefix := "begin 644 REMOTE\n"
	if !bytes.HasPrefix(stdout.Bytes(), []byte(expectedPrefix)) {
		t.Errorf("Expected prefix %q, got:\n%s", expectedPrefix, stdout.String())
	}

	// Case: missing file
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"nonexistent.txt", "REMOTE"}, nil, &stdout, &stderr, tempDir)
	if code == 0 {
		t.Error("Expected error for nonexistent input file")
	}

	// Case: JSON mode with file input
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"--json", "input.txt", "REMOTE"}, nil, &stdout, &stderr, tempDir)
	if code != 0 {
		t.Fatal(code)
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"encodedData"`)) || !bytes.Contains(stdout.Bytes(), []byte(`"uuencode"`)) {
		t.Errorf("Expected JSON uuencode output, got:\n%s", stdout.String())
	}

	// Case: JSON mode base64
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"--json", "-m", "input.txt", "REMOTE"}, nil, &stdout, &stderr, tempDir)
	if code != 0 {
		t.Fatal(code)
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"encodedData"`)) || !bytes.Contains(stdout.Bytes(), []byte(`"base64"`)) {
		t.Errorf("Expected JSON base64 output, got:\n%s", stdout.String())
	}
}
