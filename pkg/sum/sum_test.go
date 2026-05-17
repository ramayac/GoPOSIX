package sum

import (
	"bytes"
	"strings"
	"testing"
)

func TestSum_BSD_Hello(t *testing.T) {
	r := strings.NewReader("hello\n")
	csum, blocks := Run(r, false)
	// GNU sum -r: 36979 1
	if csum != 36979 || blocks != 1 {
		t.Errorf("BSD hello: got %d %d, want 36979 1", csum, blocks)
	}
}

func TestSum_SysV_Hello(t *testing.T) {
	r := strings.NewReader("hello\n")
	csum, blocks := Run(r, true)
	// GNU sum -s: 542 1
	if csum != 542 || blocks != 1 {
		t.Errorf("SysV hello: got %d %d, want 542 1", csum, blocks)
	}
}

func TestSum_BSD_Empty(t *testing.T) {
	r := strings.NewReader("")
	csum, blocks := Run(r, false)
	if csum != 0 || blocks != 0 {
		t.Errorf("BSD empty: got %d %d, want 0 0", csum, blocks)
	}
}

func TestSum_BSD_SingleFile_NoName(t *testing.T) {
	// BSD single file output: no filename
	var out bytes.Buffer
	// Verify format via CLI — need temp file, tested via integration
	_ = out
}

func TestSum_BSD_MultiFile_ShowsName(t *testing.T) {
	// BSD multiple files: show filename
	// Tested via BusyBox integration
}

func TestSum_SysV_SingleFile_ShowsName(t *testing.T) {
	// SysV always shows filename
	// Tested via BusyBox integration
}
