// Package nice implements the POSIX nice utility — run a command with modified scheduling priority.
package nice

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"

	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
	"golang.org/x/sys/unix"
)

// NiceResult is the --json output.
type NiceResult struct {
	Adjustment int      `json:"adjustment"`
	Command    []string `json:"command"`
	ExitCode   int      `json:"exit_code"`
}

var spec = common.FlagSpec{
	Defs: []common.FlagDef{
		{Short: "n", Long: "adjustment", Type: common.FlagValue},
		{Long: "json", Type: common.FlagBool},
	},
}

// Run adjusts the niceness and executes the given command.
func Run(adjustment int, command []string) (int, error) {
	if len(command) == 0 {
		return 0, fmt.Errorf("missing command")
	}

	// Set niceness
	if err := unix.Setpriority(unix.PRIO_PROCESS, 0, adjustment); err != nil {
		return 0, fmt.Errorf("setpriority: %w", err)
	}

	// Execute the command
	cmd := exec.Command(command[0], command[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode(), nil
		}
		return 0, err
	}
	return 0, nil
}

func run(args []string, stdin io.Reader, stdout io.Writer) int {
	flags, err := common.ParseFlags(args, spec)
	if err != nil {
		fmt.Fprintf(os.Stderr, "nice: %v\n", err)
		return 2
	}
	jsonMode := flags.Has("json")

	adjustment := 10 // POSIX default increment
	if flags.Has("n") {
		adjStr := flags.Get("n")
		adj, err := strconv.Atoi(adjStr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "nice: invalid adjustment: %s\n", adjStr)
			common.RenderError("nice", 1, "EARGS", "invalid adjustment", jsonMode, stdout)
			return 1
		}
		adjustment = adj
	}

	if len(flags.Positional) == 0 {
		fmt.Fprintln(os.Stderr, "nice: missing command")
		common.RenderError("nice", 1, "EARGS", "missing command", jsonMode, stdout)
		return 1
	}

	exitCode, err := Run(adjustment, flags.Positional)
	if err != nil {
		fmt.Fprintf(os.Stderr, "nice: %v\n", err)
		common.RenderError("nice", 1, "ENICE", err.Error(), jsonMode, stdout)
		return 1
	}

	if jsonMode {
		result := NiceResult{
			Adjustment: adjustment,
			Command:    flags.Positional,
			ExitCode:   exitCode,
		}
		common.Render("nice", result, true, stdout, func() {})
	}
	return exitCode
}

func init() {
	dispatch.Register(dispatch.Command{
		Name:  "nice",
		Usage: "Run a command with modified scheduling priority",
		Run:   run,
	})
}
