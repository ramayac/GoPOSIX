package common

import (
	"syscall"
	"testing"
)

func TestParseSignalByName(t *testing.T) {
	cases := []struct {
		in   string
		want syscall.Signal
	}{
		{"HUP", syscall.SIGHUP},
		{"hup", syscall.SIGHUP},
		{"SIGTERM", syscall.SIGTERM},
		{"sigkill", syscall.SIGKILL},
		{"KILL", syscall.SIGKILL},
		{"INT", syscall.SIGINT},
		{"USR1", syscall.SIGUSR1},
		{"CHLD", syscall.SIGCHLD},
		{"SYS", syscall.Signal(31)},
	}
	for _, c := range cases {
		got, err := ParseSignal(c.in)
		if err != nil {
			t.Fatalf("ParseSignal(%q): %v", c.in, err)
		}
		if got != c.want {
			t.Errorf("ParseSignal(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestParseSignalByNumber(t *testing.T) {
	cases := []struct {
		in   string
		want syscall.Signal
	}{
		{"0", syscall.Signal(0)},
		{"1", syscall.SIGHUP},
		{"9", syscall.SIGKILL},
		{"15", syscall.SIGTERM},
		{"64", syscall.Signal(64)},
	}
	for _, c := range cases {
		got, err := ParseSignal(c.in)
		if err != nil {
			t.Fatalf("ParseSignal(%q): %v", c.in, err)
		}
		if got != c.want {
			t.Errorf("ParseSignal(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestParseSignalInvalid(t *testing.T) {
	for _, in := range []string{"BOGUS", "-1", "", "SIG"} {
		if _, err := ParseSignal(in); err == nil {
			t.Errorf("ParseSignal(%q): expected error", in)
		}
	}
}

func TestSignalName(t *testing.T) {
	if got := SignalName(syscall.SIGKILL); got != "KILL" {
		t.Errorf("SignalName(SIGKILL) = %q, want KILL", got)
	}
	if got := SignalName(syscall.Signal(0)); got != "0" {
		t.Errorf("SignalName(0) = %q, want 0", got)
	}
	if got := SignalName(syscall.Signal(64)); got != "64" {
		t.Errorf("SignalName(64) = %q, want 64", got)
	}
}

func TestSignalNamesList(t *testing.T) {
	names := SignalNames()
	if len(names) == 0 {
		t.Fatal("expected non-empty signal name list")
	}
	seen := make(map[string]bool)
	for _, n := range names {
		if seen[n] {
			t.Errorf("duplicate name %q", n)
		}
		seen[n] = true
	}
	for _, want := range []string{"HUP", "KILL", "TERM", "SYS"} {
		found := false
		for _, n := range names {
			if n == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("signal list missing %q", want)
		}
	}
}
