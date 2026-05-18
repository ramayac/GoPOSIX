// Package mkdir implements the POSIX mkdir utility.
package mkdir

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"strconv"

	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
)

// MkdirResult is the --json output.
type MkdirResult struct {
	Created []string `json:"created"`
}

var spec = common.FlagSpec{
	Defs: []common.FlagDef{
		{Short: "p", Long: "parents", Type: common.FlagBool},
		{Short: "m", Long: "mode", Type: common.FlagValue},
		{Long: "json", Type: common.FlagBool},
	},
}

// Run creates directories and returns the list of created paths.
func Run(dirs []string, parents bool, mode fs.FileMode) (MkdirResult, error) {
	var created []string
	for _, d := range dirs {
		var err error
		if parents {
			err = os.MkdirAll(d, mode)
		} else {
			err = os.Mkdir(d, mode)
		}
		if err != nil {
			return MkdirResult{Created: created}, err
		}
		created = append(created, d)
	}
	return MkdirResult{Created: created}, nil
}

func run(args []string, out io.Writer) int {
	flags, err := common.ParseFlags(args, spec)
	if err != nil {
		fmt.Fprintf(os.Stderr, "mkdir: %v\n", err)
		return 2
	}
	jsonMode := flags.Has("json")
	parents := flags.Has("p")
	mode := fs.FileMode(0755)
	if mStr := flags.Get("m"); mStr != "" {
		m, err := strconv.ParseUint(mStr, 8, 32)
		if err != nil {
			fmt.Fprintf(os.Stderr, "mkdir: invalid mode %q\n", mStr)
			return 2
		}
		mode = fs.FileMode(m)
	}
	if len(flags.Positional) == 0 {
		fmt.Fprintln(os.Stderr, "mkdir: missing operand")
		return 1
	}
	result, err := Run(flags.Positional, parents, mode)
	if err != nil {
		fmt.Fprintf(os.Stderr, "mkdir: %v\n", err)
		common.RenderError("mkdir", 1, "EMKDIR", err.Error(), jsonMode, out)
		return 1
	}
	common.Render("mkdir", result, jsonMode, out, func() {})
	return 0
}

func init() {
	dispatch.Register(dispatch.Command{Name: "mkdir", Usage: "Make directories", Run: run})
}
