package logname

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/user"
	"strings"
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

func TestLognameBadFlag(t *testing.T) {
	var buf bytes.Buffer
	if code := run([]string{"--badflag"}, nil, &buf, &buf, ""); code != 2 {
		t.Errorf("expected exit 2, got %d", code)
	}
}

func TestCLIRunError(t *testing.T) {
	t.Setenv("LOGNAME", "")
	orig := userCurrent
	defer func() { userCurrent = orig }()
	userCurrent = func() (*user.User, error) {
		return nil, fmt.Errorf("no user")
	}
	var out, errBuf bytes.Buffer
	code := run([]string{}, nil, &out, &errBuf, "")
	if code != 1 {
		t.Errorf("expected exit 1, got %d", code)
	}
	if !strings.Contains(errBuf.String(), "logname:") {
		t.Errorf("expected stderr message, got %q", errBuf.String())
	}
}
