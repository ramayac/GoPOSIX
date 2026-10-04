// Package kill implements the POSIX kill utility — send a signal to a process.
package kill

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"syscall"

	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
)

var spec = common.FlagSpec{
	Defs: []common.FlagDef{
		{Short: "s", Long: "signal", Type: common.FlagValue},
		{Short: "l", Long: "list", Type: common.FlagBool},
		{Long: "json", Type: common.FlagBool},
	},
}

type KillResult struct {
	PID     int    `json:"pid"`
	Signal  string `json:"signal"`
	Success bool   `json:"success"`
}

type KillResp struct {
	Signaled []KillResult `json:"signaled"`
}

// KillListResp is the --json output for kill -l.
type KillListResp struct {
	Signals []string `json:"signals"`
}

// preprocessArgs rewrites dash-form signal options (-TERM, -15) into
// "-s <signal>" pairs so common.ParseFlags never sees them as bundled
// short flags. Known flags (-s, -l, --json) pass through untouched.
func preprocessArgs(args []string) []string {
	var out []string
	for _, a := range args {
		if len(a) > 1 && a[0] == '-' && a[1] != '-' {
			rest := a[1:]
			if isAllDigits(rest) {
				out = append(out, "-s", rest)
				continue
			}
			if _, err := common.ParseSignal(rest); err == nil {
				out = append(out, "-s", rest)
				continue
			}
		}
		out = append(out, a)
	}
	return out
}

func isAllDigits(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return s != ""
}

// sigLabel renders a signal for --json output: SIGTERM-style for named
// signals, plain number otherwise (signal 0 has no name).
func sigLabel(sig syscall.Signal) string {
	name := common.SignalName(sig)
	if isAllDigits(name) {
		return name
	}
	return "SIG" + name
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer, cwd string) int {
	flags, err := common.ParseFlags(preprocessArgs(args), spec)
	if err != nil {
		return common.RenderFlagError("kill", args, err, stderr, 1)
	}
	jsonMode := flags.Has("json")

	// -l: list signal names, optionally resolving numbers to names.
	if flags.Has("l") {
		if len(flags.Positional) == 0 {
			signals := common.SignalNames()
			if jsonMode {
				common.Render("kill", KillListResp{Signals: signals}, true, stdout, nil)
			} else {
				fmt.Fprintln(stdout, strings.Join(signals, " "))
			}
			return 0
		}
		var names []string
		exitCode := 0
		for _, p := range flags.Positional {
			sig, err := common.ParseSignal(p)
			if err != nil {
				fmt.Fprintf(stderr, "kill: invalid signal: %s\n", p)
				exitCode = 1
				continue
			}
			names = append(names, common.SignalName(sig))
		}
		if jsonMode {
			common.Render("kill", KillListResp{Signals: names}, true, stdout, nil)
		} else {
			fmt.Fprintln(stdout, strings.Join(names, " "))
		}
		return exitCode
	}

	sig := syscall.SIGTERM
	if v := flags.Get("s"); v != "" {
		s, err := common.ParseSignal(v)
		if err != nil {
			fmt.Fprintf(stderr, "kill: invalid signal: %s\n", v)
			return 1
		}
		sig = s
	}

	var res []KillResult
	exitCode := 0

	for _, p := range flags.Positional {
		pid, err := strconv.Atoi(p)
		if err != nil {
			fmt.Fprintf(stderr, "kill: %s: arguments must be process or job IDs\n", p)
			exitCode = 1
			continue
		}

		err = syscall.Kill(pid, sig)
		res = append(res, KillResult{
			PID:     pid,
			Signal:  sigLabel(sig),
			Success: err == nil,
		})

		if err != nil {
			fmt.Fprintf(stderr, "kill: (%d) - %v\n", pid, err)
			exitCode = 1
		}
	}

	if jsonMode {
		common.Render("kill", KillResp{Signaled: res}, true, stdout, nil)
	}

	return exitCode
}

func init() {
	dispatch.Register(dispatch.Command{Name: "kill", Usage: "Send a signal to a process", Run: run})
}
