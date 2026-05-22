package sha1sum

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHashFile(t *testing.T) {
	r := strings.NewReader("")
	hash, err := HashFile(r)
	if err != nil {
		t.Fatal(err)
	}
	expected := "da39a3ee5e6b4b0d3255bfef95601890afd80709"
	if hash != expected {
		t.Errorf("got %q, want %q", hash, expected)
	}
}

func TestRunHashSingleFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(testFile, []byte(""), 0644)

	var buf bytes.Buffer
	code := run([]string{testFile}, nil, &buf, &buf, "")
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}

	output := buf.String()
	if !strings.Contains(output, "da39a3ee5e6b4b0d3255bfef95601890afd80709") {
		t.Errorf("output missing expected hash: %s", output)
	}
}

func TestRunHashStdin(t *testing.T) {
	var buf bytes.Buffer
	stdinReader := strings.NewReader("hello")
	// sha1("hello") = aaf4c61ddcc5e8a2dabede0f3b482cd9aea9434d
	code := run([]string{"-"}, stdinReader, &buf, &buf, "")
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}
	if !strings.Contains(buf.String(), "aaf4c61ddcc5e8a2dabede0f3b482cd9aea9434d") {
		t.Errorf("expected hash not found: %s", buf.String())
	}
}

func TestRunHashNonexistent(t *testing.T) {
	var buf bytes.Buffer
	code := run([]string{"/nonexistent_12345"}, nil, &buf, &buf, "")
	if code != 1 {
		t.Errorf("exit code %d, want 1", code)
	}
}

func TestRunHashJSON(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(testFile, []byte(""), 0644)

	var buf bytes.Buffer
	code := run([]string{"--json", testFile}, nil, &buf, &buf, "")
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}

	var env map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	data := env["data"].([]interface{})
	item := data[0].(map[string]interface{})
	if item["hash"] != "da39a3ee5e6b4b0d3255bfef95601890afd80709" {
		t.Errorf("unexpected JSON hash: %v", item["hash"])
	}
}

func TestCheck(t *testing.T) {
	tmpDir := t.TempDir()
	target := filepath.Join(tmpDir, "f.txt")
	os.WriteFile(target, []byte(""), 0644)

	checkFile := filepath.Join(tmpDir, "check.txt")
	os.WriteFile(checkFile, []byte("da39a3ee5e6b4b0d3255bfef95601890afd80709  "+target+"\n"), 0644)

	var buf bytes.Buffer
	code := run([]string{"-c", checkFile}, nil, &buf, &buf, "")
	if code != 0 {
		t.Fatalf("exit code %d, want 0, output: %s", code, buf.String())
	}
	if !strings.Contains(buf.String(), "OK") {
		t.Errorf("expected OK status in check output: %s", buf.String())
	}
}

func TestCheckErrors(t *testing.T) {
	tmpDir := t.TempDir()
	var buf bytes.Buffer

	// Flag error
	code := run([]string{"--invalid-flag"}, nil, &buf, &buf, "")
	if code != 1 {
		t.Errorf("expected exit 1, got %d", code)
	}

	// Missing check file
	buf.Reset()
	code = run([]string{"-c"}, nil, &buf, &buf, "")
	if code != 1 {
		t.Errorf("expected exit 1, got %d", code)
	}

	// Nonexistent check file
	buf.Reset()
	code = run([]string{"-c", "/nonexistent_check_file"}, nil, &buf, &buf, "")
	if code != 1 {
		t.Errorf("expected exit 1, got %d", code)
	}

	// Empty check file
	emptyCheck := filepath.Join(tmpDir, "empty.txt")
	os.WriteFile(emptyCheck, []byte(""), 0644)
	buf.Reset()
	code = run([]string{"-c", emptyCheck}, nil, &buf, &buf, "")
	if code != 1 {
		t.Errorf("expected exit 1, got %d", code)
	}

	// Improperly formatted line
	badCheck := filepath.Join(tmpDir, "bad.txt")
	os.WriteFile(badCheck, []byte("somehash_without_spaces\n"), 0644)
	buf.Reset()
	code = run([]string{"-c", badCheck}, nil, &buf, &buf, "")
	if code != 1 {
		t.Errorf("expected exit 1, got %d", code)
	}

	// Failed target check (hash mismatch)
	target := filepath.Join(tmpDir, "target.txt")
	os.WriteFile(target, []byte("different content"), 0644)
	mismatchCheck := filepath.Join(tmpDir, "mismatch.txt")
	os.WriteFile(mismatchCheck, []byte("da39a3ee5e6b4b0d3255bfef95601890afd80709  "+target+"\n"), 0644)
	buf.Reset()
	code = run([]string{"-c", mismatchCheck}, nil, &buf, &buf, "")
	if code != 1 {
		t.Errorf("expected exit 1 for mismatch, got %d", code)
	}

	// Nonexistent target file inside check
	noTargetCheck := filepath.Join(tmpDir, "notarget.txt")
	os.WriteFile(noTargetCheck, []byte("da39a3ee5e6b4b0d3255bfef95601890afd80709  /nonexistent_target_file\n"), 0644)
	buf.Reset()
	code = run([]string{"-c", noTargetCheck}, nil, &buf, &buf, "")
	if code != 1 {
		t.Errorf("expected exit 1, got %d", code)
	}
}
