package logname

import (
	"bytes"
	"io"
	"os"
	"testing"
)

func TestLogname(t *testing.T) {
	// Save and set LOGNAME for test
	orig := os.Getenv("LOGNAME")
	os.Setenv("LOGNAME", "testuser")
	defer os.Setenv("LOGNAME", orig)

	var buf bytes.Buffer
	code := run([]string{}, nil, &buf, &buf, "")
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	if buf.String() != "testuser\n" {
		t.Errorf("expected 'testuser\\n', got %q", buf.String())
	}
}

func TestLognameNoEnv(t *testing.T) {
	orig := os.Getenv("LOGNAME")
	os.Unsetenv("LOGNAME")
	defer os.Setenv("LOGNAME", orig)

	code := run([]string{}, nil, io.Discard, io.Discard, "")
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
}

func TestLognameJson(t *testing.T) {
	orig := os.Getenv("LOGNAME")
	os.Setenv("LOGNAME", "testuser")
	defer os.Setenv("LOGNAME", orig)

	var buf bytes.Buffer
	code := run([]string{"--json"}, nil, &buf, &buf, "")
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	if !bytes.Contains(buf.Bytes(), []byte(`"logname"`)) {
		t.Error("JSON output missing logname field")
	}
}
