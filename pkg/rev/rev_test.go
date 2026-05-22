package rev

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReverseRunes(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello", "olleh"},
		{"", ""},
		{"a", "a"},
		{"日本語", "語本日"},
		{"hello world", "dlrow olleh"},
	}

	for _, tc := range tests {
		got := reverseRunes(tc.input)
		if got != tc.expected {
			t.Errorf("reverseRunes(%q) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}

func TestRunRevStdin(t *testing.T) {
	var buf bytes.Buffer
	stdin := strings.NewReader("line 1\n\nline 3\n")
	code := run([]string{}, stdin, &buf, &buf, "")
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}

	expected := "1 enil\n\n3 enil\n"
	if buf.String() != expected {
		t.Errorf("got %q, want %q", buf.String(), expected)
	}
}

func TestRunRevMissingNewline(t *testing.T) {
	var buf bytes.Buffer
	stdin := strings.NewReader("line 1\n\nline 3")
	code := run([]string{}, stdin, &buf, &buf, "")
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}

	expected := "1 enil\n\n3 enil"
	if buf.String() != expected {
		t.Errorf("got %q, want %q", buf.String(), expected)
	}
}

func TestRunRevNulTruncation(t *testing.T) {
	var buf bytes.Buffer
	stdin := strings.NewReader("lin\x00e 1\n\nline 3\n")
	code := run([]string{}, stdin, &buf, &buf, "")
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}

	expected := "nil\n3 enil\n"
	if buf.String() != expected {
		t.Errorf("got %q, want %q", buf.String(), expected)
	}
}

func TestRunRevFiles(t *testing.T) {
	tmpDir := t.TempDir()
	f1 := filepath.Join(tmpDir, "f1.txt")
	f2 := filepath.Join(tmpDir, "f2.txt")

	os.WriteFile(f1, []byte("abc\ndef\n"), 0644)
	os.WriteFile(f2, []byte("123\n456"), 0644)

	var buf bytes.Buffer
	code := run([]string{f1, f2}, nil, &buf, &buf, "")
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}

	expected := "cba\nfed\n321\n654"
	if buf.String() != expected {
		t.Errorf("got %q, want %q", buf.String(), expected)
	}
}

func TestRunRevNonexistentFile(t *testing.T) {
	var buf bytes.Buffer
	code := run([]string{"/nonexistent_file_xyz"}, nil, &buf, &buf, "")
	if code != 1 {
		t.Errorf("exit code %d, want 1", code)
	}
}

func TestRunRevJSON(t *testing.T) {
	var buf bytes.Buffer
	stdin := strings.NewReader("abc\ndef\n")
	code := run([]string{"--json"}, stdin, &buf, &buf, "")
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	data := resp["data"].(map[string]interface{})
	lines := data["lines"].([]interface{})
	if len(lines) != 2 || lines[0] != "cba" || lines[1] != "fed" {
		t.Errorf("unexpected JSON output structure: %v", resp)
	}
}

func TestRunRevInvalidFlag(t *testing.T) {
	var buf bytes.Buffer
	code := run([]string{"--invalid-flag"}, nil, &buf, &buf, "")
	if code != 1 {
		t.Errorf("exit code %d, want 1", code)
	}
}
