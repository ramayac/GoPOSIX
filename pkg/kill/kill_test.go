package kill

import (
	"bytes"
	"math"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
)

func spawnSleep(t *testing.T) *exec.Cmd {
	t.Helper()
	cmd := exec.Command("sleep", "5")
	if err := cmd.Start(); err != nil {
		t.Fatalf("start sleep: %v", err)
	}
	t.Cleanup(func() { _ = cmd.Wait() })
	return cmd
}

func runJSON(t *testing.T, args []string) (int, string) {
	t.Helper()
	var out bytes.Buffer
	code := run(args, nil, &out, &out, "")
	return code, out.String()
}

func TestKillMissingArgs(t *testing.T) {
	var out bytes.Buffer
	rc := run([]string{}, nil, &out, &out, "")
	if rc != 0 {
		t.Errorf("expected 0, got %d", rc)
	}
}

func TestKillJSON(t *testing.T) {
	code, out := runJSON(t, []string{"--json", "9999999"})
	if code != 1 {
		t.Errorf("expected 1, got %d", code)
	}
	if !strings.Contains(out, "command") {
		t.Errorf("expected JSON, got %s", out)
	}
}

func TestKillSignalFlagTerm(t *testing.T) {
	child := spawnSleep(t)
	code, out := runJSON(t, []string{"--json", "-s", "TERM", strconv.Itoa(child.Process.Pid)})
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(out, `"success":true`) {
		t.Errorf("expected success, got %s", out)
	}
	if !strings.Contains(out, `"SIGTERM"`) {
		t.Errorf("expected SIGTERM in output, got %s", out)
	}
}

func TestKillLowercaseSignalName(t *testing.T) {
	child := spawnSleep(t)
	code, out := runJSON(t, []string{"--json", "-s", "term", strconv.Itoa(child.Process.Pid)})
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(out, `"success":true`) {
		t.Errorf("expected success, got %s", out)
	}
}

func TestKillDashName(t *testing.T) {
	child := spawnSleep(t)
	code, out := runJSON(t, []string{"--json", "-TERM", strconv.Itoa(child.Process.Pid)})
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(out, `"success":true`) {
		t.Errorf("expected success, got %s", out)
	}
}

func TestKillDashNumberNine(t *testing.T) {
	child := spawnSleep(t)
	code, out := runJSON(t, []string{"--json", "-9", strconv.Itoa(child.Process.Pid)})
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(out, `"success":true`) {
		t.Errorf("expected success, got %s", out)
	}
	if !strings.Contains(out, `"SIGKILL"`) {
		t.Errorf("expected SIGKILL in output, got %s", out)
	}
}

func TestKillDashNumberFifteen(t *testing.T) {
	child := spawnSleep(t)
	code, out := runJSON(t, []string{"--json", "-15", strconv.Itoa(child.Process.Pid)})
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(out, `"success":true`) {
		t.Errorf("expected success, got %s", out)
	}
}

func TestKillInvalidSignal(t *testing.T) {
	code, _ := runJSON(t, []string{"-s", "BOGUS", "1"})
	if code != 1 {
		t.Errorf("expected 1, got %d", code)
	}
}

func TestKillListSignals(t *testing.T) {
	code, out := runJSON(t, []string{"-l"})
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	for _, want := range []string{"HUP", "KILL", "TERM", "SYS"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in list, got %s", want, out)
		}
	}
}

func TestKillListNumber(t *testing.T) {
	code, out := runJSON(t, []string{"-l", "9"})
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if strings.TrimSpace(out) != "KILL" {
		t.Errorf("expected KILL, got %q", out)
	}
}

func TestKillListJSON(t *testing.T) {
	code, out := runJSON(t, []string{"--json", "-l", "15"})
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(out, `"signals"`) || !strings.Contains(out, "TERM") {
		t.Errorf("expected signals list JSON, got %s", out)
	}
}

func TestKillListInvalidSignal(t *testing.T) {
	code, _ := runJSON(t, []string{"-l", "BOGUS"})
	if code != 1 {
		t.Errorf("expected 1, got %d", code)
	}
}

func TestKillSignalZeroSelf(t *testing.T) {
	code, out := runJSON(t, []string{"--json", "-s", "0", strconv.Itoa(os.Getpid())})
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(out, `"success":true`) {
		t.Errorf("expected success probing self, got %s", out)
	}
}

func TestKillSignalZeroNonexistent(t *testing.T) {
	code, out := runJSON(t, []string{"--json", "-s", "0", strconv.Itoa(math.MaxInt32)})
	if code != 1 {
		t.Errorf("expected 1, got %d", code)
	}
	if !strings.Contains(out, `"success":false`) {
		t.Errorf("expected failure, got %s", out)
	}
}

func TestKillInvalidPID(t *testing.T) {
	var buf bytes.Buffer
	code := run([]string{"abc"}, nil, &buf, &buf, "")
	if code == 0 {
		t.Error("expected non-zero exit for invalid PID")
	}
}
