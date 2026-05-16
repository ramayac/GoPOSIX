// Package env implements the GNU/POSIX env utility.
package env

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ramayac/korego/internal/dispatch"
	"github.com/ramayac/korego/pkg/common"
)

// EnvResult is the structured result for --json mode.
type EnvResult struct {
	Vars map[string]string `json:"vars"`
}

var spec = common.FlagSpec{
	Defs: []common.FlagDef{
		{Short: "i", Long: "ignore-environment", Type: common.FlagBool},
		{Short: "j", Long: "json", Type: common.FlagBool},
	},
}

// Run collects the environment.  If ignoreEnv is true the host environment
// is excluded; only VAR=val assignments from the positional args are used.
func Run(ignoreEnv bool, positional []string) EnvResult {
	vars := make(map[string]string)

	if !ignoreEnv {
		for _, e := range os.Environ() {
			parts := strings.SplitN(e, "=", 2)
			if len(parts) == 2 {
				vars[parts[0]] = parts[1]
			}
		}
	}
	// Overlay VAR=val positional assignments.
	for _, p := range positional {
		if strings.Contains(p, "=") {
			parts := strings.SplitN(p, "=", 2)
			vars[parts[0]] = parts[1]
		}
	}
	return EnvResult{Vars: vars}
}

func run(args []string, out io.Writer) int {
	flags, err := common.ParseFlags(args, spec)
	if err != nil {
		fmt.Fprintf(os.Stderr, "env: %v\n", err)
		return 2
	}
	jsonMode := flags.Has("json")
	ignoreEnv := flags.Has("i")
	result := Run(ignoreEnv, flags.Positional)

	common.Render("env", result, jsonMode, out, func() {
		for k, v := range result.Vars {
			fmt.Fprintf(out, "%s=%s\n", k, v)
		}
	})
	return 0
}

func init() {
	dispatch.Register(dispatch.Command{
		Name:  "env",
		Usage: "Print environment or run a program in a modified environment",
		Run:   run,
	})
}
