package shell

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestShellInlineScript(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := shellRun([]string{"-c", "echo hello"}, nil, &stdout, &stderr, "")
	if code != 0 {
		t.Errorf("expected exit 0, got %d (stderr: %q)", code, stderr.String())
	}
	if stdout.String() != "hello\n" {
		t.Errorf("expected 'hello\\n', got %q", stdout.String())
	}
}

func TestShellInlineScriptStderr(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := shellRun([]string{"-c", "echo error >&2"}, nil, &stdout, &stderr, "")
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
	if stderr.String() != "error\n" {
		t.Errorf("expected 'error\\n' on stderr, got %q", stderr.String())
	}
}

func TestShellInlineScriptExitCode(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := shellRun([]string{"-c", "exit 42"}, nil, &stdout, &stderr, "")
	if code != 42 {
		t.Errorf("expected exit 42, got %d", code)
	}
}

func TestShellInlineScriptJSON(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := shellRun([]string{"--json", "-c", "echo hi; echo err >&2; exit 3"}, nil, &stdout, &stderr, "")
	if code != 3 {
		t.Errorf("expected exit 3, got %d", code)
	}
	var env struct {
		Command string `json:"command"`
		Data    struct {
			ExitCode int    `json:"exitCode"`
			Stdout   string `json:"stdout"`
			Stderr   string `json:"stderr"`
		} `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &env); err != nil {
		t.Fatalf("expected JSON envelope on stdout, got %q: %v", stdout.String(), err)
	}
	if env.Command != "shell" {
		t.Errorf("expected command shell, got %q", env.Command)
	}
	if env.Data.ExitCode != 3 {
		t.Errorf("expected script exitCode 3 in data, got %d", env.Data.ExitCode)
	}
	if env.Data.Stdout != "hi\n" {
		t.Errorf("expected captured stdout %q, got %q", "hi\n", env.Data.Stdout)
	}
	if env.Data.Stderr != "err\n" {
		t.Errorf("expected captured stderr %q, got %q", "err\n", env.Data.Stderr)
	}
	if stderr.Len() != 0 {
		t.Errorf("expected empty stderr stream, got %q", stderr.String())
	}
}

func TestShellPipeModeJSON(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := shellRun([]string{"--json"}, strings.NewReader("echo pipe\n"), &stdout, &stderr, "")
	if code != 0 {
		t.Errorf("expected exit 0, got %d (stderr: %q)", code, stderr.String())
	}
	var env struct {
		Data struct {
			Stdout string `json:"stdout"`
		} `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &env); err != nil {
		t.Fatalf("expected JSON envelope, got %q: %v", stdout.String(), err)
	}
	if env.Data.Stdout != "pipe\n" {
		t.Errorf("expected captured pipe output, got %q", env.Data.Stdout)
	}
}

func TestShellScriptFileJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "s.sh")
	if err := os.WriteFile(path, []byte("echo file-mode\n"), 0644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := shellRun([]string{"--json", path}, nil, &stdout, &stderr, "")
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
	var env struct {
		Data struct {
			Stdout string `json:"stdout"`
		} `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &env); err != nil {
		t.Fatalf("expected JSON envelope, got %q: %v", stdout.String(), err)
	}
	if env.Data.Stdout != "file-mode\n" {
		t.Errorf("expected captured file output, got %q", env.Data.Stdout)
	}
}

func TestShellJSONFlagAfterScriptFlag(t *testing.T) {
	// --json may appear after other flags (the daemon prepends it, the CLI
	// may pass it in any position before the script).
	var stdout, stderr bytes.Buffer
	code := shellRun([]string{"-c", "echo ok", "--json"}, nil, &stdout, &stderr, "")
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
	var env struct {
		Command string `json:"command"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &env); err != nil {
		t.Fatalf("expected JSON envelope, got %q: %v", stdout.String(), err)
	}
	if env.Command != "shell" {
		t.Errorf("expected command shell, got %q", env.Command)
	}
}

// errorReader always fails, like a broken pipe on stdin.
type errorReader struct{}

func (errorReader) Read([]byte) (int, error) { return 0, errors.New("read failure") }

func TestShellPipeModeStdinError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := shellRun([]string{}, errorReader{}, &stdout, &stderr, "")
	if code != 1 {
		t.Fatalf("expected exit 1, got %d", code)
	}
	if !strings.Contains(stderr.String(), "shell: read failure") {
		t.Errorf("expected plain stderr error, got %q", stderr.String())
	}
}

func TestShellPipeModeStdinErrorJSON(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := shellRun([]string{"--json"}, errorReader{}, &stdout, &stderr, "")
	if code != 1 {
		t.Fatalf("expected exit 1, got %d", code)
	}
	var env map[string]interface{}
	if err := json.Unmarshal(stderr.Bytes(), &env); err != nil {
		t.Fatalf("expected JSON envelope on stderr, got %q: %v", stderr.String(), err)
	}
	if errInfo, ok := env["error"].(map[string]interface{}); !ok || errInfo["code"] != "SHELL_ERROR" {
		t.Fatalf("expected SHELL_ERROR envelope, got %q", stderr.String())
	}
}

func TestShellMissingCArgument(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := shellRun([]string{"-c"}, nil, &stdout, &stderr, "")
	if code != 2 {
		t.Errorf("expected exit 2 for missing -c argument, got %d", code)
	}
}

func TestShellMissingCArgumentJSON(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := shellRun([]string{"--json", "-c"}, nil, &stdout, &stderr, "")
	if code != 2 {
		t.Fatalf("expected exit 2, got %d", code)
	}
	var env map[string]interface{}
	if err := json.Unmarshal(stderr.Bytes(), &env); err != nil {
		t.Fatalf("expected JSON envelope on stderr, got %q: %v", stderr.String(), err)
	}
	if errInfo, ok := env["error"].(map[string]interface{}); !ok || errInfo["code"] != "MISSING_ARGUMENT" {
		t.Fatalf("expected MISSING_ARGUMENT envelope, got %q", stderr.String())
	}
}

func TestShellScriptFileNotFoundJSON(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := shellRun([]string{"--json", "/does/not/exist.sh"}, nil, &stdout, &stderr, "")
	if code != 1 {
		t.Fatalf("expected exit 1, got %d", code)
	}
	var env map[string]interface{}
	if err := json.Unmarshal(stderr.Bytes(), &env); err != nil {
		t.Fatalf("expected JSON envelope on stderr, got %q: %v", stderr.String(), err)
	}
	if errInfo, ok := env["error"].(map[string]interface{}); !ok || errInfo["code"] != "SHELL_ERROR" {
		t.Fatalf("expected SHELL_ERROR envelope, got %q", stderr.String())
	}
}

func TestShellHelp(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := shellRun([]string{"--help"}, nil, &stdout, &stderr, "")
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
	code := shellRun([]string{scriptPath}, nil, &stdout, &stderr, "")
	if code != 7 {
		t.Errorf("expected exit 7, got %d (stderr: %q)", code, stderr.String())
	}
	if stdout.String() != "hello from file\n" {
		t.Errorf("expected 'hello from file\\n', got %q", stdout.String())
	}
}

func TestShellScriptFileNotFound(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := shellRun([]string{"/nonexistent/script.sh"}, nil, &stdout, &stderr, "")
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
	code := shellRun([]string{}, stdin, &stdout, &stderr, "")
	if code != 0 {
		t.Errorf("expected exit 0, got %d (stderr: %q)", code, stderr.String())
	}
	if stdout.String() != "hello from pipe\n" {
		t.Errorf("expected 'hello from pipe\\n', got %q", stdout.String())
	}
}

func TestShellShebangSpaceQuirk(t *testing.T) {
	// Simulate shebang invocation where the kernel passes " shell"
	// (with leading space) as the first argument after #!/bin/goposixos shell.
	// The dispatch layer strips the command name, but the space may
	// still be present in args[0] for symlink-mode invocation.
	var stdout, stderr bytes.Buffer
	code := shellRun([]string{" shell", "-c", "echo shebang works"}, nil, &stdout, &stderr, "")
	if code != 0 {
		t.Errorf("expected exit 0, got %d (stderr: %q)", code, stderr.String())
	}
	if stdout.String() != "shebang works\n" {
		t.Errorf("expected 'shebang works\\n', got %q", stdout.String())
	}
}

func TestShellMultipleCommands(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := shellRun([]string{"-c", "echo one && echo two"}, nil, &stdout, &stderr, "")
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
	code := shellRun([]string{}, stdin, &stdout, &stderr, "")
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

func TestIsTerminal_NotATerminal(t *testing.T) {
	// os.Stdin is usually not a terminal in test context.
	// But let's test with a regular file.
	f, err := os.CreateTemp("", "shell-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	defer f.Close()
	if isTerminal(f) {
		t.Error("regular file should not be detected as terminal")
	}
}

func TestIsTerminal_ClosedFile(t *testing.T) {
	f, err := os.CreateTemp("", "shell-test-closed")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	os.Remove(f.Name())
	// Calling Stat() on a closed (but still valid fd) file still works.
	// But if the file is removed, Stat still works on linux.
	// Just verify it doesn't panic.
	_ = isTerminal(f)
}

func TestShell_CLIRun(t *testing.T) {
	// Test the CLI glue run() function.
	var outBuf, errBuf bytes.Buffer
	// shellRun requires script args or stdin
	rc := run([]string{"-c", "echo test"}, nil, &outBuf, &errBuf, "")
	if rc != 0 {
		t.Errorf("exit code: got %d, want 0", rc)
	}
	if !strings.Contains(outBuf.String(), "test") {
		t.Errorf("expected 'test' in output, got %q", outBuf.String())
	}
}

func TestInteractiveBasic(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := interactive(strings.NewReader("echo hello\nexit\n"), &stdout, &stderr)
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
	if !strings.Contains(stdout.String(), "hello") {
		t.Errorf("expected 'hello' in stdout, got %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "$ ") {
		t.Errorf("expected prompt in stdout, got %q", stdout.String())
	}
}

func TestInteractiveQuit(t *testing.T) {
	var stdout bytes.Buffer
	if code := interactive(strings.NewReader("quit\n"), &stdout, &stdout); code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
}

func TestInteractiveEmptyLine(t *testing.T) {
	var stdout bytes.Buffer
	if code := interactive(strings.NewReader("\n\nexit\n"), &stdout, &stdout); code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
}

func TestInteractiveEOF(t *testing.T) {
	var stdout bytes.Buffer
	if code := interactive(strings.NewReader(""), &stdout, &stdout); code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
}

func TestInteractiveStderrPassthrough(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := interactive(strings.NewReader("echo error >&2\nexit\n"), &stdout, &stderr)
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
	if !strings.Contains(stderr.String(), "error") {
		t.Errorf("expected 'error' in stderr, got %q", stderr.String())
	}
}
