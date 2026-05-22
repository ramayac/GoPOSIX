package sum

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
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

func TestSum_SysV_Empty(t *testing.T) {
	r := strings.NewReader("")
	csum, blocks := Run(r, true)
	if csum != 0 || blocks != 0 {
		t.Errorf("SysV empty: got %d %d, want 0 0", csum, blocks)
	}
}

func TestSum_BSD_LargerData(t *testing.T) {
	// Data that spans multiple blocks (each block = 1024 bytes for BSD)
	data := strings.Repeat("x", 2048)
	r := strings.NewReader(data)
	csum, blocks := Run(r, false)
	if blocks != 2 {
		t.Errorf("BSD 2048 bytes: expected 2 blocks, got %d", blocks)
	}
	if csum == 0 {
		t.Error("BSD larger data: checksum should not be 0")
	}
}

func TestSum_SysV_LargerData(t *testing.T) {
	// Data that spans multiple blocks (each block = 512 bytes for SysV)
	data := strings.Repeat("x", 1024)
	r := strings.NewReader(data)
	csum, blocks := Run(r, true)
	if blocks != 2 {
		t.Errorf("SysV 1024 bytes: expected 2 blocks, got %d", blocks)
	}
	if csum == 0 {
		t.Error("SysV larger data: checksum should not be 0")
	}
}

func TestSum_BSD_KnownValues(t *testing.T) {
	// Verify BSD checksum matches GNU sum -r reference outputs.
	// These values were verified against: printf '%s' "input" | sum -r
	tests := []struct {
		input    string
		expected int
	}{
		{"a", 97},
		{"hello\n", 36979},
		{"", 0},
		{"1234567890", 59623},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			csum, _ := Run(strings.NewReader(tt.input), false)
			if csum != tt.expected {
				t.Errorf("%q: got %d, want %d", tt.input, csum, tt.expected)
			}
		})
	}
}

// --- CLI layer tests (sumRun) ---

func TestSumRun_BSD_Stdin(t *testing.T) {
	var outBuf, errBuf bytes.Buffer
	in := strings.NewReader("hello\n")
	rc := sumRun([]string{}, &outBuf, &errBuf, in, "")
	if rc != 0 {
		t.Errorf("exit code: got %d, want 0", rc)
	}
	// BSD single-file output: "%05d %5d\n"
	expected := fmt.Sprintf("%05d %5d\n", 36979, 1)
	if outBuf.String() != expected {
		t.Errorf("got %q, want %q", outBuf.String(), expected)
	}
}

func TestSumRun_BSD_SingleFile(t *testing.T) {
	tmp := t.TempDir()
	fpath := filepath.Join(tmp, "testfile")
	if err := os.WriteFile(fpath, []byte("hello\n"), 0644); err != nil {
		t.Fatal(err)
	}

	var outBuf, errBuf bytes.Buffer
	rc := sumRun([]string{fpath}, &outBuf, &errBuf, nil, "")
	if rc != 0 {
		t.Errorf("exit code: got %d, want 0", rc)
	}
	// BSD single file: no filename (unlike SysV)
	expected := fmt.Sprintf("%05d %5d\n", 36979, 1)
	if outBuf.String() != expected {
		t.Errorf("got %q, want %q", outBuf.String(), expected)
	}
}

func TestSumRun_BSD_MultiFile(t *testing.T) {
	tmp := t.TempDir()
	f1 := filepath.Join(tmp, "a")
	f2 := filepath.Join(tmp, "b")
	os.WriteFile(f1, []byte("hello\n"), 0644)
	os.WriteFile(f2, []byte("hello\n"), 0644)

	var outBuf, errBuf bytes.Buffer
	rc := sumRun([]string{f1, f2}, &outBuf, &errBuf, nil, "")
	if rc != 0 {
		t.Errorf("exit code: got %d, want 0", rc)
	}
	lines := strings.Split(strings.TrimSpace(outBuf.String()), "\n")
	if len(lines) != 2 {
		t.Errorf("expected 2 output lines for 2 files, got %d: %q", len(lines), outBuf.String())
	}
	if !strings.Contains(lines[0], f1) {
		t.Errorf("first line missing filename %q", f1)
	}
	if !strings.Contains(lines[1], f2) {
		t.Errorf("second line missing filename %q", f2)
	}
}

func TestSumRun_SysV_Stdin(t *testing.T) {
	var outBuf, errBuf bytes.Buffer
	in := strings.NewReader("hello\n")
	rc := sumRun([]string{"-s"}, &outBuf, &errBuf, in, "")
	if rc != 0 {
		t.Errorf("exit code: got %d, want 0", rc)
	}
	expected := fmt.Sprintf("%d %d\n", 542, 1)
	if outBuf.String() != expected {
		t.Errorf("got %q, want %q", outBuf.String(), expected)
	}
}

func TestSumRun_SysV_SingleFile(t *testing.T) {
	tmp := t.TempDir()
	fpath := filepath.Join(tmp, "testfile")
	if err := os.WriteFile(fpath, []byte("hello\n"), 0644); err != nil {
		t.Fatal(err)
	}

	var outBuf, errBuf bytes.Buffer
	rc := sumRun([]string{"-s", fpath}, &outBuf, &errBuf, nil, "")
	if rc != 0 {
		t.Errorf("exit code: got %d, want 0", rc)
	}
	expected := fmt.Sprintf("%d %d %s\n", 542, 1, fpath)
	if outBuf.String() != expected {
		t.Errorf("got %q, want %q", outBuf.String(), expected)
	}
}

func TestSumRun_Json(t *testing.T) {
	tmp := t.TempDir()
	fpath := filepath.Join(tmp, "testfile")
	if err := os.WriteFile(fpath, []byte("hello\n"), 0644); err != nil {
		t.Fatal(err)
	}

	var outBuf, errBuf bytes.Buffer
	rc := sumRun([]string{"--json", fpath}, &outBuf, &errBuf, nil, "")
	if rc != 0 {
		t.Errorf("exit code: got %d, want 0", rc)
	}
	if !strings.Contains(outBuf.String(), "\"files\"") {
		t.Errorf("JSON missing 'files': %s", outBuf.String())
	}
	if !strings.Contains(outBuf.String(), "\"checksum\"") {
		t.Errorf("JSON missing 'checksum': %s", outBuf.String())
	}
}

func TestSumRun_MissingFile(t *testing.T) {
	var outBuf, errBuf bytes.Buffer
	rc := sumRun([]string{"/nonexistent/file"}, &outBuf, &errBuf, nil, "")
	if rc != 1 {
		t.Errorf("exit code: got %d, want 1 for missing file", rc)
	}
}

func TestSumRun_StdinDash(t *testing.T) {
	var outBuf, errBuf bytes.Buffer
	in := strings.NewReader("hello\n")
	rc := sumRun([]string{"-"}, &outBuf, &errBuf, in, "")
	if rc != 0 {
		t.Errorf("exit code: got %d, want 0", rc)
	}
	if outBuf.Len() == 0 {
		t.Error("expected output for stdin via dash")
	}
}

func TestSumRun_Dispatch(t *testing.T) {
	var outBuf bytes.Buffer
	rc := run([]string{}, strings.NewReader(""), &outBuf, &outBuf, "")
	if rc != 0 {
		t.Errorf("exit code: got %d, want 0", rc)
	}
}

func TestSumRun_BadFlag(t *testing.T) {
	var outBuf, errBuf bytes.Buffer
	rc := sumRun([]string{"--nonexistent"}, &outBuf, &errBuf, strings.NewReader(""), "")
	if rc != 2 {
		t.Errorf("exit code: got %d, want 2 for bad flag", rc)
	}
}
