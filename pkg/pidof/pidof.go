// Package pidof implements the POSIX-aligned pidof utility.
package pidof

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
)

// PidofResult is the --json output.
type PidofResult struct {
	Pids []int `json:"pids"`
}

var spec = common.FlagSpec{
	Defs: []common.FlagDef{
		{Short: "s", Type: common.FlagBool},
		{Short: "o", Type: common.FlagValue}, // omit PID
		{Long: "json", Type: common.FlagBool},
	},
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer, cwd string) int {
	flags, err := common.ParseFlags(args, spec)
	if err != nil {
		fmt.Fprintf(stderr, "pidof: %v\n", err)
		return 1
	}

	if len(flags.Positional) == 0 {
		fmt.Fprintf(stderr, "pidof: missing operand\n")
		return 1
	}

	jsonMode := flags.Has("json")
	single := flags.Has("s")

	// Parse omit PIDs
	omitPIDs := make(map[int]bool)
	for _, o := range flags.GetAll("o") {
		if o == "%PPID" {
			omitPIDs[os.Getppid()] = true
		} else {
			if pidVal, err := strconv.Atoi(o); err == nil {
				omitPIDs[pidVal] = true
			}
		}
	}

	targetNames := flags.Positional

	// Scan /proc
	procDirs, err := os.ReadDir("/proc")
	if err != nil {
		fmt.Fprintf(stderr, "pidof: failed to read /proc: %v\n", err)
		return 1
	}

	var matchedPIDs []int

	for _, dir := range procDirs {
		if !dir.IsDir() {
			continue
		}
		pidVal, err := strconv.Atoi(dir.Name())
		if err != nil {
			// Not a PID directory
			continue
		}

		if omitPIDs[pidVal] {
			continue
		}

		// Read /proc/<pid>/comm
		commBytes, err := os.ReadFile(filepath.Join("/proc", dir.Name(), "comm"))
		comm := ""
		if err == nil {
			comm = strings.TrimSpace(string(commBytes))
		}

		// Read /proc/<pid>/exe target
		exeTarget := ""
		if link, err := os.Readlink(filepath.Join("/proc", dir.Name(), "exe")); err == nil {
			exeTarget = filepath.Base(link)
		}

		// Read /proc/<pid>/cmdline arguments
		var cmdlineArgs []string
		if cmdlineBytes, err := os.ReadFile(filepath.Join("/proc", dir.Name(), "cmdline")); err == nil {
			parts := strings.Split(string(cmdlineBytes), "\x00")
			for _, p := range parts {
				if p != "" {
					cmdlineArgs = append(cmdlineArgs, filepath.Base(p))
				}
			}
		}

		// Check matches
		matched := false
		for _, target := range targetNames {
			if comm == target || exeTarget == target {
				matched = true
				break
			}
			for _, arg := range cmdlineArgs {
				if arg == target {
					matched = true
					break
				}
			}
			if matched {
				break
			}
		}

		if matched {
			matchedPIDs = append(matchedPIDs, pidVal)
		}
	}

	if len(matchedPIDs) == 0 {
		return 1
	}

	// Sort PIDs in descending order
	sort.Slice(matchedPIDs, func(i, j int) bool {
		return matchedPIDs[i] > matchedPIDs[j]
	})

	if single {
		matchedPIDs = matchedPIDs[:1]
	}

	common.Render("pidof", PidofResult{Pids: matchedPIDs}, jsonMode, stdout, func() {
		var strPids []string
		for _, p := range matchedPIDs {
			strPids = append(strPids, strconv.Itoa(p))
		}
		fmt.Fprintln(stdout, strings.Join(strPids, " "))
	})

	return 0
}

func init() {
	dispatch.Register(dispatch.Command{
		Name:  "pidof",
		Usage: "Find the process ID of a running program",
		Run:   run,
	})
}
