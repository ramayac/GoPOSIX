// Package which implements the POSIX-compliant which utility.
package which

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
)

// WhichResult contains the search results.
type WhichResult struct {
	Matches map[string][]string `json:"matches"`
}

var spec = common.FlagSpec{
	Defs: []common.FlagDef{
		{Short: "a", Long: "all", Type: common.FlagBool},
		{Long: "json", Type: common.FlagBool},
	},
}

// isExecutable checks if the path exists, is a regular file, and is executable.
func isExecutable(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	if info.IsDir() {
		return false
	}
	// Check if any of the user, group, or other execute bits are set
	return info.Mode().Perm()&0111 != 0
}

// findCommand searches for a command in the provided path directories.
func findCommand(cmd string, pathDirs []string, all bool) []string {
	var matches []string

	// If the command contains a slash, bypass PATH lookup and check directly
	if filepath.Base(cmd) != cmd {
		if isExecutable(cmd) {
			matches = append(matches, cmd)
		}
		return matches
	}

	for _, dir := range pathDirs {
		if dir == "" {
			dir = "."
		}
		path := filepath.Join(dir, cmd)
		if isExecutable(path) {
			matches = append(matches, path)
			if !all {
				break
			}
		}
	}
	return matches
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer, cwd string) int {
	flags, err := common.ParseFlags(args, spec)
	if err != nil {
		fmt.Fprintf(stderr, "which: %v\n", err)
		return 2
	}

	all := flags.Has("all")
	jsonMode := flags.Has("json")

	posArgs := flags.Positional
	if len(posArgs) == 0 {
		// POSIX says if no arguments, exit status is 0 or 1.
		// Standard which usually prints usage or exits. Let's exit with 1.
		fmt.Fprintf(stderr, "which: missing argument\n")
		return 1
	}

	pathEnv := os.Getenv("PATH")
	if pathEnv == "" {
		pathEnv = "/usr/bin:/bin:/usr/sbin:/sbin"
	}
	pathDirs := filepath.SplitList(pathEnv)

	result := WhichResult{
		Matches: make(map[string][]string),
	}

	anyFailed := false
	for _, cmd := range posArgs {
		matches := findCommand(cmd, pathDirs, all)
		if len(matches) == 0 {
			anyFailed = true
		} else {
			result.Matches[cmd] = matches
		}
	}

	common.Render("which", result, jsonMode, stdout, func() {
		for _, cmd := range posArgs {
			if matches, exists := result.Matches[cmd]; exists {
				for _, match := range matches {
					fmt.Fprintln(stdout, match)
				}
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
		Name:  "which",
		Usage: "Locate a command",
		Run:   run,
	})
}
