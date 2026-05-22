// Package logname implements the POSIX logname utility — print the user's login name.
package logname

import (
	"fmt"
	"io"
	"os"
	"os/user"

	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
)

// LognameResult is the --json output.
type LognameResult struct {
	Logname string `json:"logname"`
}

var spec = common.FlagSpec{
	Defs: []common.FlagDef{
		{Long: "json", Type: common.FlagBool},
	},
}

// Run returns the login name of the user.
// POSIX: returns the name from getlogin(), falling back to LOGNAME env.
func Run() (LognameResult, error) {
	// Try LOGNAME environment variable first (POSIX recommendation)
	if name := os.Getenv("LOGNAME"); name != "" {
		return LognameResult{Logname: name}, nil
	}
	// Fall back to current user
	u, err := user.Current()
	if err != nil {
		return LognameResult{}, fmt.Errorf("cannot determine login name: %w", err)
	}
	return LognameResult{Logname: u.Username}, nil
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer, cwd string) int {
	flags, err := common.ParseFlags(args, spec)
	if err != nil {
		fmt.Fprintf(os.Stderr, "logname: %v\n", err)
		return 2
	}
	jsonMode := flags.Has("json")

	result, err := Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "logname: %v\n", err)
		common.RenderError("logname", 1, "ELOGNAME", err.Error(), jsonMode, stdout)
		return 1
	}

	common.Render("logname", result, jsonMode, stdout, func() {
		fmt.Fprintln(stdout, result.Logname)
	})
	return 0
}

func init() {
	dispatch.Register(dispatch.Command{
		Name:  "logname",
		Usage: "Print the user's login name",
		Run:   run,
	})
}
