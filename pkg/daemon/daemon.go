// Package daemon implements the 'daemon' utility.
package daemon

import (
	"fmt"
	"os"
	"strconv"

	"github.com/ramayac/korego/internal/daemon"
	"github.com/ramayac/korego/internal/dispatch"
	"github.com/ramayac/korego/pkg/common"
	"io"
)

var spec = common.FlagSpec{
	Defs: []common.FlagDef{
		{Short: "s", Long: "socket", Type: common.FlagValue},
		{Short: "w", Long: "workers", Type: common.FlagValue},
		{Short: "l", Long: "listen-addr", Type: common.FlagValue},
	},
}

func run(args []string, out io.Writer) int {
	flags, err := common.ParseFlags(args, spec)
	if err != nil {
		fmt.Fprintf(os.Stderr, "daemon: %v\n", err)
		return 2
	}

	socket := "/var/run/korego.sock"
	if s := flags.Get("s"); s != "" {
		socket = s
	}

	workers := 4
	if w := flags.Get("w"); w != "" {
		if v, err := strconv.Atoi(w); err == nil && v > 0 {
			workers = v
		}
	}

	httpAddr := flags.Get("l")

	if err := daemon.RunDaemon(socket, workers, httpAddr); err != nil {
		fmt.Fprintf(os.Stderr, "daemon: %v\n", err)
		return 1
	}

	return 0
}

func init() {
	dispatch.Register(dispatch.Command{Name: "daemon", Usage: "Start the persistent JSON-RPC server", Run: run})
}
