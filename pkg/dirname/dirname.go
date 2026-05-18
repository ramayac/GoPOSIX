// Package dirname implements the POSIX dirname utility.
package dirname

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
)

// DirnameResult is the --json output.
type DirnameResult struct {
	Result string `json:"result"`
}

var spec = common.FlagSpec{
	Defs: []common.FlagDef{
		{Long: "json", Type: common.FlagBool},
	},
}

// Run returns the directory component of path.
func Run(path string) DirnameResult {
	return DirnameResult{Result: filepath.Dir(path)}
}

func run(args []string, out io.Writer) int {
	flags, err := common.ParseFlags(args, spec)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dirname: %v\n", err)
		return 2
	}
	if len(flags.Positional) == 0 {
		fmt.Fprintln(os.Stderr, "dirname: missing operand")
		return 1
	}
	jsonMode := flags.Has("json")
	result := Run(flags.Positional[0])
	common.Render("dirname", result, jsonMode, out, func() {
		fmt.Println(result.Result)
	})
	return 0
}

func init() {
	dispatch.Register(dispatch.Command{Name: "dirname", Usage: "Strip last component from file name", Run: run})
}
