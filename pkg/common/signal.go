// Package common provides shared signal helpers for POSIX utilities.

package common

import (
	"fmt"
	"strconv"
	"strings"
	"syscall"
)

// signalNames maps Linux signal numbers to short names (without the SIG
// prefix), in BusyBox's kill -l order. Index 0 has no name.
var signalNames = [...]string{
	"", "HUP", "INT", "QUIT", "ILL", "TRAP", "ABRT", "BUS", "FPE", "KILL",
	"USR1", "SEGV", "USR2", "PIPE", "ALRM", "TERM", "STKFLT", "CHLD", "CONT",
	"STOP", "TSTP", "TTIN", "TTOU", "URG", "XCPU", "XFSZ", "VTALRM", "PROF",
	"WINCH", "IO", "PWR", "SYS",
}

// ParseSignal resolves a signal name (TERM, SIGKILL — case-insensitive,
// SIG prefix optional) or a non-negative decimal number to a syscall.Signal.
// Signal 0 (existence probe) is allowed.
func ParseSignal(sigStr string) (syscall.Signal, error) {
	s := strings.ToUpper(strings.TrimSpace(sigStr))
	s = strings.TrimPrefix(s, "SIG")
	for i, name := range signalNames {
		if i > 0 && s == name {
			return syscall.Signal(i), nil
		}
	}
	if val, err := strconv.Atoi(s); err == nil && val >= 0 {
		return syscall.Signal(val), nil
	}
	return 0, fmt.Errorf("unknown signal: %s", sigStr)
}

// SignalName returns the short name for a signal number (no SIG prefix),
// or the number as a string when the signal has no name.
func SignalName(sig syscall.Signal) string {
	n := int(sig)
	if n >= 0 && n < len(signalNames) && signalNames[n] != "" {
		return signalNames[n]
	}
	return strconv.Itoa(n)
}

// SignalNames returns the short names of all known signals, in BusyBox -l order.
func SignalNames() []string {
	return append([]string(nil), signalNames[1:]...)
}
