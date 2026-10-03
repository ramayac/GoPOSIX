//go:build linux

package daemon

import (
	"os"
	"strings"
	"unsafe"
)

// argvArea holds the original argv+environment memory region.
// The Go runtime stores os.Args strings in the original C argv area on Linux,
// and the kernel reports this region as /proc/<pid>/cmdline (what ps aux shows).
var argvArea []byte

func isGoTest() bool {
	for _, a := range os.Args {
		if strings.HasPrefix(a, "-test.") {
			return true
		}
	}
	if len(os.Args) > 0 && strings.HasSuffix(os.Args[0], ".test") {
		return true
	}
	return false
}

func init() {
	if len(os.Args) == 0 || isGoTest() {
		return
	}
	argvArea = scanArgvArea(os.Args, os.Environ())
}

// scanArgvArea computes the contiguous argv+environment memory region.
// It returns nil when the strings are not contiguous (the runtime copied
// argv out of the original C argv area).
func scanArgvArea(args, environ []string) []byte {
	if len(args) == 0 {
		return nil
	}

	// args[0] points to the beginning of the argv area.
	// After argv comes the environment block — the region is contiguous.
	// Keep base as unsafe.Pointer (not uintptr) to satisfy go vet.
	basePtr := unsafe.Pointer(unsafe.StringData(args[0]))
	base := uintptr(basePtr)
	var endOffset uintptr

	// Walk argv strings.
	for _, a := range args {
		ptr := uintptr(unsafe.Pointer(unsafe.StringData(a)))
		if ptr < base {
			return nil // pointer before base — not contiguous
		}
		offset := ptr - base
		if offset > endOffset+4096 { // gap too large — runtime copied argv
			return nil
		}
		next := offset + uintptr(len(a)) + 1 // +1 for null terminator
		if next > endOffset {
			endOffset = next
		}
	}

	// Walk environment strings (immediately after argv).
	for _, e := range environ {
		ptr := uintptr(unsafe.Pointer(unsafe.StringData(e)))
		if ptr < base {
			break // pointer before base — not contiguous
		}
		offset := ptr - base
		if offset > endOffset+4096 { // gap too large — stop here
			break
		}
		next := offset + uintptr(len(e)) + 1
		if next > endOffset {
			endOffset = next
		}
	}

	if endOffset > 0 {
		return unsafe.Slice((*byte)(basePtr), int(endOffset))
	}
	return nil
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
