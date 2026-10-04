// Package nohup implements the POSIX nohup utility — run a command immune to SIGHUP.
package nohup

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
	"golang.org/x/sys/unix"
)

// NohupResult is the --json output.
type NohupResult struct {
	Command    []string `json:"command"`
	OutputFile string   `json:"output_file,omitempty"`
	ExitCode   int      `json:"exit_code"`
}

var spec = common.FlagSpec{
	Defs: []common.FlagDef{
		{Long: "json", Type: common.FlagBool},
	},
}

const defaultOutputFile = "nohup.stdout"

// isTerminal returns true if fd is a terminal.
func isTerminal(fd uintptr) bool {
	_, err := unix.IoctlGetTermios(int(fd), unix.TCGETS)
	return err == nil
}

// terminalCheck is a seam for tests. It reports whether w is a real terminal.
var terminalCheck = func(w io.Writer) bool {
	f, ok := w.(*os.File)
	return ok && isTerminal(f.Fd())
}

// Run executes a command immune to SIGHUP, redirecting output if stdout is a terminal.
// stdin/stdout/stderr are the standard streams for the child process.
func Run(command []string, stdin io.Reader, stdout, stderr io.Writer) (NohupResult, error) {
	if len(command) == 0 {
		return NohupResult{}, fmt.Errorf("missing command")
	}

	// Ignore SIGHUP
	signal.Ignore(syscall.SIGHUP)

	cmd := exec.Command(command[0], command[1:]...)
	cmd.Stdin = stdin
	cmd.Stderr = stderr

	result := NohupResult{Command: command}

	// stdoutIsTerminal reports whether w is a real terminal stream.
	stdoutIsTerminal := terminalCheck

	// If stdout is a terminal, redirect to nohup.stdout
	if stdoutIsTerminal(stdout) {
		f, err := os.OpenFile(defaultOutputFile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
		if err != nil {
			return result, fmt.Errorf("cannot open %s: %w", defaultOutputFile, err)
		}
		cmd.Stdout = f
		result.OutputFile = defaultOutputFile
		defer f.Close()
	} else {
		cmd.Stdout = stdout
	}

	// If stderr is a terminal, redirect it to stdout
	if stdoutIsTerminal(stderr) {
		cmd.Stderr = cmd.Stdout
	}

	err := cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
			return result, nil
		}
		return result, err
	}
	result.ExitCode = 0
	return result, nil
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer, cwd string) int {
	flags, err := common.ParseFlags(args, spec)
	if err != nil {
		fmt.Fprintf(stderr, "nohup: %v\n", err)
		return 2
	}
	jsonMode := flags.Has("json")

	if len(flags.Positional) == 0 {
		fmt.Fprintln(stderr, "nohup: missing operand")
		common.RenderError("nohup", 1, "EARGS", "missing operand", jsonMode, stdout)
		return 1
	}

	result, err := Run(flags.Positional, stdin, stdout, stderr)
	if err != nil {
		fmt.Fprintf(stderr, "nohup: %v\n", err)
		common.RenderError("nohup", 1, "ENOHUP", err.Error(), jsonMode, stdout)
		return 1
	}

	common.Render("nohup", result, jsonMode, stdout, func() {})
	return result.ExitCode
}

func init() {
	dispatch.Register(dispatch.Command{
		Name:  "nohup",
		Usage: "Run a command immune to hangups, with output to a non-tty",
		Run:   run,
	})
}
