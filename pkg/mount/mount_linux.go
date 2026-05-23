//go:build linux

package mount

import (
	"fmt"
	"io"
	"strings"

	"golang.org/x/sys/unix"
)

// Linux-specific mount flag constants from the kernel ABI.
const (
	msRdOnly  = uintptr(unix.MS_RDONLY)
	msNoExec  = uintptr(unix.MS_NOEXEC)
	msNoSUID  = uintptr(unix.MS_NOSUID)
	msNoDev   = uintptr(unix.MS_NODEV)
	msSync    = uintptr(unix.MS_SYNCHRONOUS)
	msBind    = uintptr(unix.MS_BIND)
	msRemount = uintptr(unix.MS_REMOUNT)
	msRec     = uintptr(unix.MS_REC)
)

// doMount performs the actual mount(2) syscall on Linux.
func doMount(device, mountPt, fsType, options string, jsonMode bool, stdout, stderr io.Writer) int {
	mountFlags, data := parseOptions(options)

	if err := unix.Mount(device, mountPt, fsType, mountFlags, data); err != nil {
		fmt.Fprintf(stderr, "mount: %s on %s: %v\n", device, mountPt, err)
		return 1
	}

	if !jsonMode {
		fmt.Fprintf(stdout, "Mounted %s on %s type %s (%s)\n", device, mountPt, fsType, options)
	} else {
		// Emit a single-entry JSON result
		parseMountTable(
			strings.NewReader(fmt.Sprintf("%s %s %s %s 0 0\n", device, mountPt, fsType, options)),
			true,
			stdout,
		)
	}
	return 0
}
