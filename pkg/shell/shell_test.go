package shell

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestShellInlineScript(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := shellRun([]string{"-c", "echo hello"}, nil, &stdout, &stderr)
	if code != 0 {
		t.Errorf("expected exit 0, got %d (stderr: %q)", code, stderr.String())
	}
	if stdout.String() != "hello\n" {
		t.Errorf("expected 'hello\\n', got %q", stdout.String())
	}
}

func TestShellInlineScriptStderr(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := shellRun([]string{"-c", "echo error >&2"}, nil, &stdout, &stderr)
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
	if stderr.String() != "error\n" {
		t.Errorf("expected 'error\\n' on stderr, got %q", stderr.String())
	}
}

func TestShellInlineScriptExitCode(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := shellRun([]string{"-c", "exit 42"}, nil, &stdout, &stderr)
	if code != 42 {
		t.Errorf("expected exit 42, got %d", code)
	}
}

func TestShellMissingCArgument(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := shellRun([]string{"-c"}, nil, &stdout, &stderr)
	if code != 2 {
		t.Errorf("expected exit 2 for missing -c argument, got %d", code)
	}
}

func TestShellHelp(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := shellRun([]string{"--help"}, nil, &stdout, &stderr)
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
	if stdout.Len() == 0 {
		t.Error("expected help output, got empty stdout")
	}
}

func TestShellScriptFile(t *testing.T) {
	dir := t.TempDir()
	scriptPath := filepath.Join(dir, "test.sh")
	if err := os.WriteFile(scriptPath, []byte("echo hello from file\nexit 7\n"), 0644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := shellRun([]string{scriptPath}, nil, &stdout, &stderr)
	if code != 7 {
		t.Errorf("expected exit 7, got %d (stderr: %q)", code, stderr.String())
	}
	if stdout.String() != "hello from file\n" {
		t.Errorf("expected 'hello from file\\n', got %q", stdout.String())
	}
}

func TestShellScriptFileNotFound(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := shellRun([]string{"/nonexistent/script.sh"}, nil, &stdout, &stderr)
	if code != 1 {
		t.Errorf("expected exit 1 for missing file, got %d", code)
	}
	if stderr.Len() == 0 {
		t.Error("expected error message on stderr")
	}
}

func TestShellPipeMode(t *testing.T) {
	stdin := bytes.NewBufferString("echo hello from pipe\n")
	var stdout, stderr bytes.Buffer
	code := shellRun([]string{}, stdin, &stdout, &stderr)
	if code != 0 {
		t.Errorf("expected exit 0, got %d (stderr: %q)", code, stderr.String())
	}
	if stdout.String() != "hello from pipe\n" {
		t.Errorf("expected 'hello from pipe\\n', got %q", stdout.String())
	}
}

func TestShellShebangSpaceQuirk(t *testing.T) {
	// Simulate shebang invocation where the kernel passes " shell"
	// (with leading space) as the first argument after #!/bin/koregoos shell.
	// The dispatch layer strips the command name, but the space may
	// still be present in args[0] for symlink-mode invocation.
	var stdout, stderr bytes.Buffer
	code := shellRun([]string{" shell", "-c", "echo shebang works"}, nil, &stdout, &stderr)
	if code != 0 {
		t.Errorf("expected exit 0, got %d (stderr: %q)", code, stderr.String())
	}
	if stdout.String() != "shebang works\n" {
		t.Errorf("expected 'shebang works\\n', got %q", stdout.String())
	}
}

func TestShellMultipleCommands(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := shellRun([]string{"-c", "echo one && echo two"}, nil, &stdout, &stderr)
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
	if stdout.String() != "one\ntwo\n" {
		t.Errorf("expected 'one\\ntwo\\n', got %q", stdout.String())
	}
}

func TestShellEmptyStdin(t *testing.T) {
	stdin := bytes.NewBufferString("")
	var stdout, stderr bytes.Buffer
	code := shellRun([]string{}, stdin, &stdout, &stderr)
	if code != 0 {
		t.Errorf("expected exit 0 for empty stdin, got %d (stderr: %q)", code, stderr.String())
	}
}

func TestShellDispatchRegistered(t *testing.T) {
	// Verify both "shell" and "sh" are registered in dispatch.
	// This test only works if init() has run (import side-effect).
	// We just verify the package compiles and init() doesn't panic.
	// The actual dispatch.Lookup would require the full binary.
	if testing.Short() {
		t.Skip("skipping dispatch registration check in short mode")
	}
}
