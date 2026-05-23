package taskset

import (
	"bytes"
	"os"
	"strconv"
	"testing"
)

func TestTasksetHelp(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"-h"}, nil, &stdout, &stderr, "")
	if code != 0 {
		t.Errorf("Expected code 0, got %d", code)
	}
	if !bytes.Contains(stdout.Bytes(), []byte("Usage: taskset")) {
		t.Errorf("Expected help output, got: %s", stdout.String())
	}
}

func TestTasksetMissingArgs(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{}, nil, &stdout, &stderr, "")
	if code == 0 {
		t.Error("Expected error for missing arguments")
	}

	stdout.Reset()
	stderr.Reset()
	code = run([]string{"-p"}, nil, &stdout, &stderr, "")
	if code == 0 {
		t.Error("Expected error for missing PID in -p mode")
	}

	// JSON mode
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"--json", "-p"}, nil, &stdout, &stderr, "")
	if code == 0 {
		t.Error("Expected error in JSON mode for missing PID")
	}
}

func TestTasksetFlagError(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"--invalid-flag"}, nil, &stdout, &stderr, "")
	if code == 0 {
		t.Error("Expected error for invalid flags")
	}
}

func TestTasksetGetAffinity(t *testing.T) {
	myPid := os.Getpid()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"-p", strconv.Itoa(myPid)}, nil, &stdout, &stderr, "")
	if code != 0 {
		t.Fatalf("Expected code 0, got %d. Stderr: %s", code, stderr.String())
	}

	expectedPrefix := "pid " + strconv.Itoa(myPid) + "'s current affinity mask:"
	if !bytes.Contains(stdout.Bytes(), []byte(expectedPrefix)) {
		t.Errorf("Expected output to contain prefix %q, got: %s", expectedPrefix, stdout.String())
	}

	// Test invalid PID
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"-p", "0"}, nil, &stdout, &stderr, "")
	if code == 0 {
		t.Error("Expected error for PID 0")
	}
}

func TestTasksetSetAffinity(t *testing.T) {
	myPid := os.Getpid()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	// Set affinity to 0x1 (highly portable, at least CPU 0 is always present)
	code := run([]string{"-p", "0x1", strconv.Itoa(myPid)}, nil, &stdout, &stderr, "")
	// If permissions are not sufficient (e.g. inside a restricted container), SchedSetaffinity might fail.
	// But let's verify if the command finishes gracefully or fails with permission.
	// We handle setAffinity errors gracefully.
	if code != 0 {
		t.Logf("SetAffinity failed (expected if lacking privileges): %s", stderr.String())
	} else {
		expectedPrefix := "pid " + strconv.Itoa(myPid) + "'s new affinity mask: 1"
		if !bytes.Contains(stdout.Bytes(), []byte(expectedPrefix)) {
			t.Errorf("Expected output to contain %q, got: %s", expectedPrefix, stdout.String())
		}
	}

	// Test JSON mode get
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"--json", "-p", strconv.Itoa(myPid)}, nil, &stdout, &stderr, "")
	if code != 0 {
		t.Fatalf("Expected code 0 in JSON mode, got %d. Stderr: %s", code, stderr.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"currentMask"`)) {
		t.Errorf("Expected JSON to contain currentMask, got:\n%s", stdout.String())
	}

	// Test JSON mode set
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"--json", "-p", "0x1", strconv.Itoa(myPid)}, nil, &stdout, &stderr, "")
	if code != 0 {
		// SchedSetaffinity may fail
		t.Logf("JSON set affinity failed: %s", stderr.String())
	} else {
		if !bytes.Contains(stdout.Bytes(), []byte(`"newMask":"1"`)) {
			t.Errorf("Expected JSON to contain newMask: 1, got:\n%s", stdout.String())
		}
	}
}

func TestTasksetCommandMode(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	// We execute "true" command
	code := run([]string{"0x1", "true"}, nil, &stdout, &stderr, "")
	if code != 0 {
		t.Logf("Command execution failed: %s", stderr.String())
	}

	// Command not found exit code
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"0x1", "nonexistent-command-name-xyz"}, nil, &stdout, &stderr, "")
	if code != 127 {
		t.Errorf("Expected exit code 127 for missing command, got %d", code)
	}
}

func TestTasksetErrorPaths(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	// Case 1: Invalid hex mask in PID mode
	code := run([]string{"-p", "0xZ", "123"}, nil, &stdout, &stderr, "")
	if code == 0 {
		t.Error("Expected failure for invalid hex mask")
	}

	// Case 2: Extra arguments in PID mode
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"-p", "0x1", "123", "extra"}, nil, &stdout, &stderr, "")
	if code == 0 {
		t.Error("Expected failure for extra arguments in PID mode")
	}

	// Case 3: Missing command in command mode
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"0x1"}, nil, &stdout, &stderr, "")
	if code == 0 {
		t.Error("Expected failure for missing command")
	}

	// Case 4: JSON error for missing command
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"--json", "0x1"}, nil, &stdout, &stderr, "")
	if code == 0 {
		t.Error("Expected failure for missing command in JSON mode")
	}

	// Case 5: JSON error for extra args in PID mode
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"--json", "-p", "0x1", "123", "extra"}, nil, &stdout, &stderr, "")
	if code == 0 {
		t.Error("Expected JSON failure for extra arguments in PID mode")
	}

	// Case 6: Invalid hex mask in command mode
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"0xZ", "true"}, nil, &stdout, &stderr, "")
	if code == 0 {
		t.Error("Expected failure for invalid hex mask in command mode")
	}
}

