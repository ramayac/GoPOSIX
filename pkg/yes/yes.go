// Package yes implements the POSIX yes utility.
package yes

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
)

// YesResult is the structured result for --json mode.
type YesResult struct {
	String    string `json:"string"`
	Count     int    `json:"count"`
	Truncated bool   `json:"truncated"`
}

var spec = common.FlagSpec{
	Defs: []common.FlagDef{
		{Long: "json", Type: common.FlagBool},
		{Short: "n", Long: "count", Type: common.FlagValue},
	},
}

// run prints a string (default "y") forever until killed.
func run(args []string, stdin io.Reader, stdout, stderr io.Writer, cwd string) int {
	flags, err := common.ParseFlags(args, spec)
	if err != nil {
		fmt.Fprintf(os.Stderr, "yes: %v\n", err)
		return 2
	}
	jsonMode := flags.Has("json")

	text := "y"
	if len(flags.Positional) > 0 {
		text = strings.Join(flags.Positional, " ")
	}

	// In JSON mode, output only the JSON envelope (text data is in the result).
	if jsonMode {
		count := 1
		if cntStr := flags.Get("n"); cntStr != "" {
			if cntStr == "count" {
				fmt.Fprintf(os.Stderr, "yes: --count requires a value\n")
				return 1
			}
			fmt.Sscanf(cntStr, "%d", &count)
		}
		common.Render("yes", YesResult{
			String:    text,
			Count:     count,
			Truncated: true,
		}, true, stdout, func() {})
		return 0
	}

	// Gracefully handle SIGPIPE (e.g. `yes | head -n 5`).
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGPIPE)

	for {
		select {
		case <-sig:
			return 0
		default:
			if _, err := fmt.Fprintln(stdout, text); err != nil {
				return 0 // broken pipe
			}
		}
	}
}

func init() {
	dispatch.Register(dispatch.Command{
		Name:  "yes",
		Usage: "Output a string repeatedly until killed",
		Run:   run,
	})
}
