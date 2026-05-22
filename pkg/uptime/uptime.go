// Package uptime implements the POSIX uptime utility.
package uptime

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
	"github.com/ramayac/goposix/pkg/who"
)

// UptimeResult represents the machine-readable structure returned in --json mode.
type UptimeResult struct {
	CurrentTime string  `json:"current_time"`
	Uptime      float64 `json:"uptime"`
	Users       int     `json:"users"`
	Load1m      float64 `json:"load_1m"`
	Load5m      float64 `json:"load_5m"`
	Load15m     float64 `json:"load_15m"`
}

var spec = common.FlagSpec{
	Defs: []common.FlagDef{
		{Long: "json", Type: common.FlagBool},
	},
}

// Variables for mocking proc reader functions in unit tests.
var (
	readProcUptime = func() ([]byte, error) {
		return os.ReadFile("/proc/uptime")
	}
	readProcLoadavg = func() ([]byte, error) {
		return os.ReadFile("/proc/loadavg")
	}
	mockUserCount = -1
)

func run(args []string, stdin io.Reader, stdout, stderr io.Writer, cwd string) int {
	flags, err := common.ParseFlags(args, spec)
	if err != nil {
		fmt.Fprintf(stderr, "uptime: %v\n", err)
		return 2
	}

	jsonMode := flags.Has("json")

	// Parse uptime from /proc/uptime
	var uptimeSec float64
	upBytes, err := readProcUptime()
	if err == nil {
		fields := strings.Fields(string(upBytes))
		if len(fields) > 0 {
			if uVal, err := strconv.ParseFloat(fields[0], 64); err == nil {
				uptimeSec = uVal
			}
		}
	}

	// Parse load averages from /proc/loadavg
	var load1m, load5m, load15m float64
	loadBytes, err := readProcLoadavg()
	if err == nil {
		fields := strings.Fields(string(loadBytes))
		if len(fields) >= 3 {
			l1, e1 := strconv.ParseFloat(fields[0], 64)
			l5, e5 := strconv.ParseFloat(fields[1], 64)
			l15, e15 := strconv.ParseFloat(fields[2], 64)
			if e1 == nil && e5 == nil && e15 == nil {
				load1m, load5m, load15m = l1, l5, l15
			}
		}
	}

	// Determine user count
	var numUsers int
	if mockUserCount != -1 {
		numUsers = mockUserCount
	} else {
		if whoRes, err := who.Run(); err == nil {
			numUsers = whoRes.Count
		}
	}

	// Get current time
	now := time.Now()
	currentTimeStr := now.Format("15:04:05")

	result := UptimeResult{
		CurrentTime: currentTimeStr,
		Uptime:      uptimeSec,
		Users:       numUsers,
		Load1m:      load1m,
		Load5m:      load5m,
		Load15m:     load15m,
	}

	common.Render("uptime", result, jsonMode, stdout, func() {
		// Example: " 08:12:48 up 34 min,  1 user,  load average: 0.92, 0.74, 0.66"
		// or " 15:13:02 up 14 days,  3:12,  5 users,  load average: 0.12, 0.25, 0.18"
		fmt.Fprintf(stdout, " %s up ", currentTimeStr)

		uptimeVal := int(uptimeSec)
		days := uptimeVal / (24 * 3600)
		hours := (uptimeVal % (24 * 3600)) / 3600
		minutes := (uptimeVal % 3600) / 60

		if days > 0 {
			if days == 1 {
				fmt.Fprint(stdout, "1 day, ")
			} else {
				fmt.Fprintf(stdout, "%d days, ", days)
			}
			fmt.Fprintf(stdout, "%2d:%02d, ", hours, minutes)
		} else {
			if hours > 0 {
				fmt.Fprintf(stdout, "%2d:%02d, ", hours, minutes)
			} else {
				fmt.Fprintf(stdout, "%d min, ", minutes)
			}
		}

		if numUsers == 1 {
			fmt.Fprint(stdout, " 1 user, ")
		} else {
			fmt.Fprintf(stdout, " %d users, ", numUsers)
		}

		fmt.Fprintf(stdout, " load average: %.2f, %.2f, %.2f\n", load1m, load5m, load15m)
	})

	return 0
}

func init() {
	dispatch.Register(dispatch.Command{
		Name:  "uptime",
		Usage: "Display how long the system has been running",
		Run:   run,
	})
}
