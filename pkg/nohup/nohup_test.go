package nohup

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func TestNohupExecute(t *testing.T) {
	dir := t.TempDir()
	// Change to temp dir so nohup.out is created there
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	var buf bytes.Buffer
	code := run([]string{"echo", "hello"}, nil, &buf, &buf, "")
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}

	// Check nohup.out was created (since test stdout may not be a tty,
	// this only applies if stdout IS a tty)
	if data, err := os.ReadFile("nohup.out"); err == nil {
		t.Logf("nohup.out contains: %q", string(data))
	}
	os.Remove("nohup.out")
}

func TestNohupExitCode(t *testing.T) {
	var buf bytes.Buffer
	code := run([]string{"false"}, nil, &buf, &buf, "")
	if code != 1 {
		t.Errorf("expected exit 1 from false, got %d", code)
	}
}

func TestNohupMissingCommand(t *testing.T) {
	code := run([]string{}, nil, io.Discard, io.Discard, "")
	if code != 1 {
		t.Errorf("expected exit 1 for missing command, got %d", code)
	}
}

func TestNohupJson(t *testing.T) {
	var buf bytes.Buffer
	code := run([]string{"--json", "true"}, nil, &buf, &buf, "")
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	if !bytes.Contains(buf.Bytes(), []byte(`"command"`)) {
		t.Error("JSON output missing command field")
	}
}

func TestNohup_BadFlag(t *testing.T) {
	var buf bytes.Buffer
	code := run([]string{"--bad-flag"}, nil, &buf, &buf, "")
	if code != 2 {
		t.Errorf("expected exit 2 for bad flag, got %d", code)
	}
}

func TestRun_EmptyCommand(t *testing.T) {
	_, err := Run([]string{}, nil, io.Discard, io.Discard)
	if err == nil {
		t.Error("expected error for empty command")
	}
}

func TestNohup_JSON_MissingCommand(t *testing.T) {
	var buf bytes.Buffer
	code := run([]string{"--json"}, nil, &buf, &buf, "")
	if code != 1 {
		t.Errorf("expected exit 1 for missing command, got %d", code)
	}
}

func TestNohupStdinRedirection(t *testing.T) {
	var buf bytes.Buffer
	code := run([]string{"echo", "hello"}, strings.NewReader(""), &buf, &buf, "")
	if code != 0 {
		t.Errorf("nohup echo: exit %d", code)
	}
}

func TestNohupWithArgs(t *testing.T) {
	var buf bytes.Buffer
	code := run([]string{"echo", "test123"}, strings.NewReader(""), &buf, &buf, "")
	// echo should output "test123" to stdout which nohup captures
	if code != 0 {
		t.Errorf("nohup echo: exit %d", code)
	}
}
func TestNohupJSONMode(t *testing.T) {
	var buf bytes.Buffer
	code := run([]string{"--json", "echo", "hi"}, strings.NewReader(""), &buf, &buf, "")
	_ = code
}

func TestNohupCommandNotFound(t *testing.T) {
	var out, errBuf bytes.Buffer
	// The error message goes to the injected stderr writer.
	code := run([]string{"nonexistent-command-xyz"}, nil, &out, &errBuf, "")
	if code != 1 {
		t.Errorf("expected exit 1, got %d", code)
	}
}

func TestRunTerminalRedirect(t *testing.T) {
	orig := terminalCheck
	defer func() { terminalCheck = orig }()
	dir := t.TempDir()
	t.Chdir(dir)
	terminalCheck = func(w io.Writer) bool { return true }

	// stdout redirects to nohup.stdout in the cwd; stderr follows stdout.
	var out, errBuf bytes.Buffer
	code := run([]string{"echo", "hi"}, strings.NewReader(""), &out, &errBuf, dir)
	if code != 0 {
		t.Fatalf("exit %d, want 0", code)
	}
	// output lands in dir/nohup.stdout
	b, err := os.ReadFile("nohup.stdout")
	if err != nil {
		t.Fatalf("expected nohup.stdout: %v", err)
	}
	if string(b) != "hi\n" {
		t.Errorf("nohup.stdout = %q, want %q", b, "hi\n")
	}
}
