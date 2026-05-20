package kill

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"syscall"

	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
)

var spec = common.FlagSpec{
	Defs: []common.FlagDef{
		{Long: "json", Type: common.FlagBool},
		{Short: "9", Long: "kill", Type: common.FlagBool}, // cheat a bit for short flags
	},
}

type KillResult struct {
	PID     int    `json:"pid"`
	Signal  string `json:"signal"`
	Success bool   `json:"success"`
}

type KillResp struct {
	Signaled []KillResult `json:"signaled"`
}

func run(args []string, stdin io.Reader, stdout io.Writer) int {
	flags, err := common.ParseFlags(args, spec)
	if err != nil {
		fmt.Fprintf(os.Stderr, "kill: %v\n", err)
		return 1
	}

	sig := syscall.SIGTERM
	sigName := "SIGTERM"
	if flags.Has("9") {
		sig = syscall.SIGKILL
		sigName = "SIGKILL"
	}

	var res []KillResult
	exitCode := 0

	for _, p := range flags.Positional {
		pid, err := strconv.Atoi(p)
		if err != nil {
			fmt.Fprintf(os.Stderr, "kill: %s: arguments must be process or job IDs\n", p)
			exitCode = 1
			continue
		}

		err = syscall.Kill(pid, sig)
		res = append(res, KillResult{
			PID:     pid,
			Signal:  sigName,
			Success: err == nil,
		})

		if err != nil {
			fmt.Fprintf(os.Stderr, "kill: (%d) - %v\n", pid, err)
			exitCode = 1
		}
	}

	if flags.Has("json") {
		common.Render("kill", KillResp{Signaled: res}, true, stdout, func() {})
	}

	return exitCode
}

func init() {
	dispatch.Register(dispatch.Command{Name: "kill", Usage: "Send a signal to a process", Run: run})
}
