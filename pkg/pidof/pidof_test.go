package pidof

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
	"testing"
)

func TestPidofBasic(t *testing.T) {
	// Start a background process
	cmd := exec.Command("sleep", "15")
	if err := cmd.Start(); err != nil {
		t.Skip("skipping /proc dependent test: cannot run sleep")
		return
	}
	defer cmd.Process.Kill()

	sleepPid := cmd.Process.Pid
	sleepPidStr := fmt.Sprintf("%d", sleepPid)

	// Try to find "sleep"
	var stdout, stderr bytes.Buffer
	gotExit := run([]string{"sleep"}, nil, &stdout, &stderr, "")
	if gotExit != 0 {
		t.Fatalf("expected exit 0 for finding 'sleep', got %d. stderr: %s", gotExit, stderr.String())
	}

	pids := strings.Fields(stdout.String())
	found := false
	for _, p := range pids {
		if p == sleepPidStr {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("pidof sleep did not find PID %d, got: %s", sleepPid, stdout.String())
	}
}

func TestPidofFlags(t *testing.T) {
	cmd := exec.Command("sleep", "15")
	if err := cmd.Start(); err != nil {
		t.Skip("skipping /proc dependent test: cannot run sleep")
		return
	}
	defer cmd.Process.Kill()

	sleepPid := cmd.Process.Pid

	// 1. Single shot -s
	var stdout, stderr bytes.Buffer
	gotExit := run([]string{"-s", "sleep"}, nil, &stdout, &stderr, "")
	if gotExit != 0 {
		t.Fatalf("expected exit 0, got %d", gotExit)
	}
	pids := strings.Fields(stdout.String())
	if len(pids) != 1 {
		t.Errorf("expected exactly 1 PID for -s, got %d: %v", len(pids), pids)
	}

	// 2. Omit PID
	stdout.Reset()
	stderr.Reset()
	gotExit = run([]string{"-o", fmt.Sprintf("%d", sleepPid), "sleep"}, nil, &stdout, &stderr, "")
	pids = strings.Fields(stdout.String())
	for _, p := range pids {
		if p == fmt.Sprintf("%d", sleepPid) {
			t.Errorf("expected PID %d to be omitted, but found it in %v", sleepPid, pids)
		}
	}

	// 3. Omit %PPID
	stdout.Reset()
	stderr.Reset()
	_ = run([]string{"-o", "%PPID", "sleep"}, nil, &stdout, &stderr, "")

	// 4. JSON Mode
	stdout.Reset()
	stderr.Reset()
	gotExit = run([]string{"--json", "sleep"}, nil, &stdout, &stderr, "")
	if gotExit == 0 {
		if !strings.Contains(stdout.String(), `"pids":`) {
			t.Errorf("expected JSON output format, got: %s", stdout.String())
		}
	}
}

func TestPidofEdgeCases(t *testing.T) {
	// 1. Invalid Flag
	var stdout, stderr bytes.Buffer
	gotExit := run([]string{"--invalid-flag"}, nil, &stdout, &stderr, "")
	if gotExit != 1 {
		t.Errorf("expected exit 1 for invalid flag, got %d", gotExit)
	}

	// 2. Missing Operand
	stdout.Reset()
	stderr.Reset()
	gotExit = run([]string{}, nil, &stdout, &stderr, "")
	if gotExit != 1 {
		t.Errorf("expected exit 1 for missing operand, got %d", gotExit)
	}

	// 3. Very unlikely binary name
	stdout.Reset()
	stderr.Reset()
	gotExit = run([]string{"veryunlikelyoccuringbinaryname"}, nil, &stdout, &stderr, "")
	if gotExit != 1 {
		t.Errorf("expected exit 1 for missing process, got %d", gotExit)
	}
}
