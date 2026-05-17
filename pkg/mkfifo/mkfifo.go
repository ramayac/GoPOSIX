// Package mkfifo implements the POSIX mkfifo utility — create FIFO special files.
package mkfifo

import (
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
	"golang.org/x/sys/unix"
)

// MkfifoResult is the --json output.
type MkfifoResult struct {
	Path string `json:"path"`
	Mode string `json:"mode"`
}

var spec = common.FlagSpec{
	Defs: []common.FlagDef{
		{Short: "m", Long: "mode", Type: common.FlagValue},
		{Long: "json", Type: common.FlagBool},
	},
}

// Run creates a FIFO at path with the given mode.
func Run(path string, mode os.FileMode) error {
	return unix.Mkfifo(path, uint32(mode))
}

func run(args []string, out io.Writer) int {
	flags, err := common.ParseFlags(args, spec)
	if err != nil {
		fmt.Fprintf(os.Stderr, "mkfifo: %v\n", err)
		return 2
	}
	jsonMode := flags.Has("json")

	if len(flags.Positional) == 0 {
		fmt.Fprintln(os.Stderr, "mkfifo: missing operand")
		common.RenderError("mkfifo", 1, "EARGS", "missing operand", jsonMode, out)
		return 1
	}

	path := flags.Positional[0]
	mode := os.FileMode(0666) // default with umask applied

	if flags.Has("m") {
		modeStr := flags.Get("m")
		parsed, err := strconv.ParseUint(modeStr, 8, 32)
		if err != nil {
			fmt.Fprintf(os.Stderr, "mkfifo: invalid mode: %s\n", modeStr)
			common.RenderError("mkfifo", 1, "EMODE", "invalid mode", jsonMode, out)
			return 1
		}
		mode = os.FileMode(parsed)
	}

	if err := Run(path, mode); err != nil {
		fmt.Fprintf(os.Stderr, "mkfifo: %v\n", err)
		common.RenderError("mkfifo", 1, "EMKFIFO", err.Error(), jsonMode, out)
		return 1
	}

	result := MkfifoResult{
		Path: path,
		Mode: fmt.Sprintf("0%o", mode),
	}
	common.Render("mkfifo", result, jsonMode, out, func() {})
	return 0
}

func init() {
	dispatch.Register(dispatch.Command{
		Name:  "mkfifo",
		Usage: "Create FIFO special files (named pipes)",
		Run:   run,
	})
}
