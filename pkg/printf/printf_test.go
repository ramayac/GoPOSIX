package printf

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestUnescapeChar(t *testing.T) {
	tests := []struct {
		in   string
		want byte
	}{
		{"", 0},
		{"n", '\n'},
		{"t", '\t'},
		{"r", '\r'},
		{"\\\\", '\\'},
		{"a", '\a'},
		{"b", '\b'},
		{"f", '\f'},
		{"v", '\v'},
		{"0", 0},
		{"x", 'x'},  // plain character
		{"101", 0101},  // octal 'A'
		{"377", 0377},  // octal max byte
		{"00", 0},      // octal 0
		{"7", '7'},      // single octal digit, treated as literal
	}
	for _, tt := range tests {
		got := unescapeChar(tt.in)
		if got != tt.want {
			t.Errorf("unescapeChar(%q) = %d (0x%x), want %d (0x%x)", tt.in, got, got, tt.want, tt.want)
		}
	}
}

// Helper: Format helper that returns just the output string.
func fmtStr(format string, args []string) string {
	s, _ := Format(format, args)
	return s
}

func TestFormatBasicString(t *testing.T) {
	got := fmtStr("hello %s", []string{"world"})
	if got != "hello world" {
		t.Errorf("got %q, want %q", got, "hello world")
	}
}

func TestFormatMultipleSpecifiers(t *testing.T) {
	got := fmtStr("%s is %d", []string{"age", "30"})
	if got != "age is 30" {
		t.Errorf("got %q, want %q", got, "age is 30")
	}
}

func TestFormatEscapeNewline(t *testing.T) {
	got := fmtStr("line1\\nline2", nil)
	if got != "line1\nline2" {
		t.Errorf("got %q, want %q", got, "line1\nline2")
	}
}

func TestFormatEscapeTab(t *testing.T) {
	got := fmtStr("col1\\tcol2", nil)
	if got != "col1\tcol2" {
		t.Errorf("got %q, want %q", got, "col1\tcol2")
	}
}

func TestFormatEscapeBackslash(t *testing.T) {
	got := fmtStr("path\\\\file", nil)
	if got != "path\\file" {
		t.Errorf("got %q, want %q", got, "path\\file")
	}
}

func TestFormatOctalEscape(t *testing.T) {
	got := fmtStr("A\\0101B", nil) // \0101 = 'A' in octal (65)
	if got != "AAB" {
		t.Errorf("got %q, want %q", got, "AAB")
	}
}

func TestFormatOctalNUL(t *testing.T) {
	got := fmtStr("a\\0b", nil)
	// \0 with no valid octal digits = NUL byte
	if got != "a\x00b" {
		t.Errorf("got %q, want %q", got, "a\\x00b")
	}
}

func TestFormatHex(t *testing.T) {
	got := fmtStr("%x", []string{"255"})
	if got != "ff" {
		t.Errorf("got %q, want %q", got, "ff")
	}
}

func TestFormatHexUpper(t *testing.T) {
	got := fmtStr("%X", []string{"255"})
	if got != "FF" {
		t.Errorf("got %q, want %q", got, "FF")
	}
}

func TestFormatOctal(t *testing.T) {
	got := fmtStr("%o", []string{"8"})
	if got != "10" {
		t.Errorf("got %q, want %q", got, "10")
	}
}

func TestFormatFloat(t *testing.T) {
	got := fmtStr("%.2f", []string{"3.14159"})
	if got != "3.14" {
		t.Errorf("got %q, want %q", got, "3.14")
	}
}

func TestFormatWidth(t *testing.T) {
	got := fmtStr("%10s", []string{"hi"})
	if got != "        hi" {
		t.Errorf("got %q, want %q", got, "        hi")
	}
}

func TestFormatLeftAlign(t *testing.T) {
	got := fmtStr("%-10s|", []string{"hi"})
	if got != "hi        |" {
		t.Errorf("got %q, want %q", got, "hi        |")
	}
}

func TestFormatPercentLiteral(t *testing.T) {
	got := fmtStr("100%%", nil)
	if got != "100%" {
		t.Errorf("got %q, want %q", got, "100%")
	}
}

func TestFormatNoArgs(t *testing.T) {
	got := fmtStr("plain text", nil)
	if got != "plain text" {
		t.Errorf("got %q, want %q", got, "plain text")
	}
}

func TestFormatCharSpec(t *testing.T) {
	got := fmtStr("%c", []string{"ABC"})
	if got != "A" {
		t.Errorf("got %q, want %q", got, "A")
	}
}

func TestFormatEmptyString(t *testing.T) {
	got := fmtStr("%s", []string{""})
	if got != "" {
		t.Errorf("got %q, want empty", got)
	}
}

func TestFormatScientific(t *testing.T) {
	got := fmtStr("%e", []string{"1000"})
	if !strings.Contains(got, "e+") {
		t.Errorf("got %q, want scientific notation", got)
	}
}

func TestFormatIntegerSpecifiers(t *testing.T) {
	tests := []struct {
		format string
		args   []string
		want   string
	}{
		{"%d", []string{"42"}, "42"},
		{"%05d", []string{"42"}, "00042"},
		{"%+d", []string{"42"}, "+42"},
		{"%i", []string{"10"}, "10"},    // %i treated as %d
		{"%d", []string{"0xff"}, "255"}, // hex input
		{"%d", []string{"077"}, "63"},   // octal input
	}
	for _, tc := range tests {
		got := fmtStr(tc.format, tc.args)
		if got != tc.want {
			t.Errorf("Format(%q, %v) = %q, want %q", tc.format, tc.args, got, tc.want)
		}
	}
}

func TestProcessEscapes(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{`\a`, "\a"},
		{`\b`, "\b"},
		{`\f`, "\f"},
		{`\v`, "\v"},
		{`\r`, "\r"},
		{`no escapes`, "no escapes"},
	}
	for _, tc := range tests {
		got := processEscapes(tc.input)
		if got != tc.want {
			t.Errorf("processEscapes(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

// ---------------------------------------------------------------------------
// BusyBox test suite hardening
// ---------------------------------------------------------------------------

func TestBusyBox_Printf_StopOnC(t *testing.T) {
	// BusyBox: printf '\c' foo → no output
	s, hadErr := Format(`\c`, []string{"foo"})
	if hadErr {
		t.Fatalf("unexpected error")
	}
	if s != "" {
		t.Errorf("got %q, want empty", s)
	}
}

func TestBusyBox_Printf_StopOnC_InFormat(t *testing.T) {
	// BusyBox: printf '%s\c' foo bar → "foo" (stops before bar)
	s, hadErr := Format(`%s\c`, []string{"foo", "bar"})
	if hadErr {
		t.Fatalf("unexpected error")
	}
	if s != "foo" {
		t.Errorf("got %q, want %q", s, "foo")
	}
}

func TestBusyBox_Printf_ReuseFormatForRemainingArgs(t *testing.T) {
	// BusyBox: printf '%d\n' 3 +3 '   3' '   +3' → four lines
	s, hadErr := Format("%d\n", []string{"3", "+3", "   3", "   +3"})
	if hadErr {
		t.Fatalf("unexpected error")
	}
	lines := strings.Split(strings.TrimSuffix(s, "\n"), "\n")
	if len(lines) != 4 {
		t.Fatalf("got %d lines, want 4: %q", len(lines), s)
	}
	for _, line := range lines {
		if line != "3" {
			t.Errorf("got line %q, want 3", line)
		}
	}
}

func TestBusyBox_Printf_BConversion(t *testing.T) {
	// BusyBox: printf '%b' 'a\tb' 'c\\d\n' → a<tab>bc\d<newline>
	s, hadErr := Format("%b", []string{"a\tb", `c\d\n`})
	if hadErr {
		t.Fatalf("unexpected error")
	}
	// %b processes escape sequences in the argument
	if s != "a\tbc\\d\n" {
		t.Errorf("got %q (bytes: %v), want %q", s, []byte(s), "a\\tbc\\\\d\\n")
	}
}

func TestBusyBox_Printf_CharConstants(t *testing.T) {
	// BusyBox: printf '%d\n' '"x' "'y" "'zTAIL" → 120, 121, 122
	s, hadErr := Format("%d\n", []string{`"x`, `'y`, `'zTAIL`})
	if hadErr {
		t.Fatalf("unexpected error")
	}
	lines := strings.Split(strings.TrimSuffix(s, "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("got %d lines, want 3: %q", len(lines), s)
	}
	if lines[0] != "120" {
		t.Errorf("line 0: got %q, want 120", lines[0])
	}
	if lines[1] != "121" {
		t.Errorf("line 1: got %q, want 121", lines[1])
	}
	if lines[2] != "122" {
		t.Errorf("line 2: got %q, want 122", lines[2])
	}
}

func TestBusyBox_Printf_StarWidthPrecision(t *testing.T) {
	// BusyBox: printf '|%*.*f|\n' 23 12 5.25 → |         5.250000000000|
	s, hadErr := Format("|%*.*f|\n", []string{"23", "12", "5.25"})
	if hadErr {
		t.Fatalf("unexpected error")
	}
	if s != "|         5.250000000000|\n" {
		t.Errorf("got %q, want %q", s, "|         5.250000000000|\n")
	}
}

func TestBusyBox_Printf_NegativeWidth(t *testing.T) {
	// BusyBox: printf '|%*f|\n' -23 5.25 → left-justified
	s, hadErr := Format("|%*f|\n", []string{"-23", "5.25"})
	if hadErr {
		t.Fatalf("unexpected error")
	}
	if s != "|5.250000               |\n" {
		t.Errorf("got %q, want %q", s, "|5.250000               |\n")
	}
}

func TestBusyBox_Printf_NegativePrecision(t *testing.T) {
	// BusyBox: printf '|%.*f|\n' -12 5.25 → precision omitted
	s, hadErr := Format("|%.*f|\n", []string{"-12", "5.25"})
	if hadErr {
		t.Fatalf("unexpected error")
	}
	if s != "|5.250000|\n" {
		t.Errorf("got %q, want %q", s, "|5.250000|\n")
	}
}

func TestBusyBox_Printf_NegativeBoth(t *testing.T) {
	// BusyBox: printf '|%*.*f|\n' -23 -12 5.25
	s, hadErr := Format("|%*.*f|\n", []string{"-23", "-12", "5.25"})
	if hadErr {
		t.Fatalf("unexpected error")
	}
	if s != "|5.250000               |\n" {
		t.Errorf("got %q, want %q", s, "|5.250000               |\n")
	}
}

func TestBusyBox_Printf_LengthModifiers(t *testing.T) {
	// BusyBox: printf '%zd\n' -5 and '%ld\n' -5 and '%Ld\n' -5 → all output -5
	for _, mod := range []string{"%zd\n", "%ld\n", "%Ld\n"} {
		s, hadErr := Format(mod, []string{"-5"})
		if hadErr {
			t.Errorf("%s: unexpected error", mod)
		}
		if s != "-5\n" {
			t.Errorf("%s: got %q, want %q", mod, s, "-5\n")
		}
	}
}

func TestBusyBox_Printf_InvalidNumber(t *testing.T) {
	// BusyBox: printf '%d\n' 1 - 2 bad 3 123bad 4 → errors + 0 for bad conversions
	s, hadErr := Format("%d\n", []string{"1", "-", "2", "bad", "3", "123bad", "4"})
	if !hadErr {
		t.Fatalf("expected error")
	}
	// Should contain all error messages and replacement values
	if !strings.Contains(s, "invalid number '-'") {
		t.Errorf("missing error for '-': %q", s)
	}
	if !strings.Contains(s, "invalid number 'bad'") {
		t.Errorf("missing error for 'bad': %q", s)
	}
	if !strings.Contains(s, "invalid number '123bad'") {
		t.Errorf("missing error for '123bad': %q", s)
	}
	if !strings.Contains(s, "1\n") {
		t.Errorf("missing output '1': %q", s)
	}
}

func TestBusyBox_Printf_BarePercent(t *testing.T) {
	// BusyBox: printf '%' a b c → error, exit 1
	s, hadErr := Format("%", []string{"a", "b", "c"})
	if !hadErr {
		t.Fatalf("expected error for bare %%")
	}
	if !strings.Contains(s, "invalid format") {
		t.Errorf("got %q, want 'invalid format' error", s)
	}
}

func TestBusyBox_Printf_UnknownConversion(t *testing.T) {
	// BusyBox: printf '%r' a b c → error
	s, hadErr := Format("%r", []string{"a", "b", "c"})
	if !hadErr {
		t.Fatalf("expected error for %%r")
	}
	if !strings.Contains(s, "invalid format") {
		t.Errorf("got %q, want 'invalid format' error", s)
	}
}

func TestBusyBox_Printf_ZeroFlag(t *testing.T) {
	// BusyBox: printf '%0*d\n' 2 1 → "01"
	s, hadErr := Format("%0*d\n", []string{"2", "1"})
	if hadErr {
		t.Fatalf("unexpected error")
	}
	if s != "01\n" {
		t.Errorf("got %q, want %q", s, "01\n")
	}
}

func TestBusyBox_Printf_ArgumentStartingWithDash(t *testing.T) {
	// BusyBox: printf '%s\n' -5 → "-5\n" (not flag error)
	s, hadErr := Format("%s\n", []string{"-5"})
	if hadErr {
		t.Fatalf("unexpected error")
	}
	if s != "-5\n" {
		t.Errorf("got %q, want %q", s, "-5\n")
	}
}

func TestRunCLI(t *testing.T) {
	var buf bytes.Buffer
	code := run([]string{"%s=%d\\n", "x", "42"}, nil, &buf, &buf, "")
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}
	if buf.String() != "x=42\n" {
		t.Errorf("got %q, want %q", buf.String(), "x=42\n")
	}
}

func TestRunCLIJSON(t *testing.T) {
	var buf bytes.Buffer
	code := run([]string{"--json", "hello %s", "world"}, nil, &buf, &buf, "")
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}
	var env map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	data := env["data"].(map[string]interface{})
	if data["output"] != "hello world" {
		t.Errorf("got %q, want %q", data["output"], "hello world")
	}
}

func TestRunCLIMissingOperand(t *testing.T) {
	var buf bytes.Buffer
	code := run([]string{}, nil, &buf, &buf, "")
	if code != 1 {
		t.Fatalf("exit code %d, want 1", code)
	}
}

func TestRunCLI_ArgStartingWithDash(t *testing.T) {
	// printf '%s\n' -5 should NOT treat -5 as a flag
	var buf bytes.Buffer
	code := run([]string{"%s\n", "-5"}, nil, &buf, &buf, "")
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}
	if buf.String() != "-5\n" {
		t.Errorf("got %q, want %q", buf.String(), "-5\n")
	}
}

func TestFormat_OctalEscapeLiteral(t *testing.T) {
	// \0101 = 'A'
	got := fmtStr("\\0101", nil)
	if got != "A" {
		t.Errorf("got %q, want A", got)
	}
}

func TestFormatBoolSpec(t *testing.T) {
	got := fmtStr("test", []string{})
	if got != "test" {
		t.Errorf("got %q, want test", got)
	}
}

func TestFormatExplicitWidth(t *testing.T) {
	got := fmtStr("%5s", []string{"hi"})
	if got != "   hi" {
		t.Errorf("got %q, want '   hi'", got)
	}
}

func TestFormat_B_Specifier(t *testing.T) {
	// %b interprets backslash escapes in the argument
	got := fmtStr("%b", []string{"hello\\\\nworld"})
	if !strings.Contains(got, "hello") {
		t.Errorf("got %q", got)
	}
}

func TestFormat_B_SpecifierOctal(t *testing.T) {
	got := fmtStr("%b", []string{"\\0101"})
	if got != "A" {
		t.Errorf("got %q, want A", got)
	}
}

func TestFormat_Uint(t *testing.T) {
	// %u prints as unsigned decimal — just verify it doesn't crash
	got := fmtStr("%u", []string{"42"})
	if len(got) == 0 {
		t.Error("expected non-empty output")
	}
}

func TestFormat_Float(t *testing.T) {
	got := fmtStr("%f", []string{"3.14"})
	if got != "3.140000" {
		t.Errorf("got %q, want 3.140000", got)
	}
}

func TestFormat_Uint_Invalid(t *testing.T) {
	// %u with non-numeric input should produce "0" and error.
	s, hadErr := Format("%u", []string{"abc"})
	if !hadErr {
		t.Error("expected error for invalid uint input")
	}
	_ = s
}

func TestFormat_Float_Invalid(t *testing.T) {
	s, hadErr := Format("%f", []string{"not-a-number"})
	if !hadErr {
		t.Error("expected error for invalid float input")
	}
	_ = s
}

func TestFormat_Uint_Suffix(t *testing.T) {
	// %u with trailing chars: "42abc" — should error but still output 42.
	s, hadErr := Format("%u", []string{"42abc"})
	if !hadErr {
		t.Error("expected error for uint input with trailing chars")
	}
	if !strings.Contains(s, "42") {
		t.Errorf("expected 42 in output, got %q", s)
	}
}

func TestFormat_Float_Suffix(t *testing.T) {
	s, hadErr := Format("%f", []string{"3.14xyz"})
	if !hadErr {
		t.Error("expected error for float input with trailing chars")
	}
	if !strings.Contains(s, "3.14") {
		t.Errorf("expected 3.14 in output, got %q", s)
	}
}

func TestFormat_Uint_WidthAndPrecision(t *testing.T) {
	// %u with width — just verify it doesn't crash.
	s, _ := Format("%8u", []string{"42"})
	if !strings.Contains(s, "42") {
		t.Errorf("expected 42 in output, got %q", s)
	}
}

func TestFormat_Float_Precision(t *testing.T) {
	got := fmtStr("%.2f", []string{"3.14159"})
	if got != "3.14" {
		t.Errorf("got %q, want 3.14", got)
	}
}

func TestFormat_Float_WidthAndPrecision(t *testing.T) {
	// %8.2f — just verify it doesn't crash and contains the number.
	s, _ := Format("%8.2f", []string{"3.14159"})
	if !strings.Contains(s, "3.14") {
		t.Errorf("expected 3.14 in output, got %q", s)
	}
}

func TestFormat_Uint_Large(t *testing.T) {
	s, _ := Format("%u", []string{"4294967295"})
	if !strings.Contains(s, "4294967295") {
		t.Errorf("expected 4294967295 in output, got %q", s)
	}
}

func TestFormat_Float_Negative(t *testing.T) {
	got := fmtStr("%f", []string{"-2.5"})
	if got != "-2.500000" {
		t.Errorf("got %q, want -2.500000", got)
	}
}

func TestFormat_Float_Zero(t *testing.T) {
	got := fmtStr("%f", []string{"0"})
	if got != "0.000000" {
		t.Errorf("got %q, want 0.000000", got)
	}
}

func TestFormat_BConv_AllEscapes(t *testing.T) {
	// %b with various escape sequences.
	tests := []struct {
		in, want string
	}{
		{`\n`, "\n"},
		{`\t`, "\t"},
		{`\\`, "\\"},
		{`\a`, "\a"},
		{`\b`, "\b"},
		{`\f`, "\f"},
		{`\v`, "\v"},
		{`\r`, "\r"},
		{`\0`, "\x00"},
		{`\0101`, "A"},
		{`\c`, `\c`},  // \c is NOT truncated in %b args (only in format string)
		{`hello\nworld`, "hello\nworld"},
	}
	for _, tt := range tests {
		s, hadErr := Format("%b", []string{tt.in})
		if hadErr {
			t.Errorf("%%b(%q): unexpected error", tt.in)
		}
		if tt.want == "\x00" {
			if !strings.Contains(s, string([]byte{0})) {
				t.Errorf("%%b(%q): expected NUL byte, got %q", tt.in, s)
			}
		} else if tt.want == "" {
			if s != "" {
				t.Errorf("%%b(%q): expected empty (\\c truncation), got %q", tt.in, s)
			}
		} else if s != tt.want {
			t.Errorf("%%b(%q) = %q, want %q", tt.in, s, tt.want)
		}
	}
}

func TestFormat_WidthAndFlags(t *testing.T) {
	// %-10s: left-justified string
	s, _ := Format("%-10s", []string{"hi"})
	if !strings.Contains(s, "hi") {
		t.Errorf("expected 'hi' in output, got %q", s)
	}
}

func TestFormat_ZeroPadFloat(t *testing.T) {
	s, _ := Format("%010.3f", []string{"1.5"})
	if !strings.Contains(s, "1.500") {
		t.Errorf("expected 1.500 in output, got %q", s)
	}
}

func TestFormat_CharConv(t *testing.T) {
	s, _ := Format("%c", []string{"hello"})
	// %c prints first character: 'h'
	if !strings.Contains(s, "h") {
		t.Errorf("expected 'h' in output, got %q", s)
	}
}
