package yes

import (
	"bytes"
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
