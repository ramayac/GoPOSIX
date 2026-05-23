package uudecode

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestUudecodeHelp(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"-h"}, nil, &stdout, &stderr, "")
	if code != 0 {
		t.Errorf("Expected code 0, got %d", code)
	}
	if !bytes.Contains(stdout.Bytes(), []byte("Usage: uudecode")) {
		t.Errorf("Expected help output, got: %s", stdout.String())
	}
}

func TestUudecodeFlagError(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"--invalid-flag"}, nil, &stdout, &stderr, "")
	if code == 0 {
		t.Error("Expected non-zero exit code on invalid flag")
	}

	stdout.Reset()
	stderr.Reset()
	code = run([]string{"--json", "--invalid-flag"}, nil, &stdout, &stderr, "")
	if code == 0 {
		t.Error("Expected non-zero exit code on invalid flag in JSON mode")
	}
}

func TestUudecodeBadUsage(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"FILE1", "FILE2"}, nil, &stdout, &stderr, "")
	if code == 0 {
		t.Error("Expected non-zero exit code for too many arguments")
	}

	stdout.Reset()
	stderr.Reset()
	code = run([]string{"--json", "FILE1", "FILE2"}, nil, &stdout, &stderr, "")
	if code == 0 {
		t.Error("Expected non-zero exit code in JSON mode for too many arguments")
	}
}

func TestUudecodeMissingHeader(t *testing.T) {
	stdin := bytes.NewReader([]byte("not-a-valid-uuencoded-file"))
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{}, stdin, &stdout, &stderr, "")
	if code == 0 {
		t.Error("Expected non-zero exit code for missing header")
	}

	stdin.Seek(0, 0)
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"--json"}, stdin, &stdout, &stderr, "")
	if code == 0 {
		t.Error("Expected non-zero exit code in JSON mode for missing header")
	}
}

func TestUudecodeInvalidHeader(t *testing.T) {
	stdin := bytes.NewReader([]byte("begin\n"))
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{}, stdin, &stdout, &stderr, "")
	if code == 0 {
		t.Error("Expected non-zero exit code for invalid header format")
	}

	stdin.Seek(0, 0)
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"--json"}, stdin, &stdout, &stderr, "")
	if code == 0 {
		t.Error("Expected non-zero exit code in JSON mode for invalid header format")
	}
}

func TestUudecodeTraditional(t *testing.T) {
	input := "begin 644 OUTFILE\n!00``\n`\nend\n"
	stdin := bytes.NewReader([]byte(input))
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"-o", "-"}, stdin, &stdout, &stderr, "")
	if code != 0 {
		t.Fatalf("Expected code 0, got %d. Stderr: %s", code, stderr.String())
	}

	if stdout.String() != "A" {
		t.Errorf("Expected 'A', got %q", stdout.String())
	}
}

func TestUudecodeBase64(t *testing.T) {
	input := "begin-base64 644 OUTFILE\nQUI=\n====\n"
	stdin := bytes.NewReader([]byte(input))
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"-o", "-"}, stdin, &stdout, &stderr, "")
	if code != 0 {
		t.Fatalf("Expected code 0, got %d. Stderr: %s", code, stderr.String())
	}

	if stdout.String() != "AB" {
		t.Errorf("Expected 'AB', got %q", stdout.String())
	}
}

func TestUudecodeToFileAndJSON(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "uudecode-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	input := "begin 600 custom.txt\n#04)#\n`\nend\n"
	stdin := bytes.NewReader([]byte(input))

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"--json"}, stdin, &stdout, &stderr, tempDir)
	if code != 0 {
		t.Fatalf("Expected code 0, got %d. Stderr: %s", code, stderr.String())
	}

	destPath := filepath.Join(tempDir, "custom.txt")
	data, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatal(err)
	}

	if string(data) != "ABC" {
		t.Errorf("Expected 'ABC', got %q", string(data))
	}

	if !bytes.Contains(stdout.Bytes(), []byte(`"bytesDecoded":3`)) {
		t.Errorf("Expected json response in stdout, got:\n%s", stdout.String())
	}
}

func TestUudecodeFromFile(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "uudecode-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	uuPath := filepath.Join(tempDir, "encoded.uu")
	input := "begin-base64 644 file.txt\nQUJD\n====\n"
	if err := os.WriteFile(uuPath, []byte(input), 0644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"-o", "decoded.txt", "encoded.uu"}, nil, &stdout, &stderr, tempDir)
	if code != 0 {
		t.Fatalf("Expected code 0, got %d. Stderr: %s", code, stderr.String())
	}

	decPath := filepath.Join(tempDir, "decoded.txt")
	data, err := os.ReadFile(decPath)
	if err != nil {
		t.Fatal(err)
	}

	if string(data) != "ABC" {
		t.Errorf("Expected 'ABC', got %q", string(data))
	}

	// Non-existent file
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"nonexistent.uu"}, nil, &stdout, &stderr, tempDir)
	if code == 0 {
		t.Error("Expected non-zero exit code for nonexistent input file")
	}

	stdout.Reset()
	stderr.Reset()
	code = run([]string{"--json", "nonexistent.uu"}, nil, &stdout, &stderr, tempDir)
	if code == 0 {
		t.Error("Expected non-zero exit code for nonexistent input file in JSON mode")
	}
}
