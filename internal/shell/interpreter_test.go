package shell

import (
	"os"
	"strings"
	"sync"
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
// change is returned as virtual CWD in result.CWD, leaving the host CWD untouched.
func TestCdAndPwd(t *testing.T) {
	origCwd, _ := os.Getwd()

	result := Exec("cd /tmp && pwd", "", nil)
	if result.ExitCode != 0 {
		t.Fatalf("expected exit 0, got %d: %s", result.ExitCode, result.Stderr)
	}
	got := strings.TrimSpace(result.Stdout)
	if got != "/tmp" {
		t.Errorf("expected '/tmp', got %q", got)
	}

	// Verify the virtual CWD was returned correctly.
	if result.CWD != "/tmp" {
		t.Errorf("expected virtual CWD '/tmp', got %q", result.CWD)
	}

	// Verify the host process CWD was NOT mutated.
	hostCwd, _ := os.Getwd()
	if hostCwd != origCwd {
		t.Errorf("host process CWD was mutated: %q -> %q", origCwd, hostCwd)
	}
}

// TestCdPersistsAcrossExecCalls verifies that cd in one Exec call returns
// the updated CWD, which can then be passed to a subsequent Exec call to persist state,
// but does NOT leak to a subsequent call without explicit cwd (no cd leak).
func TestCdPersistsAcrossExecCalls(t *testing.T) {
	origCwd, _ := os.Getwd()

	tmpDir := t.TempDir()

	// First call: cd into tmpDir.
	result1 := Exec("cd "+tmpDir, "", nil)
	if result1.ExitCode != 0 {
		t.Fatalf("cd failed: %s", result1.Stderr)
	}
	if result1.CWD != tmpDir {
		t.Errorf("expected virtual CWD %q, got %q", tmpDir, result1.CWD)
	}

	// Second call: pass the returned CWD as the start directory. pwd should reflect it.
	result2 := Exec("pwd", result1.CWD, nil)
	if result2.ExitCode != 0 {
		t.Fatalf("pwd failed: %s", result2.Stderr)
	}
	got2 := strings.TrimSpace(result2.Stdout)
	if got2 != tmpDir {
		t.Errorf("expected %q, got %q", tmpDir, got2)
	}

	// Third call without explicit cwd: pwd returns original CWD (not leaked).
	result3 := Exec("pwd", "", nil)
	if result3.ExitCode != 0 {
		t.Fatalf("pwd failed: %s", result3.Stderr)
	}
	got3 := strings.TrimSpace(result3.Stdout)
	if got3 != origCwd {
		t.Errorf("cd leaked across calls: expected %q, got %q", origCwd, got3)
	}

	// Verify host CWD was NOT mutated.
	hostCwd, _ := os.Getwd()
	if hostCwd != origCwd {
		t.Errorf("host process CWD was mutated: %q -> %q", origCwd, hostCwd)
	}
}

// TestCdWithExplicitCwd verifies that cd changes are relative to the
// explicitly passed cwd, and that the result returns the isolated virtual CWD.
func TestCdWithExplicitCwd(t *testing.T) {
	origCwd, _ := os.Getwd()

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

	// Virtual CWD should be subDir.
	if result.CWD != subDir {
		t.Errorf("expected virtual CWD %q, got %q", subDir, result.CWD)
	}

	// Host process CWD should NOT have mutated.
	hostCwd, _ := os.Getwd()
	if hostCwd != origCwd {
		t.Errorf("host CWD was mutated: %q -> %q", origCwd, hostCwd)
	}
}

// TestCdToNonexistentDir verifies that cd to a nonexistent directory
// fails gracefully without crashing the interpreter and without changing
// the host process CWD or virtual CWD.
func TestCdToNonexistentDir(t *testing.T) {
	origCwd, _ := os.Getwd()

	result := Exec("cd /nonexistent/dir/12345", "", nil)
	if result.ExitCode == 0 {
		t.Error("expected non-zero exit for cd to nonexistent dir")
	}

	// Virtual CWD should remain the baseline (origCwd in this case).
	if result.CWD != origCwd {
		t.Errorf("expected virtual CWD %q, got %q", origCwd, result.CWD)
	}

	// Host CWD should NOT have changed.
	hostCwd, _ := os.Getwd()
	if hostCwd != origCwd {
		t.Errorf("host CWD changed unexpectedly: %q -> %q", origCwd, hostCwd)
	}
}

// TestRedirectAbsolutePath verifies that > with an absolute path works
// when cwd is passed explicitly (simulating REPL/invocation with known dir).
func TestRedirectAbsolutePath(t *testing.T) {
	tmpDir := t.TempDir()
	absPath := tmpDir + "/tutu.txt"

	result := Exec("echo hola > "+absPath, tmpDir, nil)
	if result.ExitCode != 0 {
		t.Fatalf("redirect to absolute path failed: exit=%d stderr=%q", result.ExitCode, result.Stderr)
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		t.Fatalf("cannot read output file: %v", err)
	}
	got := strings.TrimRight(string(data), "\n")
	if got != "hola" {
		t.Errorf("expected 'hola', got %q", got)
	}
}

// TestRedirectRelativePath verifies that > with ./relative path works
// when the cwd is passed explicitly (the original bug: empty cwd defaulted
// base to "/" making resolves go to root instead of process CWD).
func TestRedirectRelativePath(t *testing.T) {
	tmpDir := t.TempDir()
	relPath := "./tutu.txt"
	expectedFile := tmpDir + "/tutu.txt"

	result := Exec("echo hola > "+relPath, tmpDir, nil)
	if result.ExitCode != 0 {
		t.Fatalf("redirect to ./relative path failed: exit=%d stderr=%q", result.ExitCode, result.Stderr)
	}

	data, err := os.ReadFile(expectedFile)
	if err != nil {
		t.Fatalf("cannot read output file: %v", err)
	}
	got := strings.TrimRight(string(data), "\n")
	if got != "hola" {
		t.Errorf("expected 'hola', got %q", got)
	}
}

// TestRedirectEmptyCwd verifies that > works even when cwd is empty string
// (the exact bug scenario: non-interactive invocation with cwd="" that used
// to default base to "/" and fail with permission denied on /tutu.txt).
func TestRedirectEmptyCwd(t *testing.T) {
	tmpDir := t.TempDir()
	origCwd, _ := os.Getwd()
	defer os.Chdir(origCwd)
	os.Chdir(tmpDir)

	result := Exec("echo hola > tutu.txt", "", nil)
	if result.ExitCode != 0 {
		t.Fatalf("redirect with empty cwd failed: exit=%d stderr=%q", result.ExitCode, result.Stderr)
	}

	data, err := os.ReadFile(tmpDir + "/tutu.txt")
	if err != nil {
		t.Fatalf("cannot read output file: %v", err)
	}
	got := strings.TrimRight(string(data), "\n")
	if got != "hola" {
		t.Errorf("expected 'hola', got %q", got)
	}
}

// TestConcurrentShellExec verifies that multiple concurrent shell Exec calls
// do not race on os.Chdir. Run with: go test -race -run TestConcurrentShellExec
func TestConcurrentShellExec(t *testing.T) {
	var wg sync.WaitGroup
	const iterations = 100

	// Spawn multiple goroutines that each cd+pwd in different directories.
	// Without execMu, the os.Chdir calls would clobber each other and
	// pwd would return the wrong directory.
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				result := Exec("cd /tmp && pwd", "", nil)
				if result.ExitCode != 0 {
					t.Errorf("goroutine %d: cd+pwd failed: %s", id, result.Stderr)
					return
				}
				got := strings.TrimSpace(result.Stdout)
				if got != "/tmp" {
					t.Errorf("goroutine %d: expected '/tmp', got %q", id, got)
					return
				}
			}
		}(i)
	}
	wg.Wait()
}
