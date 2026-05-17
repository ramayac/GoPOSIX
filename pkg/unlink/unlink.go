// Package unlink implements the POSIX unlink utility — remove a single file or symlink.
package unlink

import (
	"fmt"
	"io"
	"os"

	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
)

// UnlinkResult is the --json output.
type UnlinkResult struct {
	Removed string `json:"removed"`
}

var spec = common.FlagSpec{
	Defs: []common.FlagDef{
		{Long: "json", Type: common.FlagBool},
	},
}

// Run removes a single file or symlink. Returns an error for directories.
func Run(path string) error {
	return os.Remove(path)
}

func run(args []string, out io.Writer) int {
	flags, err := common.ParseFlags(args, spec)
	if err != nil {
		fmt.Fprintf(os.Stderr, "unlink: %v\n", err)
		return 2
	}
	jsonMode := flags.Has("json")

	if len(flags.Positional) != 1 {
		fmt.Fprintln(os.Stderr, "unlink: missing operand")
		common.RenderError("unlink", 1, "EARGS", "missing operand", jsonMode, out)
		return 1
	}

	path := flags.Positional[0]

	if err := Run(path); err != nil {
		fmt.Fprintf(os.Stderr, "unlink: %v\n", err)
		common.RenderError("unlink", 1, "EUNLINK", err.Error(), jsonMode, out)
		return 1
	}

	result := UnlinkResult{Removed: path}
	common.Render("unlink", result, jsonMode, out, func() {})
	return 0
}

func init() {
	dispatch.Register(dispatch.Command{
		Name:  "unlink",
		Usage: "Remove a single file or symbolic link",
		Run:   run,
	})
}
