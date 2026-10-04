package whoami

import (
	"bytes"
	"fmt"
	"os/user"
	"strings"
	"testing"
)

func TestRunReturnsUser(t *testing.T) {
	result, err := Run()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.User == "" {
		t.Error("expected non-empty username")
	}
	if result.UID < 0 {
		t.Errorf("expected UID >= 0, got %d", result.UID)
	}
}

func TestRunCLI(t *testing.T) {
	var buf bytes.Buffer
	code := run([]string{}, nil, &buf, &buf, "")
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
	if buf.Len() == 0 {
		t.Error("expected output")
	}
}

func TestRunCLIJSON(t *testing.T) {
	var buf bytes.Buffer
	code := run([]string{"--json"}, nil, &buf, &buf, "")
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
	if buf.Len() == 0 {
		t.Error("expected JSON output")
	}
	if !bytes.Contains(buf.Bytes(), []byte("\"user\"")) {
		t.Error("JSON output missing 'user' field")
	}
}

func TestRunCLI_UnknownFlag(t *testing.T) {
	var buf bytes.Buffer
	code := run([]string{"--no-such-flag"}, nil, &buf, &buf, "")
	if code != 2 {
		t.Errorf("expected exit 2 for unknown flag, got %d", code)
	}
}

func TestWhoamiJSON(t *testing.T) {
	var buf bytes.Buffer
	code := run([]string{"--json"}, nil, &buf, &buf, "")
	if code != 0 {
		t.Fatalf("whoami --json exit %d", code)
	}
	if !strings.Contains(buf.String(), "\"user\"") {
		t.Errorf("expected user field in JSON, got: %s", buf.String())
	}
}
func TestWhoamiBadFlag(t *testing.T) {
	var buf bytes.Buffer
	code := run([]string{"--no-such-flag"}, nil, &buf, &buf, "")
	if code != 2 {
		t.Errorf("expected exit 2 for unknown flag, got %d", code)
	}
}

func TestRunError(t *testing.T) {
	orig := userCurrent
	defer func() { userCurrent = orig }()
	userCurrent = func() (*user.User, error) {
		return nil, fmt.Errorf("no user")
	}

	if _, err := Run(); err == nil {
		t.Error("expected error from Run")
	}
}

func TestRunCLIErrorText(t *testing.T) {
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
	if !strings.Contains(errBuf.String(), "whoami: no user") {
		t.Errorf("expected stderr message, got %q", errBuf.String())
	}
}

func TestRunCLIErrorJSON(t *testing.T) {
	orig := userCurrent
	defer func() { userCurrent = orig }()
	userCurrent = func() (*user.User, error) {
		return nil, fmt.Errorf("no user")
	}

	var out, errBuf bytes.Buffer
	code := run([]string{"--json"}, nil, &out, &errBuf, "")
	if code != 1 {
		t.Errorf("expected exit 1, got %d", code)
	}
	if !strings.Contains(out.String(), "EUSER") {
		t.Errorf("expected JSON error envelope on stdout, got %q", out.String())
	}
}

func TestRunBadFlag(t *testing.T) {
	var out, errBuf bytes.Buffer
	code := run([]string{"--no-such-flag"}, nil, &out, &errBuf, "")
	if code != 2 {
		t.Errorf("expected exit 2 for bad flag, got %d", code)
	}
	if !strings.Contains(errBuf.String(), "whoami:") {
		t.Errorf("expected stderr message, got %q", errBuf.String())
	}
}
