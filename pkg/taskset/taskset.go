// Package taskset implements the POSIX-compliant taskset utility.
package taskset

import (
	"fmt"
	"io"
	"os/exec"
	"strconv"

	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
)

var spec = common.FlagSpec{
	Defs: []common.FlagDef{
		{Short: "p", Long: "pid", Type: common.FlagBool},
		{Short: "h", Long: "help", Type: common.FlagBool},
		{Long: "json", Type: common.FlagBool},
	},
}

func init() {
	dispatch.Register(dispatch.Command{
		Name:  "taskset",
		Usage: "Retrieve or set a process's CPU affinity",
		Run:   run,
	})
}

// TasksetResult represents the JSON response structure.
type TasksetResult struct {
	Pid         int    `json:"pid"`
	CurrentMask string `json:"currentMask,omitempty"`
	NewMask     string `json:"newMask,omitempty"`
	Command     string `json:"command,omitempty"`
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer, cwd string) int {
	jsonMode := false
	for _, arg := range args {
		if arg == "--json" {
			jsonMode = true
			break
		}
	}

	flags, err := common.ParseFlags(args, spec)
	if err != nil {
		if jsonMode {
			common.RenderError("taskset", 1, "FLAG_ERROR", err.Error(), true, stderr)
		} else {
			fmt.Fprintf(stderr, "taskset: %v\n", err)
		}
		return 1
	}

	if flags.Has("h") || flags.Has("help") {
		helpText := "Usage: taskset [-p] [MASK] [PID | COMMAND [ARG]...]\n\n" +
			"Retrieve or set a process's CPU affinity.\n\n" +
			"Options:\n" +
			"  -p, --pid      Operate on an existing PID\n" +
			"  -h, --help     Print help"
		common.Render("taskset", struct {
			Help string `json:"help"`
		}{Help: helpText}, jsonMode, stdout, func() {
			fmt.Fprintln(stdout, helpText)
		})
		return 0
	}

	pidMode := flags.Has("p") || flags.Has("pid")
	pos := flags.Positional

	if pidMode {
		if len(pos) == 0 {
			if jsonMode {
				common.RenderError("taskset", 1, "MISSING_ARGUMENT", "missing PID", true, stderr)
			} else {
				fmt.Fprintln(stderr, "taskset: missing PID")
			}
			return 1
		}

		// Mode 1: taskset -p PID (retrieve affinity)
		if len(pos) == 1 {
			pid, err := strconv.Atoi(pos[0])
			if err != nil || pid <= 0 {
				if jsonMode {
					common.RenderError("taskset", 1, "INVALID_PID", "invalid pid", true, stderr)
				} else {
					fmt.Fprintf(stderr, "taskset: invalid pid: %s\n", pos[0])
				}
				return 1
			}

			mask, err := getAffinity(pid)
			if err != nil {
				if jsonMode {
					common.RenderError("taskset", 1, "GET_AFFINITY_ERROR", err.Error(), true, stderr)
				} else {
					fmt.Fprintf(stderr, "taskset: failed to get affinity for pid %d: %v\n", pid, err)
				}
				return 1
			}

			if jsonMode {
				common.Render("taskset", TasksetResult{Pid: pid, CurrentMask: mask}, true, stdout, nil)
			} else {
				fmt.Fprintf(stdout, "pid %d's current affinity mask: %s\n", pid, mask)
			}
			return 0
		}

		// Mode 2: taskset -p MASK PID (set affinity)
		if len(pos) == 2 {
			maskStr := pos[0]
			pid, err := strconv.Atoi(pos[1])
			if err != nil || pid <= 0 {
				if jsonMode {
					common.RenderError("taskset", 1, "INVALID_PID", "invalid pid", true, stderr)
				} else {
					fmt.Fprintf(stderr, "taskset: invalid pid: %s\n", pos[1])
				}
				return 1
			}

			oldMask, newMask, err := setAffinity(pid, maskStr)
			if err != nil {
				if jsonMode {
					common.RenderError("taskset", 1, "SET_AFFINITY_ERROR", err.Error(), true, stderr)
				} else {
					fmt.Fprintf(stderr, "taskset: failed to set affinity for pid %d: %v\n", pid, err)
				}
				return 1
			}

			if jsonMode {
				common.Render("taskset", TasksetResult{Pid: pid, CurrentMask: oldMask, NewMask: newMask}, true, stdout, nil)
			} else {
				fmt.Fprintf(stdout, "pid %d's current affinity mask: %s\n", pid, oldMask)
				fmt.Fprintf(stdout, "pid %d's new affinity mask: %s\n", pid, newMask)
			}
			return 0
		}

		if jsonMode {
			common.RenderError("taskset", 1, "BAD_USAGE", "invalid arguments", true, stderr)
		} else {
			fmt.Fprintln(stderr, "taskset: invalid arguments")
		}
		return 1
	}

	// Command mode: taskset MASK COMMAND [ARGS...]
	if len(pos) < 2 {
		if jsonMode {
			common.RenderError("taskset", 1, "MISSING_ARGUMENT", "missing mask or command", true, stderr)
		} else {
			fmt.Fprintln(stderr, "taskset: missing mask or command")
		}
		return 1
	}

	maskStr := pos[0]
	cmdName := pos[1]
	cmdArgs := pos[2:]

	// Set affinity of our own process so the spawned process inherits it!
	_, _, err = setAffinity(0, maskStr)
	if err != nil {
		if jsonMode {
			common.RenderError("taskset", 1, "SET_AFFINITY_ERROR", err.Error(), true, stderr)
		} else {
			fmt.Fprintf(stderr, "taskset: failed to set affinity: %v\n", err)
		}
		return 1
	}

	cmd := exec.Command(cmdName, cmdArgs...)
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Dir = cwd

	err = cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode()
		}
		if jsonMode {
			common.RenderError("taskset", 127, "COMMAND_NOT_FOUND", err.Error(), true, stderr)
		} else {
			fmt.Fprintf(stderr, "taskset: %v\n", err)
		}
		return 127
	}

	return 0
}
