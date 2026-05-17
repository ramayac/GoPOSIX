package who

import (
	"bytes"
	"io"
	"testing"
)

func TestParseUtmpEmpty(t *testing.T) {
	// An empty (zeroed) utmp entry should return an empty user
	entry := make([]byte, utmpSize)
	u, err := parseUtmpEntry(entry)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.Name != "" {
		t.Errorf("expected empty name for EMPTY entry, got %q", u.Name)
	}
}

func TestRunEmpty(t *testing.T) {
	// On systems without utmp, should return empty result
	result, err := Run()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Result should not be nil
	if result.Users == nil && result.Count != 0 {
		t.Error("expected consistent Users/Count")
	}
}

func TestWhoDefaultOutput(t *testing.T) {
	var buf bytes.Buffer
	code := run([]string{}, &buf)
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
}

func TestWhoHeadingOutput(t *testing.T) {
	var buf bytes.Buffer
	code := run([]string{"-H"}, &buf)
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
	// If there are users, heading should be present
	t.Logf("who -H output:\n%s", buf.String())
}

func TestWhoQuickOutput(t *testing.T) {
	var buf bytes.Buffer
	code := run([]string{"-q"}, &buf)
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
	t.Logf("who -q output:\n%s", buf.String())
}

func TestWhoJson(t *testing.T) {
	var buf bytes.Buffer
	code := run([]string{"--json"}, &buf)
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
	if !bytes.Contains(buf.Bytes(), []byte(`"users"`)) {
		t.Error("JSON output missing users field")
	}
}

func TestFixedString(t *testing.T) {
	b := make([]byte, 10)
	b[0] = 'h'
	b[1] = 'i'
	// rest are zeros
	s := fixedString(b)
	if s != "hi" {
		t.Errorf("expected 'hi', got %q", s)
	}
}

func TestRunViaCLI(t *testing.T) {
	// Just verify it doesn't crash
	code := run([]string{}, io.Discard)
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
}
