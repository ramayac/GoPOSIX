// Package tty implements the POSIX tty utility — print the file name of the terminal.
package tty

import (
	"fmt"
	"io"
	"os"
	"syscall"

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
	fd := int(os.Stdin.Fd())
	_, err := unix.IoctlGetTermios(fd, unix.TCGETS)
	if err != nil {
		return TtyResult{IsTTY: false}, nil
	}
	path, err := ttyname(fd)
	if err != nil {
		// Still a tty, just can't get the name
		return TtyResult{IsTTY: true, Path: "unknown"}, nil
	}
	return TtyResult{IsTTY: true, Path: path}, nil
}

func run(args []string, out io.Writer) int {
	flags, err := common.ParseFlags(args, spec)
	if err != nil {
		fmt.Fprintf(os.Stderr, "tty: %v\n", err)
		return 2
	}
	jsonMode := flags.Has("json")
	silent := flags.Has("s")

	result, err := Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "tty: %v\n", err)
		common.RenderError("tty", 1, "ETTY", err.Error(), jsonMode, out)
		return 1
	}

	if silent {
		if !result.IsTTY {
			return 1
		}
		return 0
	}

	if !jsonMode {
		if result.IsTTY {
			fmt.Fprintln(out, result.Path)
		} else {
			fmt.Fprintln(out, "not a tty")
		}
	} else {
		common.Render("tty", result, jsonMode, out, func() {})
	}
	return 0
}

func init() {
	dispatch.Register(dispatch.Command{
		Name:  "tty",
		Usage: "Print the file name of the terminal connected to standard input",
		Run:   run,
	})
}
