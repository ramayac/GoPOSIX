// Package basename implements the POSIX basename utility.
package basename

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
)

// BasenameResult is the --json output.
type BasenameResult struct {
	Result string `json:"result"`
}

var spec = common.FlagSpec{
	Defs: []common.FlagDef{
		{Long: "json", Type: common.FlagBool},
	},
}

// Run returns the base name of path, optionally stripping suffix.
// POSIX: suffix is only stripped if it is NOT identical to the entire
// remaining string AND it matches a suffix of that string.
func Run(path, suffix string) BasenameResult {
	base := filepath.Base(path)
	if suffix != "" && suffix != base && strings.HasSuffix(base, suffix) {
		base = base[:len(base)-len(suffix)]
	}
	return BasenameResult{Result: base}
}

func run(args []string, out io.Writer) int {
	flags, err := common.ParseFlags(args, spec)
	if err != nil {
		fmt.Fprintf(os.Stderr, "basename: %v\n", err)
		return 2
	}
	if len(flags.Positional) == 0 {
		fmt.Fprintln(os.Stderr, "basename: missing operand")
		return 1
	}
	jsonMode := flags.Has("json")
	path := flags.Positional[0]
	suffix := ""
	if len(flags.Positional) >= 2 {
		suffix = flags.Positional[1]
	}
	result := Run(path, suffix)
	common.Render("basename", result, jsonMode, out, func() {
		fmt.Fprintln(out, result.Result)
	})
	return 0
}

func init() {
	dispatch.Register(dispatch.Command{Name: "basename", Usage: "Strip directory and suffix from filenames", Run: run})
}
