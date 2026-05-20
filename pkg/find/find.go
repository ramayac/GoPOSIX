package find

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
)

var spec = common.FlagSpec{
	Defs: []common.FlagDef{
		{Long: "json", Type: common.FlagBool},
		{Short: "n", Long: "name", Type: common.FlagValue},
		{Short: "t", Long: "type", Type: common.FlagValue},
		{Short: "d", Long: "maxdepth", Type: common.FlagValue},
		// -xdev: stay on this filesystem (no-op in goposix; filepath.WalkDir
		// already respects filesystem boundaries).
		{Short: "x", Long: "xdev", Type: common.FlagBool},
		{Short: "m", Long: "mtime", Type: common.FlagValue},
	},
}

// getMaxDepth extracts the -maxdepth N value, returning -1 if not set (unlimited).
func getMaxDepth(flags *common.ParseResult) int {
	dStr := flags.Get("d")
	if dStr == "" {
		return -1
	}
	d, err := strconv.Atoi(dStr)
	if err != nil || d < 0 {
		return -1
	}
	return d
}

type FileInfo struct {
	Path  string `json:"path"`
	Type  string `json:"type"`
	Size  int64  `json:"size"`
	Mtime string `json:"mtime"`
}

func run(args []string, stdin io.Reader, stdout io.Writer) int {
	return findRun(args, stdout, os.Stderr, os.Stdin)
}

func findRun(args []string, stdout io.Writer, errOut io.Writer, stdin io.Reader) int {
	var execCmd []string
	execPlus := false

	specCopy := spec
	specCopy.PreProcess = func(in []string) ([]string, error) {
		var flagArgs []string
		for i := 0; i < len(in); i++ {
			a := in[i]
			if a == "-exec" {
				i++
				for i < len(in) && in[i] != ";" && in[i] != "+" {
					execCmd = append(execCmd, in[i])
					i++
				}
				if i < len(in) && in[i] == "+" {
					execPlus = true
				}
			} else if strings.HasPrefix(a, "--") {
				flagArgs = append(flagArgs, a)
			} else if strings.HasPrefix(a, "-") && len(a) > 2 {
				flagArgs = append(flagArgs, "-"+a)
			} else {
				flagArgs = append(flagArgs, a)
			}
		}
		return flagArgs, nil
	}

	flags, err := common.ParseFlags(args, specCopy)
	if err != nil {
		fmt.Fprintf(errOut, "find: %v\n", err)
		return 1
	}

	root := "."
	if len(flags.Positional) > 0 {
		root = flags.Positional[0]
	}

	namePattern := flags.Get("n")
	typeFilter := flags.Get("t")
	maxDepth := getMaxDepth(flags)

	// Normalize root for depth counting.
	rootClean := filepath.Clean(root)

	var results []FileInfo

	err = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			fmt.Fprintf(errOut, "find: %s: %v\n", p, err)
			return nil
		}

		// Compute depth relative to root.
		if maxDepth >= 0 {
			depth := 0
			if p != rootClean {
				rel := p
				if strings.HasPrefix(rel, rootClean+"/") || rel == rootClean {
					rel = rel[len(rootClean):]
				}
				rel = strings.TrimLeft(rel, "/")
				if rel != "" {
					depth = strings.Count(rel, string(filepath.Separator)) + 1
				}
			}
			if depth > maxDepth {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		if namePattern != "" {
			match, _ := filepath.Match(namePattern, filepath.Base(p))
			if !match {
				return nil
			}
		}

		tStr := "f"
		if d.IsDir() {
			tStr = "d"
		} else if d.Type()&fs.ModeSymlink != 0 {
			tStr = "l"
		}

		if typeFilter != "" && tStr != typeFilter {
			return nil
		}

		info, _ := d.Info()
		size := int64(0)
		mtime := ""
		if info != nil {
			size = info.Size()
			mtime = info.ModTime().Format(time.RFC3339)
		}

		// Normalize path: when root is ".", WalkDir returns paths without "./"
		outPath := p
		if rootClean == "." && p != "." && !strings.HasPrefix(p, ".") {
			outPath = "./" + p
		}

		results = append(results, FileInfo{
			Path:  outPath,
			Type:  tStr,
			Size:  size,
			Mtime: mtime,
		})

		return nil
	})

	if err != nil {
		return 1
	}

	// Execute -exec commands if present.
	if len(execCmd) > 0 {
		return runExec(results, execCmd, execPlus, stdout, errOut)
	}

	jsonMode := flags.Has("json")

	common.Render("find", results, jsonMode, stdout, func() {
		for _, r := range results {
			fmt.Fprintln(stdout, r.Path)
		}
	})

	return 0
}

// runExec executes a command for each matched file (or in batches with +).
func runExec(results []FileInfo, execCmd []string, execPlus bool, stdout io.Writer, errOut io.Writer) int {
	if len(results) == 0 {
		return 0
	}
	maxExit := 0

	if execPlus {
		// + mode: batch all paths together in one invocation.
		args := buildExecArgs(execCmd, results)
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Stdout = stdout
		cmd.Stderr = errOut
		err := cmd.Run()
		if err != nil {
			if ee, ok := err.(*exec.ExitError); ok {
				code := ee.ExitCode()
				if code > maxExit {
					maxExit = code
				}
			} else {
				maxExit = 1
			}
		}
	} else {
		// \; mode: one invocation per matched file.
		// BusyBox: command exit code is ignored for \; mode.
		for _, r := range results {
			args := buildExecArgs(execCmd, []FileInfo{r})
			cmd := exec.Command(args[0], args[1:]...)
			cmd.Stdout = stdout
			cmd.Stderr = errOut
			cmd.Run() // ignore exit code
		}
	}
	return maxExit
}

// buildExecArgs builds the argument list with {} replaced by file paths.
func buildExecArgs(execCmd []string, files []FileInfo) []string {
	var args []string
	for _, a := range execCmd {
		if a == "{}" {
			for _, f := range files {
				args = append(args, f.Path)
			}
		} else {
			args = append(args, a)
		}
	}
	return args
}

func init() {
	dispatch.Register(dispatch.Command{Name: "find", Usage: "Search for files in a directory hierarchy", Run: run})
}
