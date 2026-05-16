// Package truefalse implements the POSIX true and false utilities.
// They are combined here because both are trivially simple.
package truefalse

import (
	"fmt"
	"io"
	"os"

	"github.com/ramayac/korego/internal/dispatch"
	"github.com/ramayac/korego/pkg/common"
)

// BoolResult is the structured result for --json mode.
type BoolResult struct {
	ExitCode int  `json:"exitCode"`
	Value    bool `json:"value"`
}

var spec = common.FlagSpec{
	Defs: []common.FlagDef{
		{Short: "j", Long: "json", Type: common.FlagBool},
	},
}

func runTrue(args []string, out io.Writer) int {
	flags, err := common.ParseFlags(args, spec)
	if err != nil {
		fmt.Fprintf(os.Stderr, "true: %v\n", err)
		return 2
	}
	if flags.Has("json") {
		common.Render("true", BoolResult{ExitCode: 0, Value: true}, true, out, func() {})
	}
	return 0
}

func runFalse(args []string, out io.Writer) int {
	flags, err := common.ParseFlags(args, spec)
	if err != nil {
		fmt.Fprintf(os.Stderr, "false: %v\n", err)
		return 2
	}
	if flags.Has("json") {
		common.Render("false", BoolResult{ExitCode: 1, Value: false}, true, out, func() {})
	}
	return 1
}

func init() {
	dispatch.Register(dispatch.Command{
		Name:  "true",
		Usage: "Return true (exit 0)",
		Run:   runTrue,
	})
	dispatch.Register(dispatch.Command{
		Name:  "false",
		Usage: "Return false (exit 1)",
		Run:   runFalse,
	})
}
