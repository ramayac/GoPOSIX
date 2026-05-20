// Package rmdir implements the POSIX rmdir utility.
package rmdir

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
)

// RmdirResult is the --json output.
type RmdirResult struct {
	Removed []string `json:"removed"`
}

var spec = common.FlagSpec{
	Defs: []common.FlagDef{
		{Short: "p", Long: "parents", Type: common.FlagBool},
		{Long: "json", Type: common.FlagBool},
	},
}

// Run removes empty directories. With parents=true, removes ancestor dirs too.
func Run(dirs []string, parents bool) (RmdirResult, error) {
	var removed []string
	for _, d := range dirs {
		if err := os.Remove(d); err != nil {
			return RmdirResult{Removed: removed}, err
		}
		removed = append(removed, d)
		if parents {
			for p := filepath.Dir(d); p != "." && p != "/"; p = filepath.Dir(p) {
				if err := os.Remove(p); err != nil {
					break // stop at first non-empty ancestor
				}
				removed = append(removed, p)
			}
		}
	}
	return RmdirResult{Removed: removed}, nil
}

func run(args []string, stdin io.Reader, stdout io.Writer) int {
	flags, err := common.ParseFlags(args, spec)
	if err != nil {
		fmt.Fprintf(os.Stderr, "rmdir: %v\n", err)
		return 2
	}
	jsonMode := flags.Has("json")
	if len(flags.Positional) == 0 {
		fmt.Fprintln(os.Stderr, "rmdir: missing operand")
		return 1
	}
	result, err := Run(flags.Positional, flags.Has("p"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "rmdir: %v\n", err)
		common.RenderError("rmdir", 1, "ERMDIR", err.Error(), jsonMode, stdout)
		return 1
	}
	common.Render("rmdir", result, jsonMode, stdout, func() {})
	return 0
}

func init() {
	dispatch.Register(dispatch.Command{Name: "rmdir", Usage: "Remove empty directories", Run: run})
}
