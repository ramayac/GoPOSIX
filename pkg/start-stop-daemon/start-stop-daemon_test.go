package startstopdaemon

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func TestStartStopDaemonHelp(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"--help"}, nil, &stdout, &stderr, "")
	if code != 0 {
		t.Errorf("Expected code 0, got %d", code)
	}
	if !bytes.Contains(stdout.Bytes(), []byte("Usage: start-stop-daemon")) {
		t.Errorf("Expected help output, got: %s", stdout.String())
	}
}

func TestStartStopDaemonMissingAction(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"-x", "true"}, nil, &stdout, &stderr, "")
	if code == 0 {
		t.Error("Expected error for missing start or stop action")
	}

	// JSON mode
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"--json", "-x", "true"}, nil, &stdout, &stderr, "")
	if code == 0 {
		t.Error("Expected JSON error for missing action")
	}
}

func TestStartStopDaemonFlagError(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"--invalid-flag"}, nil, &stdout, &stderr, "")
	if code == 0 {
		t.Error("Expected error for invalid flags")
	}
}

func TestStartStopDaemonStart(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	// Case 1: Start true (dry run)
	code := run([]string{"-S", "-t", "-x", "true"}, nil, &stdout, &stderr, "")
	if code != 0 {
		t.Errorf("Expected code 0, got %d. Stderr: %s", code, stderr.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte("Would start true")) {
		t.Errorf("Expected dry run message, got: %s", stdout.String())
	}

	// Case 2: Start false (dry run)
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"-S", "-t", "-a", "false", "--", "arg1"}, nil, &stdout, &stderr, "")
	if code != 0 {
		t.Errorf("Expected code 0, got %d. Stderr: %s", code, stderr.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte("Would start false")) {
		t.Errorf("Expected dry run message, got: %s", stdout.String())
	}

	// Case 3: Foreground exec true
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"-S", "-x", "true"}, nil, &stdout, &stderr, "")
	if code != 0 {
		t.Errorf("Expected code 0, got %d. Stderr: %s", code, stderr.String())
	}

	// Case 4: Foreground exec false (fails with 1)
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"-S", "-x", "false"}, nil, &stdout, &stderr, "")
	if code != 1 {
		t.Errorf("Expected exit code 1, got %d. Stderr: %s", code, stderr.String())
	}

	// Case 5: Missing command to start
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"-S"}, nil, &stdout, &stderr, "")
	if code == 0 {
		t.Error("Expected failure for missing command")
	}
}

func TestStartStopDaemonBackgroundAndPidfile(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "start-stop-daemon-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	pidPath := filepath.Join(tempDir, "daemon.pid")
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	// Case 1: Run background command (sleep 10) and write pidfile
	code := run([]string{"-S", "-b", "-m", "-p", pidPath, "-x", "sleep", "10"}, nil, &stdout, &stderr, tempDir)
	if code != 0 {
		t.Fatalf("Expected code 0, got %d. Stderr: %s", code, stderr.String())
	}

	content, err := os.ReadFile(pidPath)
	if err != nil {
		t.Fatal(err)
	}
	pid, err := strconv.Atoi(filepath.Clean(string(bytes.TrimSpace(content))))
	if err != nil || pid <= 0 {
		t.Errorf("Expected valid pid in pidfile, got: %s", string(content))
	}

	// Kill background sleep process
	p, errFind := os.FindProcess(pid)
	if errFind == nil {
		_ = p.Kill()
	}

	// Case 2: Already running matching (matched by pidfile)
	// Write our own PID to pidfile so it is matching
	myPid := os.Getpid()
	if err := os.WriteFile(pidPath, []byte(strconv.Itoa(myPid)+"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	stdout.Reset()
	stderr.Reset()
	code = run([]string{"-S", "-p", pidPath, "-x", "true"}, nil, &stdout, &stderr, tempDir)
	if code == 0 {
		t.Error("Expected failure (exit 1) because process is already running")
	}

	// With oknodo
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"-S", "-o", "-p", pidPath, "-x", "true"}, nil, &stdout, &stderr, tempDir)
	if code != 0 {
		t.Errorf("Expected exit code 0 under oknodo, got %d. Stderr: %s", code, stderr.String())
	}

	// Case 3: Stop matching processes
	stdout.Reset()
	stderr.Reset()
	// Test stop mode (dry run)
	code = run([]string{"-K", "-t", "-p", pidPath}, nil, &stdout, &stderr, tempDir)
	if code != 0 {
		t.Fatalf("Expected code 0 on dry run stop, got %d. Stderr: %s", code, stderr.String())
	}

	// Stop (send signal 0 - check only)
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"-K", "-s", "0", "-p", pidPath}, nil, &stdout, &stderr, tempDir)
	if code != 0 {
		t.Errorf("Expected code 0 on check stop, got %d. Stderr: %s", code, stderr.String())
	}

	// Case 4: Stop when nothing matches
	os.Remove(pidPath)
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"-K", "-p", pidPath}, nil, &stdout, &stderr, tempDir)
	if code == 0 {
		t.Error("Expected failure (exit 1) when nothing matches on stop")
	}

	// With oknodo
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"-K", "-o", "-p", pidPath}, nil, &stdout, &stderr, tempDir)
	if code != 0 {
		t.Errorf("Expected exit code 0 under oknodo, got %d. Stderr: %s", code, stderr.String())
	}
}

func TestStartStopDaemonSignals(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	// Check valid signal mappings
	for _, sigStr := range []string{"1", "HUP", "2", "INT", "3", "QUIT", "9", "KILL", "15", "TERM", "USR1", "USR2"} {
		stdout.Reset()
		stderr.Reset()
		code := run([]string{"-K", "-t", "-s", sigStr, "-p", "dummy.pid"}, nil, &stdout, &stderr, "")
		// Should fail because dummy.pid nonexistent, but signal parsing should be clean.
		if code == 0 {
			t.Error("Expected stop to fail on nonexistent pidfile")
		}
	}

	// Invalid signal
	stdout.Reset()
	stderr.Reset()
	code := run([]string{"-K", "-s", "INVALID_SIG", "-p", "dummy.pid"}, nil, &stdout, &stderr, "")
	if code == 0 {
		t.Error("Expected failure on invalid signal name")
	}
}

func TestStartStopDaemonJSONMode(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	// Case 1: Start (dry run) JSON
	code := run([]string{"--json", "-S", "-t", "-x", "true"}, nil, &stdout, &stderr, "")
	if code != 0 {
		t.Fatalf("Expected code 0, got %d", code)
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"action":"start"`)) || !bytes.Contains(stdout.Bytes(), []byte(`"status":"would start"`)) {
		t.Errorf("Expected valid JSON response, got:\n%s", stdout.String())
	}

	// Case 2: Foreground start JSON
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"--json", "-S", "-x", "true"}, nil, &stdout, &stderr, "")
	if code != 0 {
		t.Fatalf("Expected code 0, got %d", code)
	}

	// Case 3: Stop (nothing matched) JSON
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"--json", "-K", "-p", "nonexistent.pid"}, nil, &stdout, &stderr, "")
	// Should fail with 1 (unless oknodo, but returns valid JSON error/status envelope on stdout/stderr!)
	if !bytes.Contains(stdout.Bytes(), []byte(`"status":"nothing stopped"`)) {
		t.Errorf("Expected JSON envelope with status 'nothing stopped', got:\n%s", stdout.String())
	}
}
