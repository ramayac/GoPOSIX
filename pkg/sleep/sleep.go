package sleep

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"time"

	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
)

// SleepResult is the structured result for --json mode.
type SleepResult struct {
	Duration    float64 `json:"duration"`
	Requested   float64 `json:"requested"`
	Interrupted bool    `json:"interrupted"`
}

var spec = common.FlagSpec{
	Defs: []common.FlagDef{
		{Long: "json", Type: common.FlagBool},
	},
}

func run(args []string, out io.Writer) int {
	flags, err := common.ParseFlags(args, spec)
	if err != nil {
		fmt.Fprintf(os.Stderr, "sleep: %v\n", err)
		return 1
	}
	jsonMode := flags.Has("json")

	if len(flags.Positional) == 0 {
		if jsonMode {
			common.RenderError("sleep", 1, "MISSING", "missing operand", true, out)
		}
		fmt.Fprintln(os.Stderr, "sleep: missing operand")
		return 1
	}

	durStr := flags.Positional[0]
	var d time.Duration
	var requested float64

	// Try Go duration first
	dur, err := time.ParseDuration(durStr)
	if err == nil {
		d = dur
		requested = dur.Seconds()
	} else {
		// Try float seconds
		sec, err := strconv.ParseFloat(durStr, 64)
		if err != nil {
			if jsonMode {
				common.RenderError("sleep", 1, "INVALID", fmt.Sprintf("invalid time interval %q", durStr), true, out)
			}
			fmt.Fprintf(os.Stderr, "sleep: invalid time interval %q\n", durStr)
			return 1
		}
		d = time.Duration(sec * float64(time.Second))
		requested = sec
	}

	start := time.Now()
	time.Sleep(d)
	actual := time.Since(start).Seconds()

	if jsonMode {
		common.Render("sleep", SleepResult{
			Duration:    actual,
			Requested:   requested,
			Interrupted: false,
		}, true, out, func() {})
	}
	return 0
}

func init() {
	dispatch.Register(dispatch.Command{Name: "sleep", Usage: "Delay for a specified amount of time", Run: run})
}
