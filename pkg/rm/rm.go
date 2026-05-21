// Package rm implements the POSIX rm utility.
package rm

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
)

// RmResult is the --json output.
type RmResult struct {
	Removed []string `json:"removed"`
	Errors  []string `json:"errors,omitempty"`
}

var spec = common.FlagSpec{
	Defs: []common.FlagDef{
		{Short: "r", Long: "recursive", Type: common.FlagBool},
		{Short: "f", Long: "force", Type: common.FlagBool},
		{Short: "v", Long: "verbose", Type: common.FlagBool},
		{Long: "no-preserve-root", Type: common.FlagBool},
		{Long: "json", Type: common.FlagBool},
	},
}

// preservedRoots is the set of paths that are always protected.
var preservedRoots = map[string]bool{
	"/":    true,
	"/.":   true,
	"/..":  true,
	"//":   true,
	"/usr": true,
}

// isSafeToRemove returns false for root-equivalent paths.
// isSafeToRemove returns false for root-equivalent paths unless
// noPreserveRoot is true.
func isSafeToRemove(path string, noPreserveRoot bool) bool {
	if noPreserveRoot {
		return true
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	return !preservedRoots[abs]
}

// Run removes files/directories. Refuses to remove root unless noPreserveRoot.
func Run(paths []string, recursive, force, verbose, noPreserveRoot bool) (RmResult, error) {
	var result RmResult
	for _, p := range paths {
		if !isSafeToRemove(p, noPreserveRoot) {
			msg := fmt.Sprintf("rm: refusing to remove %q: use --no-preserve-root to override", p)
			result.Errors = append(result.Errors, msg)
			if !force {
				return result, fmt.Errorf("%s", msg)
			}
			continue
		}
		var err error
		if recursive {
			err = removeAllVerbose(p, &result.Removed, verbose)
		} else {
			err = os.Remove(p)
			if err == nil {
				result.Removed = append(result.Removed, p)
				if verbose {
					fmt.Printf("removed %q\n", p)
				}
			}
		}
		if err != nil && !force {
			result.Errors = append(result.Errors, err.Error())
			return result, err
		}
	}
	return result, nil
}

func removeAllVerbose(path string, removed *[]string, verbose bool) error {
	if err := os.RemoveAll(path); err != nil {
		return err
	}
	*removed = append(*removed, path)
	if verbose {
		fmt.Printf("removed %q\n", path)
	}
	return nil
}

func run(args []string, stdin io.Reader, stdout io.Writer) int {
	flags, err := common.ParseFlags(args, spec)
	if err != nil {
		fmt.Fprintf(os.Stderr, "rm: %v\n", err)
		return 2
	}
	jsonMode := flags.Has("json")
	recursive := flags.Has("r")
	force := flags.Has("f")
	verbose := flags.Has("v")
	noPreserveRoot := flags.Has("no-preserve-root")

	if len(flags.Positional) == 0 {
		if !force {
			fmt.Fprintln(os.Stderr, "rm: missing operand")
			return 1
		}
		return 0
	}

	var result RmResult
	exitCode := 0
	for _, p := range flags.Positional {
		if !isSafeToRemove(p, noPreserveRoot) {
			msg := fmt.Sprintf("refusing to remove %q: use --no-preserve-root to override", p)
			fmt.Fprintf(os.Stderr, "rm: %s\n", msg)
			result.Errors = append(result.Errors, msg)
			exitCode = 1
			continue
		}
		var rmErr error
		if recursive {
			rmErr = removeAllVerbose(p, &result.Removed, verbose)
		} else {
			rmErr = os.Remove(p)
			if rmErr == nil {
				result.Removed = append(result.Removed, p)
				if verbose {
					fmt.Printf("removed %q\n", p)
				}
			}
		}
		if rmErr != nil {
			if !force {
				fmt.Fprintf(os.Stderr, "rm: %v\n", rmErr)
				result.Errors = append(result.Errors, rmErr.Error())
				exitCode = 1
			}
		}
	}
	common.Render("rm", result, jsonMode, stdout, func() {})
	return exitCode
}

func init() {
	dispatch.Register(dispatch.Command{Name: "rm", Usage: "Remove files or directories", Run: run})
}
