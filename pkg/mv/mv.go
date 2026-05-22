// Package mv implements the POSIX mv utility.
package mv

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
)

// MoveRecord records a single move operation.
type MoveRecord struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// MvResult is the --json output.
type MvResult struct {
	Moved []MoveRecord `json:"moved"`
}

var spec = common.FlagSpec{
	Defs: []common.FlagDef{
		{Short: "f", Long: "force", Type: common.FlagBool},
		{Short: "i", Long: "interactive", Type: common.FlagBool},
		{Short: "t", Long: "target-directory", Type: common.FlagValue},
		{Long: "json", Type: common.FlagBool},
	},
}

// moveFile performs a rename, falling back to cross-device copy+delete.
func moveFile(src, dst string) error {
	err := os.Rename(src, dst)
	if err == nil {
		return nil
	}
	// Cross-device fallback: copy then remove.
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	si, _ := in.Stat()
	stdout, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, si.Mode())
	if err != nil {
		return err
	}
	defer stdout.Close()
	if _, err := io.Copy(stdout, in); err != nil {
		return err
	}
	return os.Remove(src)
}

// Run moves src paths to dst.
func Run(srcs []string, dst string) (MvResult, error) {
	var result MvResult
	dstInfo, dstErr := os.Stat(dst)
	dstIsDir := dstErr == nil && dstInfo.IsDir()

	for _, src := range srcs {
		target := dst
		if dstIsDir {
			target = filepath.Join(dst, filepath.Base(src))
		}
		if err := moveFile(src, target); err != nil {
			return result, err
		}
		result.Moved = append(result.Moved, MoveRecord{From: src, To: target})
	}
	return result, nil
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer, cwd string) int {
	flags, err := common.ParseFlags(args, spec)
	if err != nil {
		fmt.Fprintf(os.Stderr, "mv: %v\n", err)
		return 2
	}
	jsonMode := flags.Has("json")
	targetDir := flags.Get("t")

	var srcs []string
	var dst string
	if targetDir != "" {
		// GNU -t DIR SOURCE...: target specified before sources.
		srcs = flags.Positional
		dst = targetDir
	} else {
		if len(flags.Positional) < 2 {
			fmt.Fprintln(os.Stderr, "mv: missing file operand")
			return 1
		}
		srcs = flags.Positional[:len(flags.Positional)-1]
		dst = flags.Positional[len(flags.Positional)-1]
	}
	result, err := Run(srcs, dst)
	if err != nil {
		fmt.Fprintf(os.Stderr, "mv: %v\n", err)
		common.RenderError("mv", 1, "EMV", err.Error(), jsonMode, stdout)
		return 1
	}
	common.Render("mv", result, jsonMode, stdout, func() {})
	return 0
}

func init() {
	dispatch.Register(dispatch.Command{Name: "mv", Usage: "Move (rename) files", Run: run})
}
