// Package printenv implements the POSIX printenv utility.
package printenv

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
)

// PrintenvResult is the structured result for --json mode.
type PrintenvResult struct {
	Vars map[string]string `json:"vars"`
}

var spec = common.FlagSpec{
	Defs: []common.FlagDef{
		{Long: "json", Type: common.FlagBool},
	},
}

// Run returns the environment variables, optionally filtered to the given names.
func Run(names []string) PrintenvResult {
	vars := make(map[string]string)
	if len(names) == 0 {
		for _, e := range os.Environ() {
			parts := strings.SplitN(e, "=", 2)
			if len(parts) == 2 {
				vars[parts[0]] = parts[1]
			}
		}
	} else {
		for _, name := range names {
			val, ok := os.LookupEnv(name)
			if ok {
				vars[name] = val
			}
		}
	}
	return PrintenvResult{Vars: vars}
}

func run(args []string, stdin io.Reader, stdout io.Writer) int {
	flags, err := common.ParseFlags(args, spec)
	if err != nil {
		fmt.Fprintf(os.Stderr, "printenv: %v\n", err)
		return 2
	}
	jsonMode := flags.Has("json")
	result := Run(flags.Positional)

	exitCode := 0
	// printenv exits 1 if a named variable does not exist.
	if len(flags.Positional) > 0 {
		for _, name := range flags.Positional {
			if _, ok := result.Vars[name]; !ok {
				exitCode = 1
			}
		}
	}

	common.Render("printenv", result, jsonMode, stdout, func() {
		for k, v := range result.Vars {
			fmt.Fprintf(stdout, "%s=%s\n", k, v)
		}
	})
	return exitCode
}

func init() {
	dispatch.Register(dispatch.Command{
		Name:  "printenv",
		Usage: "Print all or specified environment variables",
		Run:   run,
	})
}
