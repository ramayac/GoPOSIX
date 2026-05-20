package od

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOd_Default(t *testing.T) {
	var out bytes.Buffer
	in := strings.NewReader("HELLO")
	code := odRun(nil, in, &out)
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}
	if !strings.Contains(out.String(), "042510") {
		t.Errorf("expected 042510 in output, got: %q", out.String())
	}
	if !strings.Contains(out.String(), "0000005") {
		t.Errorf("expected final offset 0000005, got: %q", out.String())
	}
}

func TestOd_OctalBytes(t *testing.T) {
	var out bytes.Buffer
	in := strings.NewReader("HELLO")
	code := odRun([]string{"-b"}, in, &out)
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}
	expected := "110 105 114 114 117"
	if !strings.Contains(out.String(), expected) {
		t.Errorf("expected %q in output, got: %q", expected, out.String())
	}
}

func TestOd_Char(t *testing.T) {
	var out bytes.Buffer
	in := strings.NewReader("HELLO")
	code := odRun([]string{"-c"}, in, &out)
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}
	if !strings.Contains(out.String(), "  H") {
		t.Errorf("expected char dump with 'H', got: %q", out.String())
	}
}

func TestOd_CharEscapes(t *testing.T) {
	var out bytes.Buffer
	in := strings.NewReader("a\nb")
	code := odRun([]string{"-c"}, in, &out)
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}
	if !strings.Contains(out.String(), "\\n") {
		t.Errorf("expected \\n escape in output, got: %q", out.String())
	}
}

func TestOd_Hex(t *testing.T) {
	var out bytes.Buffer
	in := strings.NewReader("HELLO")
	code := odRun([]string{"-x"}, in, &out)
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}
	expected := "4548"
	if !strings.Contains(out.String(), expected) {
		t.Errorf("expected %q in output, got: %q", expected, out.String())
	}
}

func TestOd_Count(t *testing.T) {
	var out bytes.Buffer
	in := strings.NewReader("HELLO")
	code := odRun([]string{"-N", "3"}, in, &out)
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}
	if !strings.Contains(out.String(), "0000003") {
		t.Errorf("expected final offset 0000003, got: %q", out.String())
	}
}

func TestOd_FromStdin(t *testing.T) {
	var out bytes.Buffer
	in := strings.NewReader("test")
	code := odRun(nil, in, &out)
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}
	if out.Len() == 0 {
		t.Error("expected output, got empty")
	}
}

func TestOd_Json(t *testing.T) {
	var out bytes.Buffer
	in := strings.NewReader("AB")
	code := odRun([]string{"--json"}, in, &out)
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}
	if !strings.Contains(out.String(), "\"records\"") {
		t.Errorf("expected JSON output with 'records', got: %q", out.String())
	}
}

// --- charEscape tests ---

func TestCharEscape_Printable(t *testing.T) {
	tests := []struct {
		b    byte
		want string
	}{
		{'A', "  A"},
		{'z', "  z"},
		{'0', "  0"},
		{' ', "   "},
		{'~', "  ~"},
	}
	for _, tt := range tests {
		got := charEscape(tt.b)
		if got != tt.want {
			t.Errorf("charEscape(%q) = %q, want %q", tt.b, got, tt.want)
		}
	}
}

func TestCharEscape_Specials(t *testing.T) {
	tests := []struct {
		b    byte
		want string
	}{
		{'\n', " \\n"},
		{'\t', " \\t"},
		{'\r', " \\r"},
		{'\f', " \\f"},
		{'\b', " \\b"},
		{'\a', " \\a"},
		{'\v', " \\v"},
		{'\\', " \\\\"},
		{0, " \\0"},
	}
	for _, tt := range tests {
		got := charEscape(tt.b)
		if got != tt.want {
			t.Errorf("charEscape(%d) = %q, want %q", tt.b, got, tt.want)
		}
	}
}

func TestCharEscape_NonPrintable(t *testing.T) {
	// Bytes < 0x20 that aren't special escapes
	got := charEscape(0x01)
	if got != " 001" {
		t.Errorf("charEscape(0x01) = %q, want %q", got, " 001")
	}
	// Bytes >= 0x7F
	got = charEscape(0x80)
	if got != " 200" {
		t.Errorf("charEscape(0x80) = %q, want %q", got, " 200")
	}
}

// --- float32fromBytes tests ---

func TestFloat32fromBytes(t *testing.T) {
	// IEEE 754: 0x3F800000 = 1.0f
	b := []byte{0x00, 0x00, 0x80, 0x3F}
	f := float32fromBytes(b)
	if f != 1.0 {
		t.Errorf("float32fromBytes: got %f, want 1.0", f)
	}
}

func TestFloat32fromBytes_Negative(t *testing.T) {
	// IEEE 754: 0xBF800000 = -1.0f
	b := []byte{0x00, 0x00, 0x80, 0xBF}
	f := float32fromBytes(b)
	if f != -1.0 {
		t.Errorf("float32fromBytes: got %f, want -1.0", f)
	}
}

func TestFloat32fromBytes_Zero(t *testing.T) {
	b := []byte{0, 0, 0, 0}
	f := float32fromBytes(b)
	if f != 0.0 {
		// -0.0 == 0.0 in Go
		if f != 0.0 {
			t.Errorf("float32fromBytes zero: got %f, want 0.0", f)
		}
	}
}

// --- dumpFloat tests ---

func TestOd_Float(t *testing.T) {
	var out bytes.Buffer
	// 4 bytes: IEEE 754 for 1.0f (little-endian)
	in := strings.NewReader("\x00\x00\x80\x3F")
	code := odRun([]string{"-f"}, in, &out)
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}
	if out.Len() == 0 {
		t.Error("expected float output, got empty")
	}
	// Should contain scientific notation
	if !strings.Contains(out.String(), "e+00") {
		t.Errorf("expected scientific notation in float output: %q", out.String())
	}
}

func TestOd_FloatViaTFlag(t *testing.T) {
	var out bytes.Buffer
	in := strings.NewReader("\x00\x00\x80\x3F")
	code := odRun([]string{"-t", "f"}, in, &out)
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}
	if !strings.Contains(out.String(), "e+00") {
		t.Errorf("expected scientific notation via -t f: %q", out.String())
	}
}

// --- dumpHexBytes tests (-t x1) ---

func TestOd_HexBytes_X1(t *testing.T) {
	var out bytes.Buffer
	in := strings.NewReader("HELLO")
	code := odRun([]string{"-t", "x1"}, in, &out)
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}
	// H=0x48, E=0x45, L=0x4c, L=0x4c, O=0x4f
	expected := "48 45 4c 4c 4f"
	if !strings.Contains(out.String(), expected) {
		t.Errorf("expected %q in output, got: %q", expected, out.String())
	}
}

// --- -t flags ---

func TestOd_TFlag_X2(t *testing.T) {
	var out bytes.Buffer
	in := strings.NewReader("HELLO")
	code := odRun([]string{"-t", "x2"}, in, &out)
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}
	// Same as od -x
	if !strings.Contains(out.String(), "4548") {
		t.Errorf("expected 4548 in -t x2 output: %q", out.String())
	}
}

func TestOd_TFlag_O1(t *testing.T) {
	var out bytes.Buffer
	in := strings.NewReader("HELLO")
	code := odRun([]string{"-t", "o1"}, in, &out)
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}
	// Same as od -b: octal bytes
	expected := "110 105 114 114 117"
	if !strings.Contains(out.String(), expected) {
		t.Errorf("expected %q in -t o1 output: %q", expected, out.String())
	}
}

func TestOd_TFlag_O2(t *testing.T) {
	var out bytes.Buffer
	in := strings.NewReader("HELLO")
	code := odRun([]string{"-t", "o2"}, in, &out)
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}
	// Same as default: 2-byte octal shorts
	if !strings.Contains(out.String(), "042510") {
		t.Errorf("expected 042510 in -t o2 output: %q", out.String())
	}
}

func TestOd_TFlag_C(t *testing.T) {
	var out bytes.Buffer
	in := strings.NewReader("HELLO")
	code := odRun([]string{"-t", "c"}, in, &out)
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}
	if !strings.Contains(out.String(), "  H") {
		t.Errorf("expected char dump via -t c: %q", out.String())
	}
}

// --- File input tests ---

func TestOd_FromFile(t *testing.T) {
	tmp := t.TempDir()
	fpath := filepath.Join(tmp, "od_test")
	if err := os.WriteFile(fpath, []byte("HELLO"), 0644); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	code := Run([]string{fpath}, nil, &out)
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}
	if !strings.Contains(out.String(), "042510") {
		t.Errorf("expected 042510 from file, got: %q", out.String())
	}
}

func TestOd_FileNotFound(t *testing.T) {
	var out bytes.Buffer
	code := Run([]string{"/nonexistent/od_file"}, nil, &out)
	if code != 1 {
		t.Errorf("exit code: got %d, want 1 for missing file", code)
	}
}

// --- dispatch test ---

func TestOd_Dispatch(t *testing.T) {
	var out bytes.Buffer
	code := run([]string{}, nil, &out)
	// run() uses os.Stdin which is /dev/null in tests
	if code != 0 {
		t.Errorf("exit code: got %d, want 0", code)
	}
}

// --- Empty input ---

func TestOd_Empty(t *testing.T) {
	var out bytes.Buffer
	in := strings.NewReader("")
	code := odRun([]string{"-b"}, in, &out)
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}
	// Should just emit final offset 0000000
	if !strings.Contains(out.String(), "0000000") {
		t.Errorf("expected final offset in empty output: %q", out.String())
	}
}

// --- JSON with different formats ---

func TestOd_Json_Hex(t *testing.T) {
	var out bytes.Buffer
	in := strings.NewReader("AB")
	code := odRun([]string{"--json", "-x"}, in, &out)
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}
	if !strings.Contains(out.String(), "\"records\"") {
		t.Errorf("expected JSON with records, got: %q", out.String())
	}
}
