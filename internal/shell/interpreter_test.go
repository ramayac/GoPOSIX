package shell

import (
	"os"
	"strings"
	"testing"
)

func TestExecBasic(t *testing.T) {
	result := Exec("echo hello", "", nil)
	if result.Stdout != "hello\n" {
		t.Errorf("expected 'hello\\n', got %q", result.Stdout)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit 0, got %d", result.ExitCode)
	}
}

func TestTimeout(t *testing.T) {
	os.Setenv("GOPOSIX_SHELL_TIMEOUT", "500ms")
	defer os.Unsetenv("GOPOSIX_SHELL_TIMEOUT")

	result := Exec("sleep 10", "", nil)
	if result.ExitCode == 0 {
		t.Error("expected non-zero exit code from timeout, got 0")
	}
	if !strings.Contains(result.Stderr, "deadline") && !strings.Contains(result.Stderr, "killed") && !strings.Contains(result.Stderr, "signal") {
		t.Logf("stderr from timeout: %q", result.Stderr)
	}
}

func TestTimeoutViaEnv(t *testing.T) {
	os.Setenv("GOPOSIX_SHELL_TIMEOUT", "100ms")
	defer os.Unsetenv("GOPOSIX_SHELL_TIMEOUT")

	result := Exec("sleep 5", "", nil)
	if result.ExitCode == 0 {
		t.Error("expected non-zero exit from 100ms timeout")
	}
}

func TestOutputWithinLimits(t *testing.T) {
	// Verify that output within the 128MB LimitWriter cap works correctly.
	result := Exec("echo hello && echo world", "", nil)
	if result.ExitCode != 0 {
		t.Errorf("unexpected exit %d: %s", result.ExitCode, result.Stderr)
	}
	if !strings.Contains(result.Stdout, "hello") || !strings.Contains(result.Stdout, "world") {
		t.Errorf("unexpected stdout: %q", result.Stdout)
	}
}

func TestPathEscape(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(tmpDir+"/allowed.txt", []byte("ok"), 0644)

	result := Exec("cat allowed.txt", tmpDir, nil)
	if result.ExitCode != 0 {
		t.Fatalf("allowed read failed: %s", result.Stderr)
	}
	if result.Stdout != "ok" {
		t.Errorf("expected 'ok', got %q", result.Stdout)
	}
}

func TestPathEscapeBlocked(t *testing.T) {
	tmpDir := t.TempDir()

	// openHandler intercepts shell-level file opens (redirections like <, >).
	// Use a shell redirection to test path traversal blocking.
	result := Exec("cat < ../../../etc/passwd", tmpDir, nil)
	if result.ExitCode == 0 {
		t.Error("expected non-zero exit for path traversal attempt")
	}
	errOut := strings.ToLower(result.Stderr)
	if !strings.Contains(errOut, "traversal") && !strings.Contains(errOut, "permission") && !strings.Contains(errOut, "no such file") && !strings.Contains(errOut, "not found") {
		t.Logf("stderr from path escape attempt: %q", result.Stderr)
	}
}

func TestEnvVarInjection(t *testing.T) {
	result := Exec("echo $TEST_VAR", "", map[string]string{"TEST_VAR": "injected"})
	if result.Stdout != "injected\n" {
		t.Errorf("expected 'injected\\n', got %q", result.Stdout)
	}
}

func TestStderrCapture(t *testing.T) {
	result := Exec("echo error >&2", "", nil)
	if result.Stderr != "error\n" {
		t.Errorf("expected 'error\\n' on stderr, got %q", result.Stderr)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit 0, got %d", result.ExitCode)
	}
}

func TestNonZeroExit(t *testing.T) {
	result := Exec("exit 42", "", nil)
	if result.ExitCode != 42 {
		t.Errorf("expected exit 42, got %d", result.ExitCode)
	}
}

func TestSyntaxError(t *testing.T) {
	result := Exec("{{{{", "", nil)
	if result.ExitCode != 127 {
		t.Errorf("expected exit 127 for syntax error, got %d", result.ExitCode)
	}
}

// TestCdAndPwd verifies that cd within a shell script changes the working
// directory for subsequent commands in the same script, and that the
// change is synced back to the host process via os.Chdir.
func TestCdAndPwd(t *testing.T) {
	origCwd, _ := os.Getwd()
	defer os.Chdir(origCwd)

	result := Exec("cd /tmp && pwd", "", nil)
	if result.ExitCode != 0 {
		t.Fatalf("expected exit 0, got %d: %s", result.ExitCode, result.Stderr)
	}
	got := strings.TrimSpace(result.Stdout)
	if got != "/tmp" {
		t.Errorf("expected '/tmp', got %q", got)
	}

	// Verify the host process CWD was synced to /tmp.
	hostCwd, _ := os.Getwd()
	if hostCwd != "/tmp" {
		t.Errorf("host process CWD not synced: expected '/tmp', got %q", hostCwd)
	}
}

// TestCdPersistsAcrossExecCalls verifies that a cd in one Exec call
// persists to subsequent Exec calls when no explicit cwd is passed
// (the host process CWD carries forward).
func TestCdPersistsAcrossExecCalls(t *testing.T) {
	origCwd, _ := os.Getwd()
	defer os.Chdir(origCwd)

	tmpDir := t.TempDir()

	// First call: cd into tmpDir.
	result1 := Exec("cd "+tmpDir, "", nil)
	if result1.ExitCode != 0 {
		t.Fatalf("cd failed: %s", result1.Stderr)
	}

	// Second call: pwd should reflect the tmpDir.
	result2 := Exec("pwd", "", nil)
	if result2.ExitCode != 0 {
		t.Fatalf("pwd failed: %s", result2.Stderr)
	}
	got := strings.TrimSpace(result2.Stdout)
	if got != tmpDir {
		t.Errorf("expected %q, got %q", tmpDir, got)
	}
}

// TestCdWithExplicitCwd verifies that cd changes are relative to the
// explicitly passed cwd, and that the result syncs back correctly.
func TestCdWithExplicitCwd(t *testing.T) {
	origCwd, _ := os.Getwd()
	defer os.Chdir(origCwd)

	tmpDir := t.TempDir()
	subDir := tmpDir + "/sub"
	os.Mkdir(subDir, 0755)

	// Pass tmpDir as explicit cwd, then cd into sub.
	result := Exec("cd sub && pwd", tmpDir, nil)
	if result.ExitCode != 0 {
		t.Fatalf("expected exit 0, got %d: %s", result.ExitCode, result.Stderr)
	}
	got := strings.TrimSpace(result.Stdout)
	if got != subDir {
		t.Errorf("expected %q, got %q", subDir, got)
	}

	// Host process CWD should now be subDir.
	hostCwd, _ := os.Getwd()
	if hostCwd != subDir {
		t.Errorf("host CWD not synced: expected %q, got %q", subDir, hostCwd)
	}
}

// TestCdToNonexistentDir verifies that cd to a nonexistent directory
// fails gracefully without crashing the interpreter and without changing
// the host process CWD.
func TestCdToNonexistentDir(t *testing.T) {
	origCwd, _ := os.Getwd()
	defer os.Chdir(origCwd)

	result := Exec("cd /nonexistent/dir/12345", "", nil)
	if result.ExitCode == 0 {
		t.Error("expected non-zero exit for cd to nonexistent dir")
	}

	// Host CWD should NOT have changed.
	hostCwd, _ := os.Getwd()
	if hostCwd != origCwd {
		t.Errorf("host CWD changed unexpectedly: %q -> %q", origCwd, hostCwd)
	}
}
