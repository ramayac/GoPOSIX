//go:build linux

package daemon

import (
	"bytes"
	"testing"
	"unsafe"
)

// place returns a string whose bytes live inside buf at the given offset.
// It simulates strings stored in the original contiguous argv+env area.
func place(buf []byte, off int, s string) string {
	copy(buf[off:], s)
	return unsafe.String(&buf[off], len(s))
}

func TestScanArgvAreaContiguous(t *testing.T) {
	buf := make([]byte, 1024)
	args := []string{place(buf, 0, "prog"), place(buf, 16, "arg1")}
	env := []string{place(buf, 48, "A=B"), place(buf, 64, "C=D")}

	area := scanArgvArea(args, env)
	if area == nil {
		t.Fatal("expected contiguous area, got nil")
	}
	want := 64 + len("C=D") + 1
	if len(area) != want {
		t.Errorf("area length = %d, want %d", len(area), want)
	}
}

func TestScanArgvAreaNoArgs(t *testing.T) {
	if area := scanArgvArea(nil, nil); area != nil {
		t.Errorf("expected nil for empty args, got %d bytes", len(area))
	}
}

func TestScanArgvAreaPointerBeforeBase(t *testing.T) {
	buf := make([]byte, 256)
	// args[1] sits before args[0] — not contiguous, must return nil.
	args := []string{place(buf, 64, "prog"), place(buf, 0, "early")}

	if area := scanArgvArea(args, nil); area != nil {
		t.Errorf("expected nil, got %d bytes", len(area))
	}
}

func TestScanArgvAreaGapTooLarge(t *testing.T) {
	buf := make([]byte, 8192)
	// Second arg starts 6000 bytes after the first — gap larger than 4096.
	args := []string{place(buf, 0, "prog"), place(buf, 6000, "far")}

	if area := scanArgvArea(args, nil); area != nil {
		t.Errorf("expected nil, got %d bytes", len(area))
	}
}

func TestScanArgvAreaEnvGapTooLarge(t *testing.T) {
	buf := make([]byte, 8192)
	args := []string{place(buf, 0, "prog")}
	env := []string{place(buf, 6000, "X=Y")}

	// The env string is beyond the gap — the walk stops, area covers argv only.
	area := scanArgvArea(args, env)
	if area == nil {
		t.Fatal("expected area, got nil")
	}
	want := len("prog") + 1
	if len(area) != want {
		t.Errorf("area length = %d, want %d", len(area), want)
	}
}

func TestSetProcTitleTruncatesAndZeroFills(t *testing.T) {
	argvArea = []byte("original title")
	defer func() { argvArea = nil }()

	setProcTitle("sh")
	want := append([]byte("sh"), bytes.Repeat([]byte{0}, len("original title")-2)...)
	if !bytes.Equal(argvArea, want) {
		t.Errorf("area = %q, want %q", argvArea, want)
	}
}

func TestSetProcTitleLongerThanArea(t *testing.T) {
	argvArea = []byte("tiny")
	defer func() { argvArea = nil }()

	title := "a much longer title"
	setProcTitle(title)
	if !bytes.Equal(argvArea, []byte(title[:len("tiny")])) {
		t.Errorf("area = %q, want first %d bytes of title", argvArea, len("tiny"))
	}
}

func TestSetProcTitleEmptyArea(t *testing.T) {
	argvArea = nil
	setProcTitle("anything") // must be a no-op, not a panic
}
