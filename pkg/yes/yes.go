// Package yes implements the POSIX yes utility.
package yes

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/ramayac/korego/internal/dispatch"
	"github.com/ramayac/korego/pkg/common"
)

// run prints a string (default "y") forever until killed.
// Note: yes does not support --json per spec.
func run(args []string, out io.Writer) int {
	flags, err := common.ParseFlags(args, common.FlagSpec{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "yes: %v\n", err)
		return 2
	}

	text := "y"
	if len(flags.Positional) > 0 {
		text = strings.Join(flags.Positional, " ")
	}

	// Gracefully handle SIGPIPE (e.g. `yes | head -n 5`).
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGPIPE)

	for {
		select {
		case <-sig:
			return 0
		default:
			if _, err := fmt.Fprintln(out, text); err != nil {
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
