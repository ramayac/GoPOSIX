// Package startstopdaemon implements the POSIX-compliant start-stop-daemon utility.
package startstopdaemon

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
)

var spec = common.FlagSpec{
	Defs: []common.FlagDef{
		{Short: "S", Long: "start", Type: common.FlagBool},
		{Short: "K", Long: "stop", Type: common.FlagBool},
		{Short: "x", Long: "exec", Type: common.FlagValue},
		{Short: "a", Long: "startas", Type: common.FlagValue},
		{Short: "p", Long: "pidfile", Type: common.FlagValue},
		{Short: "n", Long: "name", Type: common.FlagValue},
		{Short: "u", Long: "user", Type: common.FlagValue},
		{Short: "s", Long: "signal", Type: common.FlagValue},
		{Short: "b", Long: "background", Type: common.FlagBool},
		{Short: "m", Long: "make-pidfile", Type: common.FlagBool},
		{Short: "q", Long: "quiet", Type: common.FlagBool},
		{Short: "t", Long: "test", Type: common.FlagBool},
		{Short: "o", Long: "oknodo", Type: common.FlagBool},
		{Short: "h", Long: "help", Type: common.FlagBool},
		{Long: "json", Type: common.FlagBool},
	},
}

func init() {
	dispatch.Register(dispatch.Command{
		Name:  "start-stop-daemon",
		Usage: "Start and stop system daemons",
		Run:   run,
	})
}

// DaemonResult represents the JSON response structure.
type DaemonResult struct {
	Action      string   `json:"action"`
	MatchedPids []int    `json:"matchedPids"`
	NewPid      int      `json:"newPid,omitempty"`
	Status      string   `json:"status"`
	CommandLine []string `json:"commandLine,omitempty"`
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer, cwd string) int {
	jsonMode := false
	for _, arg := range args {
		if arg == "--json" {
			jsonMode = true
			break
		}
	}

	flags, err := common.ParseFlags(args, spec)
	if err != nil {
		if jsonMode {
			common.RenderError("start-stop-daemon", 1, "FLAG_ERROR", err.Error(), true, stderr)
		} else {
			fmt.Fprintf(stderr, "start-stop-daemon: %v\n", err)
		}
		return 1
	}

	if flags.Has("h") || flags.Has("help") {
		helpText := "Usage: start-stop-daemon [-SK] [OPTIONS] [-- [ARGS...]]\n\n" +
			"Start and stop system daemons.\n\n" +
			"Options:\n" +
			"  -S, --start          Start daemon\n" +
			"  -K, --stop           Stop daemon\n" +
			"  -x, --exec PATH      Match / Execute program at PATH\n" +
			"  -a, --startas PATH   Execute program at PATH (argv0)\n" +
			"  -p, --pidfile FILE   Match PID from FILE\n" +
			"  -n, --name NAME      Match processes with NAME\n" +
			"  -u, --user USER      Match processes owned by USER\n" +
			"  -s, --signal SIG     Signal to send when stopping (default: TERM)\n" +
			"  -b, --background     Force process into background\n" +
			"  -m, --make-pidfile   Create pidfile specified by -p\n" +
			"  -q, --quiet          Quiet mode\n" +
			"  -t, --test           Test mode (dry run)\n" +
			"  -o, --oknodo         Exit with status 0 instead of 1 if nothing is done"
		common.Render("start-stop-daemon", struct {
			Help string `json:"help"`
		}{Help: helpText}, jsonMode, stdout, func() {
			fmt.Fprintln(stdout, helpText)
		})
		return 0
	}

	startMode := flags.Has("S") || flags.Has("start")
	stopMode := flags.Has("K") || flags.Has("stop")

	if !startMode && !stopMode {
		if jsonMode {
			common.RenderError("start-stop-daemon", 1, "MISSING_ACTION", "must specify --start or --stop", true, stderr)
		} else {
			fmt.Fprintln(stderr, "start-stop-daemon: must specify --start (-S) or --stop (-K)")
		}
		return 1
	}

	execPath := flags.Get("x")
	startAs := flags.Get("a")
	pidFile := flags.Get("p")
	nameFilter := flags.Get("n")
	userFilter := flags.Get("u")
	signalStr := flags.Get("s")
	background := flags.Has("b") || flags.Has("background")
	makePidfile := flags.Has("m") || flags.Has("make-pidfile")
	quiet := flags.Has("q") || flags.Has("quiet")
	testMode := flags.Has("t") || flags.Has("test")
	oknodo := flags.Has("o") || flags.Has("oknodo")

	pos := flags.Positional

	// Resolve Signal
	sig := syscall.SIGTERM
	if signalStr != "" {
		if s, err := parseSignal(signalStr); err == nil {
			sig = s
		} else {
			if jsonMode {
				common.RenderError("start-stop-daemon", 1, "INVALID_SIGNAL", err.Error(), true, stderr)
			} else {
				fmt.Fprintf(stderr, "start-stop-daemon: %v\n", err)
			}
			return 1
		}
	}

	// Match existing processes
	matchedPids := findProcesses(pidFile, execPath, nameFilter, userFilter)

	if stopMode {
		if len(matchedPids) == 0 {
			statusStr := "nothing stopped"
			if jsonMode {
				common.Render("start-stop-daemon", DaemonResult{Action: "stop", MatchedPids: []int{}, Status: statusStr}, true, stdout, nil)
			}
			if oknodo {
				return 0
			}
			return 1
		}

		if !quiet && !jsonMode {
			fmt.Fprintf(stdout, "Stopping processes: %v\n", matchedPids)
		}

		if !testMode {
			for _, pid := range matchedPids {
				_ = syscall.Kill(pid, sig)
			}
		}

		if jsonMode {
			common.Render("start-stop-daemon", DaemonResult{Action: "stop", MatchedPids: matchedPids, Status: "stopped"}, true, stdout, nil)
		}
		return 0
	}

	// Start Mode
	if len(matchedPids) > 0 {
		statusStr := "already running"
		if !quiet && !jsonMode {
			fmt.Fprintf(stdout, "Daemon already running (PIDs: %v).\n", matchedPids)
		}
		if jsonMode {
			common.Render("start-stop-daemon", DaemonResult{Action: "start", MatchedPids: matchedPids, Status: statusStr}, true, stdout, nil)
		}
		if oknodo {
			return 0
		}
		return 1
	}

	// Determine executable and arguments
	var exeToRun string
	if execPath != "" {
		exeToRun = execPath
	} else if startAs != "" {
		exeToRun = startAs
	} else if len(pos) > 0 {
		exeToRun = pos[0]
	} else {
		if jsonMode {
			common.RenderError("start-stop-daemon", 1, "MISSING_COMMAND", "nothing to start", true, stderr)
		} else {
			fmt.Fprintln(stderr, "start-stop-daemon: nothing to start (specify -x, -a, or command)")
		}
		return 1
	}

	argv0 := exeToRun
	if startAs != "" {
		argv0 = startAs
	}

	cmdArgs := []string{argv0}
	if startAs == "" && len(pos) > 1 {
		// positional args are command args
		cmdArgs = pos
	} else if len(pos) > 0 {
		// If using -a, pos represents arguments to append after argv0
		// Wait! In "start-stop-daemon -S -x /bin/false -a qwerty false",
		// /bin/false is exe, qwerty is argv0, false is argv1.
		// So cmdArgs is ["qwerty", "false"].
		if execPath != "" && startAs != "" {
			cmdArgs = append([]string{startAs}, pos...)
		} else {
			cmdArgs = pos
		}
	}

	if testMode {
		if !quiet && !jsonMode {
			fmt.Fprintf(stdout, "Would start %s with args %v\n", exeToRun, cmdArgs)
		}
		if jsonMode {
			common.Render("start-stop-daemon", DaemonResult{Action: "start", MatchedPids: []int{}, Status: "would start", CommandLine: cmdArgs}, true, stdout, nil)
		}
		return 0
	}

	// Make sure the absolute path of the executable is resolved
	resolvedExe := exeToRun
	if !filepath.IsAbs(resolvedExe) {
		if path, err := exec.LookPath(exeToRun); err == nil {
			resolvedExe = path
		}
	}

	cmd := exec.Command(resolvedExe)
	cmd.Args = cmdArgs
	cmd.Dir = cwd
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	if background {
		err := cmd.Start()
		if err != nil {
			if jsonMode {
				common.RenderError("start-stop-daemon", 1, "START_FAILED", err.Error(), true, stderr)
			} else {
				fmt.Fprintf(stderr, "start-stop-daemon: %v\n", err)
			}
			return 1
		}

		pid := cmd.Process.Pid
		if makePidfile && pidFile != "" {
			_ = os.WriteFile(pidFile, []byte(strconv.Itoa(pid)+"\n"), 0644)
		}

		if jsonMode {
			common.Render("start-stop-daemon", DaemonResult{Action: "start", MatchedPids: []int{}, NewPid: pid, Status: "started background", CommandLine: cmdArgs}, true, stdout, nil)
		}
		return 0
	}

	// Foreground mode
	err = cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode()
		}
		if jsonMode {
			common.RenderError("start-stop-daemon", 1, "START_FAILED", err.Error(), true, stderr)
		} else {
			fmt.Fprintf(stderr, "start-stop-daemon: %v\n", err)
		}
		return 1
	}

	if jsonMode {
		common.Render("start-stop-daemon", DaemonResult{Action: "start", MatchedPids: []int{}, Status: "started foreground", CommandLine: cmdArgs}, true, stdout, nil)
	}
	return 0
}

func parseSignal(sigStr string) (syscall.Signal, error) {
	sigStr = strings.ToUpper(sigStr)
	sigStr = strings.TrimPrefix(sigStr, "SIG")

	switch sigStr {
	case "1", "HUP":
		return syscall.SIGHUP, nil
	case "2", "INT":
		return syscall.SIGINT, nil
	case "3", "QUIT":
		return syscall.SIGQUIT, nil
	case "9", "KILL":
		return syscall.SIGKILL, nil
	case "15", "TERM":
		return syscall.SIGTERM, nil
	case "USR1":
		return syscall.SIGUSR1, nil
	case "USR2":
		return syscall.SIGUSR2, nil
	default:
		// Attempt numeric parse
		if val, err := strconv.Atoi(sigStr); err == nil {
			return syscall.Signal(val), nil
		}
		return 0, fmt.Errorf("unknown signal: %s", sigStr)
	}
}

// findProcesses retrieves matching PIDs based on pidfile, executable path, process name, and user criteria.
func findProcesses(pidFile, execPath, nameFilter, userFilter string) []int {
	var pids []int

	if pidFile == "" && execPath == "" && nameFilter == "" && userFilter == "" {
		return nil
	}

	// 1. Match from pidfile
	if pidFile != "" {
		content, err := os.ReadFile(pidFile)
		if err == nil {
			clean := strings.TrimSpace(string(content))
			if pid, err := strconv.Atoi(clean); err == nil && pid > 0 {
				// Verify if process exists
				if syscall.Kill(pid, 0) == nil {
					pids = append(pids, pid)
				}
			}
		}
		return pids
	}

	// 2. Scan /proc for matches (Linux optimized)
	procDirs, err := os.ReadDir("/proc")
	if err != nil {
		return pids // non-Linux stub or proc unreadable
	}

	for _, d := range procDirs {
		if !d.IsDir() {
			continue
		}
		pid, err := strconv.Atoi(d.Name())
		if err != nil || pid <= 0 {
			continue
		}

		// Verify exec path match
		if execPath != "" {
			link, err := os.Readlink(fmt.Sprintf("/proc/%d/exe", pid))
			if err != nil || link != execPath {
				continue
			}
		}

		// Verify comm name match
		if nameFilter != "" {
			commBytes, err := os.ReadFile(fmt.Sprintf("/proc/%d/comm", pid))
			if err != nil || strings.TrimSpace(string(commBytes)) != nameFilter {
				continue
			}
		}

		// Verify user filter match
		if userFilter != "" {
			stat, err := os.Stat(fmt.Sprintf("/proc/%d", pid))
			if err != nil {
				continue
			}
			sysInfo, ok := stat.Sys().(*syscall.Stat_t)
			if !ok {
				continue
			}
			uidStr := strconv.Itoa(int(sysInfo.Uid))
			if userFilter != uidStr {
				// Attempt username matching if userFilter is username, but matching UID is simpler and standard.
				continue
			}
		}

		pids = append(pids, pid)
	}

	return pids
}
