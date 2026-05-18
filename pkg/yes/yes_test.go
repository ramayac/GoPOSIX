package yes

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"testing"
)

// testWriter wraps a bytes.Buffer and stops writing after maxLines.
type testWriter struct {
	buf      bytes.Buffer
	lines    int
	maxLines int
}

func (w *testWriter) Write(p []byte) (int, error) {
	if w.lines >= w.maxLines {
		return 0, io.ErrClosedPipe
	}
	n := len(p)
	w.buf.Write(p)
	w.lines += strings.Count(string(p), "\n")
	return n, nil
}

func TestDefaultOutput(t *testing.T) {
	w := &testWriter{maxLines: 5}
	code := run(nil, w)
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
	lines := strings.Split(strings.TrimSpace(w.buf.String()), "\n")
	if len(lines) != 5 {
		t.Errorf("expected 5 lines, got %d", len(lines))
	}
	for _, line := range lines {
		if line != "y" {
			t.Errorf("expected 'y', got %q", line)
		}
	}
}

func TestCustomString(t *testing.T) {
	w := &testWriter{maxLines: 3}
	code := run([]string{"hello"}, w)
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
	for _, line := range strings.Split(strings.TrimSpace(w.buf.String()), "\n") {
		if line != "hello" {
			t.Errorf("expected 'hello', got %q", line)
		}
	}
}

func TestMultiWordString(t *testing.T) {
	w := &testWriter{maxLines: 3}
	code := run([]string{"foo", "bar"}, w)
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
	for _, line := range strings.Split(strings.TrimSpace(w.buf.String()), "\n") {
		if line != "foo bar" {
			t.Errorf("expected 'foo bar', got %q", line)
		}
	}
}

func TestYesJSON(t *testing.T) {
	var out bytes.Buffer
	code := run([]string{"--json"}, &out)
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}
	// Output should be pure JSON only
	var env map[string]interface{}
	if err := json.Unmarshal(out.Bytes(), &env); err != nil {
		t.Fatalf("invalid JSON: %v (%s)", err, out.String())
	}
	data := env["data"].(map[string]interface{})
	if data["count"].(float64) != 1 {
		t.Errorf("count %v, want 1", data["count"])
	}
	if data["truncated"] != true {
		t.Errorf("truncated %v, want true", data["truncated"])
	}
	if data["string"] != "y" {
		t.Errorf("string %v, want 'y'", data["string"])
	}
}

func TestYesJSONWithCount(t *testing.T) {
	var out bytes.Buffer
	code := run([]string{"--json", "--count", "3"}, &out)
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}
	var env map[string]interface{}
	if err := json.Unmarshal(out.Bytes(), &env); err != nil {
		t.Fatalf("invalid JSON: %v (%s)", err, out.String())
	}
	data := env["data"].(map[string]interface{})
	if data["count"].(float64) != 3 {
		t.Errorf("count %v, want 3", data["count"])
	}
}

func TestYesJSONWithString(t *testing.T) {
	var out bytes.Buffer
	code := run([]string{"--json", "hello"}, &out)
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}
	var env map[string]interface{}
	if err := json.Unmarshal(out.Bytes(), &env); err != nil {
		t.Fatalf("invalid JSON: %v (%s)", err, out.String())
	}
	data := env["data"].(map[string]interface{})
	if data["string"] != "hello" {
		t.Errorf("string %v, want 'hello'", data["string"])
	}
}
