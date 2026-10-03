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
		{Short: "L", Long: "logical", Type: common.FlagBool},
		{Long: "json", Type: common.FlagBool},
	},
}

// Run returns the current working directory.
//
// Logical (-L): use $PWD when it names the current directory, per POSIX -L
// semantics; otherwise fall back to the physical path.
//
// Physical (default, -P): resolve all symlinks. This matches BusyBox and
// coreutils, where pwd defaults to the physical path. Note os.Getwd may
// itself honor a valid $PWD, so the physical path is derived by resolving
// symlinks rather than trusting os.Getwd directly.
func Run(logical bool) (PwdResult, error) {
	dir, err := os.Getwd()
	if err != nil {
		return PwdResult{}, err
	}
	if logical {
		if pwd := os.Getenv("PWD"); filepath.IsAbs(pwd) && sameDir(pwd, dir) {
			return PwdResult{Path: filepath.Clean(pwd)}, nil
		}
	}
	phys, err := filepath.EvalSymlinks(dir)
	if err != nil {
		return PwdResult{}, err
	}
	return PwdResult{Path: phys}, nil
}

// sameDir reports whether path names the same directory as cwd.
func sameDir(path, cwd string) bool {
	pInfo, err := os.Stat(path)
	if err != nil {
		return false
	}
	cInfo, err := os.Stat(cwd)
	if err != nil {
		return false
	}
	return os.SameFile(pInfo, cInfo)
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer, cwd string) int {
	flags, err := common.ParseFlags(args, spec)
	if err != nil {
		fmt.Fprintf(os.Stderr, "pwd: %v\n", err)
		return 2
	}
	jsonMode := flags.Has("json")
	logical := flags.Has("L") // -L wins; the default is physical

	result, err := Run(logical)
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
