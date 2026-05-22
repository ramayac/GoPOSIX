package md5sum

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
	// md5("hello\n") = b1946ac92492d2347c6235b4d2611184
	expected := "b1946ac92492d2347c6235b4d2611184"
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
	// md5("") = d41d8cd98f00b204e9800998ecf8427e
	expected := "d41d8cd98f00b204e9800998ecf8427e"
	if hash != expected {
		t.Errorf("got %q, want %q", hash, expected)
	}
}

func TestRunHashSingleFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(testFile, []byte("hello\n"), 0644)

	var buf bytes.Buffer
	code := run([]string{testFile}, nil, &buf, &buf, "")
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}

	output := buf.String()
	if !strings.Contains(output, "b1946ac92492d2347c6235b4d2611184") {
		t.Errorf("output missing expected hash: %s", output)
	}
}

func TestRunHashJSON(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(testFile, []byte("hello\n"), 0644)

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
	entry := data[0].(map[string]interface{})
	if entry["algorithm"] != "md5" {
		t.Errorf("algorithm = %v, want md5", entry["algorithm"])
	}
}

func TestRunCheckMode(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "data.txt")
	os.WriteFile(testFile, []byte("hello\n"), 0644)

	checksumFile := filepath.Join(tmpDir, "checksums.txt")
	os.WriteFile(checksumFile, []byte("b1946ac92492d2347c6235b4d2611184  "+testFile+"\n"), 0644)

	var buf bytes.Buffer
	code := run([]string{"-c", checksumFile}, nil, &buf, &buf, "")
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
	os.WriteFile(testFile, []byte("modified"), 0644)

	checksumFile := filepath.Join(tmpDir, "checksums.txt")
	os.WriteFile(checksumFile, []byte("00000000000000000000000000000000  "+testFile+"\n"), 0644)

	var buf bytes.Buffer
	code := run([]string{"-c", checksumFile}, nil, &buf, &buf, "")
	if code != 1 {
		t.Errorf("exit code %d, want 1", code)
	}
}

func TestRunNonexistent(t *testing.T) {
	var buf bytes.Buffer
	code := run([]string{"/nonexistent_12345"}, nil, &buf, &buf, "")
	if code != 1 {
		t.Errorf("exit code %d, want 1", code)
	}
}

func TestRunMultipleFiles(t *testing.T) {
	tmpDir := t.TempDir()
	f1 := filepath.Join(tmpDir, "a.txt")
	f2 := filepath.Join(tmpDir, "b.txt")
	os.WriteFile(f1, []byte("aaa"), 0644)
	os.WriteFile(f2, []byte("bbb"), 0644)

	var buf bytes.Buffer
	code := run([]string{f1, f2}, nil, &buf, &buf, "")
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 2 {
		t.Errorf("expected 2 lines, got %d", len(lines))
	}
}

func TestRunCheckEmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	checksumFile := filepath.Join(tmpDir, "empty.txt")
	os.WriteFile(checksumFile, []byte("# comment\n"), 0644)
	var buf bytes.Buffer
	code := run([]string{"-c", checksumFile}, nil, &buf, &buf, "")
	if code != 1 {
		t.Errorf("exit code %d, want 1 for empty checksum file", code)
	}
}

func TestRunCheckBadFormat(t *testing.T) {
	tmpDir := t.TempDir()
	checksumFile := filepath.Join(tmpDir, "bad.txt")
	os.WriteFile(checksumFile, []byte("nospaces\n"), 0644)
	var buf bytes.Buffer
	code := run([]string{"-c", checksumFile}, nil, &buf, &buf, "")
	if code != 1 {
		t.Errorf("exit code %d, want 1 for bad format", code)
	}
}

func TestRunCheckJSON(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "data.txt")
	os.WriteFile(testFile, []byte("hello\n"), 0644)
	checksumFile := filepath.Join(tmpDir, "checksums.txt")
	os.WriteFile(checksumFile, []byte("b1946ac92492d2347c6235b4d2611184  "+testFile+"\n"), 0644)
	var buf bytes.Buffer
	code := run([]string{"-c", "--json", checksumFile}, nil, &buf, &buf, "")
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}
	if !bytes.Contains(buf.Bytes(), []byte("\"status\"")) {
		t.Error("JSON output missing status field")
	}
}

func TestRunHashUnknownFlag(t *testing.T) {
	var buf bytes.Buffer
	code := run([]string{"--bad-flag"}, nil, &buf, &buf, "")
	if code != 1 {
		t.Errorf("exit code %d, want 1 for unknown flag", code)
	}
}

func TestRunCheckNoFile(t *testing.T) {
	var buf bytes.Buffer
	code := run([]string{"-c"}, nil, &buf, &buf, "")
	if code != 1 {
		t.Errorf("exit code %d, want 1 for missing checksum file", code)
	}
}
