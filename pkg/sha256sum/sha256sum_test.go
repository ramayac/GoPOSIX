package sha256sum

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHashFile(t *testing.T) {
	r := strings.NewReader("hello\n")
	hash, err := HashFile(r)
	if err != nil {
		t.Fatal(err)
	}
	// sha256("hello\n") = 5891b5b522d5df086d0ff0b110fbd9d21bb4fc7163af34d08286a2e846f6be03
	expected := "5891b5b522d5df086d0ff0b110fbd9d21bb4fc7163af34d08286a2e846f6be03"
	if hash != expected {
		t.Errorf("got %q, want %q", hash, expected)
	}
}

func TestHashFileEmpty(t *testing.T) {
	r := strings.NewReader("")
	hash, err := HashFile(r)
	if err != nil {
		t.Fatal(err)
	}
	// sha256("") = e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
	expected := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	if hash != expected {
		t.Errorf("got %q, want %q", hash, expected)
	}
}

func TestRunHashSingleFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(testFile, []byte("hello\n"), 0644)

	var buf bytes.Buffer
	code := run([]string{testFile}, nil, &buf)
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}

	output := buf.String()
	if !strings.Contains(output, "5891b5b522d5df086d0ff0b110fbd9d21bb4fc7163af34d08286a2e846f6be03") {
		t.Errorf("output missing expected hash: %s", output)
	}
	if !strings.Contains(output, testFile) {
		t.Errorf("output missing filename: %s", output)
	}
}

func TestRunHashMultipleFiles(t *testing.T) {
	tmpDir := t.TempDir()
	f1 := filepath.Join(tmpDir, "a.txt")
	f2 := filepath.Join(tmpDir, "b.txt")
	os.WriteFile(f1, []byte("aaa"), 0644)
	os.WriteFile(f2, []byte("bbb"), 0644)

	var buf bytes.Buffer
	code := run([]string{f1, f2}, nil, &buf)
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 2 {
		t.Errorf("expected 2 lines, got %d: %v", len(lines), lines)
	}
}

func TestRunHashNonexistent(t *testing.T) {
	var buf bytes.Buffer
	code := run([]string{"/nonexistent_12345"}, nil, &buf)
	if code != 1 {
		t.Errorf("exit code %d, want 1", code)
	}
}

func TestRunHashJSON(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(testFile, []byte("hello\n"), 0644)

	var buf bytes.Buffer
	code := run([]string{"--json", testFile}, nil, &buf)
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}

	var env map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	data := env["data"].([]interface{})
	if len(data) != 1 {
		t.Fatalf("expected 1 result, got %d", len(data))
	}
	entry := data[0].(map[string]interface{})
	if entry["algorithm"] != "sha256" {
		t.Errorf("algorithm = %v, want sha256", entry["algorithm"])
	}
}

func TestRunCheckMode(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test file
	testFile := filepath.Join(tmpDir, "data.txt")
	os.WriteFile(testFile, []byte("hello\n"), 0644)

	// Create checksum file
	checksumFile := filepath.Join(tmpDir, "checksums.txt")
	os.WriteFile(checksumFile, []byte("5891b5b522d5df086d0ff0b110fbd9d21bb4fc7163af34d08286a2e846f6be03  "+testFile+"\n"), 0644)

	var buf bytes.Buffer
	code := run([]string{"-c", checksumFile}, nil, &buf)
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}

	if !strings.Contains(buf.String(), "OK") {
		t.Errorf("output should contain OK: %s", buf.String())
	}
}

func TestRunCheckModeFailed(t *testing.T) {
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "data.txt")
	os.WriteFile(testFile, []byte("modified content"), 0644)

	// Wrong hash
	checksumFile := filepath.Join(tmpDir, "checksums.txt")
	os.WriteFile(checksumFile, []byte("0000000000000000000000000000000000000000000000000000000000000000  "+testFile+"\n"), 0644)

	var buf bytes.Buffer
	code := run([]string{"-c", checksumFile}, nil, &buf)
	if code != 1 {
		t.Errorf("exit code %d, want 1", code)
	}

	if !strings.Contains(buf.String(), "FAILED") {
		t.Errorf("output should contain FAILED: %s", buf.String())
	}
}

func TestRunCheckNoFile(t *testing.T) {
	var buf bytes.Buffer
	code := run([]string{"-c"}, nil, &buf)
	if code != 1 {
		t.Errorf("exit code %d, want 1", code)
	}
}

func TestRunCheckEmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	checksumFile := filepath.Join(tmpDir, "empty.txt")
	os.WriteFile(checksumFile, []byte("# just a comment\n\n"), 0644)

	var buf bytes.Buffer
	code := run([]string{"-c", checksumFile}, nil, &buf)
	if code != 1 {
		t.Errorf("exit code %d, want 1 for empty checksum file", code)
	}
}

func TestRunCheckBadFormat(t *testing.T) {
	tmpDir := t.TempDir()
	checksumFile := filepath.Join(tmpDir, "bad.txt")
	os.WriteFile(checksumFile, []byte("this line has no spaces\n"), 0644)

	var buf bytes.Buffer
	code := run([]string{"-c", checksumFile}, nil, &buf)
	if code != 1 {
		t.Errorf("exit code %d, want 1 for bad format", code)
	}
}

func TestRunCheckJSON(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "data.txt")
	os.WriteFile(testFile, []byte("hello\n"), 0644)
	checksumFile := filepath.Join(tmpDir, "checksums.txt")
	os.WriteFile(checksumFile, []byte("5891b5b522d5df086d0ff0b110fbd9d21bb4fc7163af34d08286a2e846f6be03  "+testFile+"\n"), 0644)

	var buf bytes.Buffer
	code := run([]string{"-c", "--json", checksumFile}, nil, &buf)
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}
	if !bytes.Contains(buf.Bytes(), []byte("\"status\"")) {
		t.Error("JSON output missing status field")
	}
}

func TestRunHashUnknownFlag(t *testing.T) {
	var buf bytes.Buffer
	code := run([]string{"--bad-flag"}, nil, &buf)
	if code != 1 {
		t.Errorf("exit code %d, want 1 for unknown flag", code)
	}
}
