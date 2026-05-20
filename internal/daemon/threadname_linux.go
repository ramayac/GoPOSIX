//go:build linux

package daemon

import (
	"fmt"
	"runtime"
	"unsafe"

	"golang.org/x/sys/unix"
)

// setWorkerThreadName locks the current goroutine to an OS thread and
// names it "goposix/wrk-NN" so it is visible in htop / top -H.
func setWorkerThreadName(id int) {
	runtime.LockOSThread()
	name := fmt.Sprintf("goposix/wrk-%02d", id)
	// On Linux the thread name is limited to 15 characters. goposix/wrk-00
	// is 15 chars exactly, goposix/wrk-99 is 15 chars, everything fits.
	nameBytes := append([]byte(name), 0)
	_ = unix.Prctl(unix.PR_SET_NAME, uintptr(unsafe.Pointer(&nameBytes[0])), 0, 0, 0)
}
