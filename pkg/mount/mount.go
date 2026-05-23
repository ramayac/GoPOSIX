// Package mount implements the mount(8) utility for GoPOSIX.
//
// mount attaches a filesystem at a specified mount point. In containers,
// this is typically used for bind mounts, tmpfs, proc, sysfs, etc.
//
// Usage:
//
//	mount [-t type] [-o options] device dir  — mount device at dir
//	mount -a                                  — mount all entries from /etc/fstab
//	mount                                     — list currently mounted filesystems
//
// On Linux, actual mounting uses the unix syscall from golang.org/x/sys/unix.
// doMount is implemented in mount_linux.go / mount_other.go depending on build.
package mount

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
)

// MountEntry describes a single active mount for JSON output.
type MountEntry struct {
	Device  string `json:"device"`
	MntPt   string `json:"mountpoint"`
	FSType  string `json:"fstype"`
	Options string `json:"options"`
}

// MountResult is the JSON envelope.
type MountResult struct {
	Mounts []MountEntry `json:"mounts"`
}

var flagSpec = common.FlagSpec{
	Defs: []common.FlagDef{
		{Short: "t", Long: "types", Type: common.FlagValue},
		{Short: "o", Long: "options", Type: common.FlagValue},
		{Short: "a", Long: "all", Type: common.FlagBool},
		{Short: "r", Long: "read-only", Type: common.FlagBool},
		{Short: "w", Long: "read-write", Type: common.FlagBool},
		{Short: "n", Long: "no-mtab", Type: common.FlagBool},
		{Long: "json", Type: common.FlagBool},
	},
}

func init() {
	dispatch.Register(dispatch.Command{
		Name:  "mount",
		Usage: "Mount a filesystem",
		Run:   run,
	})
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer, cwd string) int {
	return mountRun(args, stdin, stdout, stderr, cwd)
}

// mountRun is the injectable entry point for testing.
func mountRun(args []string, stdin io.Reader, stdout, stderr io.Writer, cwd string) int {
	flags, err := common.ParseFlags(args, flagSpec)
	if err != nil {
		fmt.Fprintf(stderr, "mount: %v\n", err)
		return 1
	}

	pos := flags.Positional

	jsonMode := flags.Has("json")
	fsType := flags.Get("t")
	options := flags.Get("o")
	mountAll := flags.Has("a")
	readOnly := flags.Has("r")

	// Default fstype
	if fsType == "" {
		fsType = "auto"
	}

	// Parse mount options into flags
	if readOnly {
		if options == "" {
			options = "ro"
		} else {
			options += ",ro"
		}
	}

	// List mode (no arguments)
	if len(pos) == 0 && !mountAll {
		return listMounts(jsonMode, stdout, stderr)
	}

	// Mount-all mode
	if mountAll {
		return mountAllFstab(fsType, options, jsonMode, stdout, stderr)
	}

	// Normal mount: need device and mountpoint
	if len(pos) < 2 {
		fmt.Fprintln(stderr, "mount: usage: mount [-t type] [-o options] device dir")
		return 1
	}

	device := pos[0]
	mountPt := pos[1]

	return doMount(device, mountPt, fsType, options, jsonMode, stdout, stderr)
}

// listMounts reads /proc/mounts and prints active mount entries.
func listMounts(jsonMode bool, stdout, stderr io.Writer) int {
	f, err := os.Open("/proc/mounts")
	if err != nil {
		// Fallback to /etc/mtab
		f, err = os.Open("/etc/mtab")
		if err != nil {
			fmt.Fprintf(stderr, "mount: cannot read mount table: %v\n", err)
			return 1
		}
	}
	defer f.Close()

	return parseMountTable(f, jsonMode, stdout)
}

// parseMountTable reads a mounts/mtab file and outputs entries.
func parseMountTable(r io.Reader, jsonMode bool, stdout io.Writer) int {
	var entries []MountEntry
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "#") || strings.TrimSpace(line) == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		e := MountEntry{
			Device:  fields[0],
			MntPt:   fields[1],
			FSType:  fields[2],
			Options: fields[3],
		}
		if jsonMode {
			entries = append(entries, e)
		} else {
			fmt.Fprintf(stdout, "%s on %s type %s (%s)\n",
				e.Device, e.MntPt, e.FSType, e.Options)
		}
	}

	if jsonMode {
		common.Render("mount", MountResult{Mounts: entries}, true, stdout, nil)
	}
	return 0
}

// mountAllFstab mounts all entries from /etc/fstab that have "auto" in their options.
func mountAllFstab(fsType, options string, jsonMode bool, stdout, stderr io.Writer) int {
	f, err := os.Open("/etc/fstab")
	if err != nil {
		fmt.Fprintf(stderr, "mount: cannot open /etc/fstab: %v\n", err)
		return 1
	}
	defer f.Close()

	exitCode := 0
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "#") || strings.TrimSpace(line) == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		device := fields[0]
		mountPt := fields[1]
		ft := fields[2]
		opts := fields[3]

		// Skip entries marked "noauto"
		if strings.Contains(opts, "noauto") {
			continue
		}

		// Override fsType/options if explicitly specified
		if fsType != "auto" {
			ft = fsType
		}
		if options != "" {
			opts = options
		}

		if rc := doMount(device, mountPt, ft, opts, false, stdout, stderr); rc != 0 {
			exitCode = rc
		}
	}
	return exitCode
}

// parseOptions parses a comma-separated options string and converts it to
// mount flags and a filesystem-specific data string.
// The actual flag constants (msRdOnly, etc.) are defined in platform files.
func parseOptions(options string) (mountFlags uintptr, data string) {
	var dataOpts []string
	for _, opt := range strings.Split(options, ",") {
		opt = strings.TrimSpace(opt)
		switch opt {
		case "ro":
			mountFlags |= msRdOnly
		case "rw":
			// read-write is the default, no flag needed
		case "noexec":
			mountFlags |= msNoExec
		case "nosuid":
			mountFlags |= msNoSUID
		case "nodev":
			mountFlags |= msNoDev
		case "sync":
			mountFlags |= msSync
		case "bind":
			mountFlags |= msBind
		case "remount":
			mountFlags |= msRemount
		case "rbind":
			mountFlags |= msBind | msRec
		case "auto", "defaults", "noauto", "user", "users", "exec", "dev", "suid", "0":
			// known opts with no direct flag equivalent — ignore
		default:
			if opt != "" {
				dataOpts = append(dataOpts, opt)
			}
		}
	}
	data = strings.Join(dataOpts, ",")
	return mountFlags, data
}
