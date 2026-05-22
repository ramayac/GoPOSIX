// Package pwd implements the POSIX pwd utility.
package pwd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
)

// PwdResult is the structured result for --json mode.
type PwdResult struct {
	Path string `json:"path"`
}

var spec = common.FlagSpec{
	Defs: []common.FlagDef{
		{Short: "P", Long: "physical", Type: common.FlagBool},
		{Long: "json", Type: common.FlagBool},
	},
}

// Run returns the current working directory.
// If physical is true, symlinks are resolved.
func Run(physical bool) (PwdResult, error) {
	dir, err := os.Getwd()
	if err != nil {
		return PwdResult{}, err
	}
	if physical {
		dir, err = filepath.EvalSymlinks(dir)
		if err != nil {
			return PwdResult{}, err
		}
	}
	return PwdResult{Path: dir}, nil
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer, cwd string) int {
	flags, err := common.ParseFlags(args, spec)
	if err != nil {
		fmt.Fprintf(os.Stderr, "pwd: %v\n", err)
		return 2
	}
	jsonMode := flags.Has("json")
	physical := flags.Has("P")

	result, err := Run(physical)
	if err != nil {
		fmt.Fprintf(os.Stderr, "pwd: %v\n", err)
		common.RenderError("pwd", 1, "EPWD", err.Error(), jsonMode, stdout)
		return 1
	}

	common.Render("pwd", result, jsonMode, stdout, func() {
		fmt.Fprintln(stdout, result.Path)
	})
	return 0
}

func init() {
	dispatch.Register(dispatch.Command{
		Name:  "pwd",
		Usage: "Print the current working directory",
		Run:   run,
	})
}
