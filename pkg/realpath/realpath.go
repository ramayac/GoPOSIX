// Package realpath implements the POSIX/GNU-compliant realpath utility.
package realpath

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
)

// RealpathResult contains the resolution results.
type RealpathResult struct {
	Resolved map[string]string `json:"resolved"`
}

var spec = common.FlagSpec{
	Defs: []common.FlagDef{
		{Short: "e", Long: "canonicalize-existing", Type: common.FlagBool},
		{Short: "m", Long: "canonicalize-missing", Type: common.FlagBool},
		{Short: "q", Long: "quiet", Type: common.FlagBool},
		{Short: "s", Long: "no-symlinks", Type: common.FlagBool},
		{Long: "json", Type: common.FlagBool},
	},
}

// resolvePath resolves relative, absolute, and symlinked paths.
// existing forces all components (including the last one) to exist.
// missingOk allows components of the path to not exist.
func resolvePath(path string, cwd string) (string, error) {
	return resolvePathFlags(path, cwd, false, false, false)
}

func resolvePathFlags(path string, cwd string, existing bool, missingOk bool, noSymlinks bool) (string, error) {
	if !filepath.IsAbs(path) {
		path = filepath.Join(cwd, path)
	}
	path = filepath.Clean(path)

	if noSymlinks {
		// Just return the clean absolute path
		if existing {
			if _, err := os.Stat(path); err != nil {
				return "", err
			}
		}
		return path, nil
	}

	vol := filepath.VolumeName(path)
	rem := path[len(vol):]
	components := strings.Split(rem, string(filepath.Separator))
	var parts []string
	for _, c := range components {
		if c != "" {
			parts = append(parts, c)
		}
	}

	curr := vol + string(filepath.Separator)
	symlinkCount := 0

	for i := 0; i < len(parts); i++ {
		next := filepath.Join(curr, parts[i])
		info, err := os.Lstat(next)
		if err != nil {
			if missingOk {
				for j := i; j < len(parts); j++ {
					curr = filepath.Join(curr, parts[j])
				}
				return curr, nil
			}
			if i == len(parts)-1 && !existing {
				return next, nil
			}
			return "", err
		}

		if info.Mode()&os.ModeSymlink != 0 {
			symlinkCount++
			if symlinkCount > 32 {
				return "", fmt.Errorf("too many levels of symbolic links")
			}
			target, err := os.Readlink(next)
			if err != nil {
				return "", err
			}

			if filepath.IsAbs(target) {
				remaining := parts[i+1:]
				vol = filepath.VolumeName(target)
				rem = target[len(vol):]
				targetParts := strings.Split(rem, string(filepath.Separator))
				var newParts []string
				for _, c := range targetParts {
					if c != "" {
						newParts = append(newParts, c)
					}
				}
				newParts = append(newParts, remaining...)
				parts = newParts
				curr = vol + string(filepath.Separator)
				i = -1
			} else {
				targetAbs := filepath.Join(curr, target)
				targetAbs = filepath.Clean(targetAbs)
				remaining := parts[i+1:]
				vol = filepath.VolumeName(targetAbs)
				rem = targetAbs[len(vol):]
				targetParts := strings.Split(rem, string(filepath.Separator))
				var newParts []string
				for _, c := range targetParts {
					if c != "" {
						newParts = append(newParts, c)
					}
				}
				newParts = append(newParts, remaining...)
				parts = newParts
				curr = vol + string(filepath.Separator)
				i = -1
			}
		} else {
			curr = next
		}
	}
	return curr, nil
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer, cwd string) int {
	flags, err := common.ParseFlags(args, spec)
	if err != nil {
		fmt.Fprintf(stderr, "realpath: %v\n", err)
		return 2
	}

	existing := flags.Has("canonicalize-existing")
	missingOk := flags.Has("canonicalize-missing")
	quiet := flags.Has("quiet")
	noSymlinks := flags.Has("no-symlinks")
	jsonMode := flags.Has("json")

	posArgs := flags.Positional
	if len(posArgs) == 0 {
		// realpath without arguments defaults to resolving "."
		posArgs = []string{"."}
	}

	result := RealpathResult{
		Resolved: make(map[string]string),
	}

	anyFailed := false
	for _, path := range posArgs {
		res, err := resolvePathFlags(path, cwd, existing, missingOk, noSymlinks)
		if err != nil {
			anyFailed = true
			if !quiet {
				fmt.Fprintf(stderr, "realpath: %s: No such file or directory\n", path)
			}
		} else {
			result.Resolved[path] = res
		}
	}

	common.Render("realpath", result, jsonMode, stdout, func() {
		for _, path := range posArgs {
			if res, exists := result.Resolved[path]; exists {
				fmt.Fprintln(stdout, res)
			}
		}
	})

	if anyFailed {
		return 1
	}
	return 0
}

func init() {
	dispatch.Register(dispatch.Command{
		Name:  "realpath",
		Usage: "Return the canonicalized absolute path",
		Run:   run,
	})
}
