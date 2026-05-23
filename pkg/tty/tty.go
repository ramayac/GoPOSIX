// Package tty implements the POSIX tty utility — print the file name of the terminal.
package tty

import (
	"fmt"
	"io"
	"os"

	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
	"golang.org/x/sys/unix"
)

// TtyResult is the --json output.
type TtyResult struct {
	IsTTY bool   `json:"is_tty"`
	Path  string `json:"path,omitempty"`
}

var spec = common.FlagSpec{
	Defs: []common.FlagDef{
		{Short: "s", Long: "silent", Type: common.FlagBool},
		{Long: "json", Type: common.FlagBool},
	},
}

var (
	isTerminalFn = func(fd int) bool {
		_, err := unix.IoctlGetTermios(fd, unix.TCGETS)
		return err == nil
	}
	ttynameFn = ttyname
	runTtyFn  = runTty
)

// ttyname returns the path of the terminal associated with fd.
func ttyname(fd int) (string, error) {
	// Use TIOCGTPEER or just read /proc/self/fd/N
	path := fmt.Sprintf("/proc/self/fd/%d", fd)
	link, err := os.Readlink(path)
	if err != nil {
		return "", err
	}
	return link, nil
}

// Run checks whether stdin is a terminal and returns its path.
func Run() (TtyResult, error) {
	return runTty(os.Stdin)
}

func runTty(stdin io.Reader) (TtyResult, error) {
	fdable, ok := stdin.(interface{ Fd() uintptr })
	if !ok {
		return TtyResult{IsTTY: false}, nil
	}
	fd := int(fdable.Fd())
	if !isTerminalFn(fd) {
		return TtyResult{IsTTY: false}, nil
	}
	path, err := ttynameFn(fd)
	if err != nil {
		// Still a tty, just can't get the name
		return TtyResult{IsTTY: true, Path: "unknown"}, nil
	}
	return TtyResult{IsTTY: true, Path: path}, nil
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer, cwd string) int {
	flags, err := common.ParseFlags(args, spec)
	if err != nil {
		fmt.Fprintf(stderr, "tty: %v\n", err)
		return 2
	}
	jsonMode := flags.Has("json")
	silent := flags.Has("s")

	result, err := runTtyFn(stdin)
	if err != nil {
		fmt.Fprintf(stderr, "tty: %v\n", err)
		common.RenderError("tty", 1, "ETTY", err.Error(), jsonMode, stdout)
		return 1
	}

	if silent {
		if !result.IsTTY {
			return 1
		}
		return 0
	}

	common.Render("tty", result, jsonMode, stdout, func() {
		if result.IsTTY {
			fmt.Fprintln(stdout, result.Path)
		} else {
			fmt.Fprintln(stdout, "not a tty")
		}
	})
	return 0
}

func init() {
	dispatch.Register(dispatch.Command{
		Name:  "tty",
		Usage: "Print the file name of the terminal connected to standard input",
		Run:   run,
	})
}
