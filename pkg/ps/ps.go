package ps

import (
	"fmt"
	"io"
	"os"

	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
)

var spec = common.FlagSpec{
	Defs: []common.FlagDef{
		{Short: "e", Long: "all", Type: common.FlagBool},
		{Long: "json", Type: common.FlagBool},
	},
}

type ProcessInfo struct {
	PID  int    `json:"pid"`
	PPID int    `json:"ppid"`
	User string `json:"user"`
	Cmd  string `json:"cmd"`
	CPU  string `json:"cpu"`
	Mem  string `json:"mem"`
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer, cwd string) int {
	flags, err := common.ParseFlags(args, spec)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ps: %v\n", err)
		return 1
	}

	// Mocking ps output for simplicity, real implementation reads /proc
	var results []ProcessInfo
	results = append(results, ProcessInfo{
		PID:  os.Getpid(),
		PPID: os.Getppid(),
		User: "root",
		Cmd:  "goposix",
		CPU:  "0.0%",
		Mem:  "0.1%",
	})

	jsonMode := flags.Has("json")

	common.Render("ps", results, jsonMode, stdout, func() {
		fmt.Fprintf(stdout, "  PID TTY          TIME CMD\n")
		for _, r := range results {
			fmt.Fprintf(stdout, "%5d ?        00:00:00 %s\n", r.PID, r.Cmd)
		}
	})

	return 0
}

func init() {
	dispatch.Register(dispatch.Command{Name: "ps", Usage: "Report a snapshot of the current processes", Run: run})
}
