package sha512sum

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type errorReader struct{}

func (e errorReader) Read(p []byte) (int, error) {
	return 0, errors.New("mock read failure")
}

func TestHashFile(t *testing.T) {
	r := strings.NewReader("")
	hash, err := HashFile(r)
	if err != nil {
		t.Fatal(err)
	}
	expected := "cf83e1357eefb8bdf1542850d66d8007d620e4050b5715dc83f4a921d36ce9ce47d0d13c5d85f2b0ff8318d2877eec2f63b931bd47417a81a538327af927da3e"
	if hash != expected {
		t.Errorf("got %q, want %q", hash, expected)
	}
}

func TestHashFileError(t *testing.T) {
	_, err := HashFile(errorReader{})
	if err == nil {
		t.Error("expected error from failing reader")
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
	if !strings.Contains(output, "cf83e1357eefb8bdf1542850d66d8007d620e4050b5715dc83f4a921d36ce9ce47d0d13c5d85f2b0ff8318d2877eec2f63b931bd47417a81a538327af927da3e") {
		t.Errorf("output missing expected hash: %s", output)
	}
}

func TestRunHashStdin(t *testing.T) {
	var buf bytes.Buffer
	stdinReader := strings.NewReader("hello")
	code := run([]string{"-"}, stdinReader, &buf, &buf, "")
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}
	if !strings.Contains(buf.String(), "9b71d224bd62f3785d96d46ad3ea3d73319bfbc2890caadae2dff72519673ca72323c3d99ba5c11d7c7acc6e14b8c5da0c4663475c2e5c3adef46f73bcdec043") {
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
	if item["hash"] != "cf83e1357eefb8bdf1542850d66d8007d620e4050b5715dc83f4a921d36ce9ce47d0d13c5d85f2b0ff8318d2877eec2f63b931bd47417a81a538327af927da3e" {
		t.Errorf("unexpected JSON hash: %v", item["hash"])
	}
}

func TestCheck(t *testing.T) {
	tmpDir := t.TempDir()
	target := filepath.Join(tmpDir, "f.txt")
	os.WriteFile(target, []byte(""), 0644)

	checkFile := filepath.Join(tmpDir, "check.txt")
	os.WriteFile(checkFile, []byte("cf83e1357eefb8bdf1542850d66d8007d620e4050b5715dc83f4a921d36ce9ce47d0d13c5d85f2b0ff8318d2877eec2f63b931bd47417a81a538327af927da3e  "+target+"\n"), 0644)

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
	os.WriteFile(mismatchCheck, []byte("cf83e1357eefb8bdf1542850d66d8007d620e4050b5715dc83f4a921d36ce9ce47d0d13c5d85f2b0ff8318d2877eec2f63b931bd47417a81a538327af927da3e  "+target+"\n"), 0644)
	buf.Reset()
	code = run([]string{"-c", mismatchCheck}, nil, &buf, &buf, "")
	if code != 1 {
		t.Errorf("expected exit 1 for mismatch, got %d", code)
	}

	// Nonexistent target file inside check
	noTargetCheck := filepath.Join(tmpDir, "notarget.txt")
	os.WriteFile(noTargetCheck, []byte("cf83e1357eefb8bdf1542850d66d8007d620e4050b5715dc83f4a921d36ce9ce47d0d13c5d85f2b0ff8318d2877eec2f63b931bd47417a81a538327af927da3e  /nonexistent_target_file\n"), 0644)
	buf.Reset()
	code = run([]string{"-c", noTargetCheck}, nil, &buf, &buf, "")
	if code != 1 {
		t.Errorf("expected exit 1, got %d", code)
	}
}
