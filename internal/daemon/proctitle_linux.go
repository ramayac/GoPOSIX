//go:build linux

package daemon

import (
	"os"
	"unsafe"
)

// argvArea holds the original argv+environment memory region.
// The Go runtime stores os.Args strings in the original C argv area on Linux,
// and the kernel reports this region as /proc/<pid>/cmdline (what ps aux shows).
var argvArea []byte

func init() {
	if len(os.Args) == 0 {
		return
	}

	// os.Args[0] points to the beginning of the argv area.
	// After argv comes the environment block — the region is contiguous.
	start := uintptr(unsafe.Pointer(unsafe.StringData(os.Args[0])))
	end := start

	// Walk argv strings.
	for _, a := range os.Args {
		ptr := uintptr(unsafe.Pointer(unsafe.StringData(a)))
		if ptr < start || ptr > end+4096 { // not contiguous — runtime copied argv
			return
		}
		next := ptr + uintptr(len(a)) + 1 // +1 for null terminator
		if next > end {
			end = next
		}
	}

	// Walk environment strings (immediately after argv).
	for _, e := range os.Environ() {
		ptr := uintptr(unsafe.Pointer(unsafe.StringData(e)))
		if ptr < start || ptr > end+4096 { // not contiguous — stop here
			break
		}
		next := ptr + uintptr(len(e)) + 1
		if next > end {
			end = next
		}
	}

	if end > start {
		argvArea = unsafe.Slice((*byte)(unsafe.Pointer(start)), int(end-start))
	}
}

// setProcTitle overwrites the process title visible in /proc/<pid>/cmdline
// (shown by ps aux). The title must not exceed the original argv+env length.
func setProcTitle(title string) {
	if len(argvArea) == 0 {
		return
	}
	n := copy(argvArea, title)
	for i := n; i < len(argvArea); i++ {
		argvArea[i] = 0
	}
}
