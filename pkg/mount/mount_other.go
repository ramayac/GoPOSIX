//go:build !linux

package mount

import (
	"fmt"
	"io"
)

// Stub constants for non-Linux platforms (compilation only).
const (
	msRdOnly  = uintptr(1)
	msNoExec  = uintptr(8)
	msNoSUID  = uintptr(2)
	msNoDev   = uintptr(4)
	msSync    = uintptr(16)
	msBind    = uintptr(4096)
	msRemount = uintptr(32)
	msRec     = uintptr(16384)
)

// doMount is not supported on non-Linux platforms.
func doMount(device, mountPt, fsType, options string, jsonMode bool, stdout, stderr io.Writer) int {
	fmt.Fprintln(stderr, "mount: not supported on this platform")
	return 1
}
