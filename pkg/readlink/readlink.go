// Package readlink implements the POSIX readlink utility.
package readlink

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
)

// ReadlinkResult is the --json output.
type ReadlinkResult struct {
	Path   string `json:"path"`
	Target string `json:"target"`
}

var spec = common.FlagSpec{
	Defs: []common.FlagDef{
		{Short: "f", Long: "canonicalize", Type: common.FlagBool},
		{Long: "json", Type: common.FlagBool},
	},
}

// Run reads the symlink target for path.
func Run(path string, canonicalize bool) (ReadlinkResult, error) {
	if canonicalize {
		var base string
		pwdEnv := os.Getenv("PWD")
		if pwdEnv != "" {
			wd, err := os.Getwd()
			if err == nil {
				pwdFi, err1 := os.Stat(pwdEnv)
				wdFi, err2 := os.Stat(wd)
				if err1 == nil && err2 == nil && os.SameFile(pwdFi, wdFi) {
					base = filepath.Clean(pwdEnv)
				}
			}
		}

		var abs string
		var underBase bool
		if base != "" {
			if filepath.IsAbs(path) {
				abs = filepath.Clean(path)
			} else {
				abs = filepath.Join(base, path)
			}
			underBase = hasPrefix(abs, base)
		}

		if underBase {
			rel, err := filepath.Rel(base, abs)
			if err == nil && !strings.HasPrefix(rel, "..") {
				if resolved, ok := evalSymlinksUnder(base, rel); ok {
					return ReadlinkResult{Path: path, Target: filepath.Clean(resolved)}, nil
				}
			}
		}

		// Fallback to standard physical evaluation
		absPhysical, err := filepath.Abs(path)
		if err != nil {
			return ReadlinkResult{}, err
		}
		resolved, err := filepath.EvalSymlinks(absPhysical)
		if err != nil {
			return ReadlinkResult{}, err
		}
		return ReadlinkResult{Path: path, Target: filepath.Clean(resolved)}, nil
	}
	target, err := os.Readlink(path)
	if err != nil {
		return ReadlinkResult{}, err
	}
	return ReadlinkResult{Path: path, Target: target}, nil
}

func hasPrefix(p, prefix string) bool {
	p = filepath.Clean(p)
	prefix = filepath.Clean(prefix)
	if p == prefix {
		return true
	}
	if !strings.HasSuffix(prefix, string(filepath.Separator)) {
		prefix += string(filepath.Separator)
	}
	return strings.HasPrefix(p, prefix)
}

func splitPath(path string) []string {
	if path == "" || path == "." {
		return nil
	}
	return strings.Split(filepath.ToSlash(path), "/")
}

func evalSymlinksUnder(base, relPath string) (string, bool) {
	currentRel := filepath.Clean(relPath)
	symlinksEncountered := 0
	const maxSymlinks = 255

	for {
		if symlinksEncountered > maxSymlinks {
			return "", false // Loop, fallback to trigger standard loop error
		}

		parts := splitPath(currentRel)
		currentResolvedPath := base
		resolvedAComponent := false

		for i, p := range parts {
			if p == "" || p == "." {
				continue
			}
			if p == ".." {
				parent := filepath.Dir(currentResolvedPath)
				if !hasPrefix(parent, base) {
					return "", false // Escaped base, fall back
				}
				currentResolvedPath = parent
				continue
			}

			testPath := filepath.Join(currentResolvedPath, p)
			target, err := os.Readlink(testPath)
			if err != nil {
				if os.IsNotExist(err) {
					return "", false
				}
				// Regular file/dir, append and continue
				currentResolvedPath = testPath
				continue
			}

			symlinksEncountered++
			resolvedAComponent = true

			if filepath.IsAbs(target) {
				if hasPrefix(target, base) {
					relTarget, err := filepath.Rel(base, target)
					if err != nil {
						return "", false
					}
					remaining := filepath.Join(parts[i+1:]...)
					currentRel = filepath.Clean(filepath.Join(relTarget, remaining))
					break
				}
				return "", false
			}

			relCurrent, err := filepath.Rel(base, currentResolvedPath)
			if err != nil {
				return "", false
			}
			if relCurrent == "." {
				relCurrent = ""
			}

			newRel := filepath.Clean(filepath.Join(relCurrent, target))
			if strings.HasPrefix(newRel, "..") {
				return "", false
			}

			remaining := filepath.Join(parts[i+1:]...)
			currentRel = filepath.Clean(filepath.Join(newRel, remaining))
			break
		}

		if !resolvedAComponent {
			return currentResolvedPath, true
		}
	}
}

func run(args []string, stdin io.Reader, stdout io.Writer) int {
	flags, err := common.ParseFlags(args, spec)
	if err != nil {
		fmt.Fprintf(os.Stderr, "readlink: %v\n", err)
		return 2
	}
	jsonMode := flags.Has("json")
	if len(flags.Positional) == 0 {
		fmt.Fprintln(os.Stderr, "readlink: missing operand")
		return 1
	}
	exitCode := 0
	for _, p := range flags.Positional {
		result, err := Run(p, flags.Has("f"))
		if err != nil {
			fmt.Fprintf(os.Stderr, "readlink: %v\n", err)
			common.RenderError("readlink", 1, "EREADLINK", err.Error(), jsonMode, stdout)
			exitCode = 1
			continue
		}
		common.Render("readlink", result, jsonMode, stdout, func() {
			fmt.Fprintln(stdout, result.Target)
		})
	}
	return exitCode
}

func init() {
	dispatch.Register(dispatch.Command{Name: "readlink", Usage: "Print resolved symbolic links", Run: run})
}
