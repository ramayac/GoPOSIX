package common

import "testing"

func TestExpandEscapes(t *testing.T) {
	tests := []struct {
		name string
		in   string
		mode EscapeMode
		want string
	}{
		{"named", `\n\t\r\\\a\b\f\v`, EscapeFormat, "\n\t\r\\\a\b\f\v"},
		{"backslash-c", `\c`, EscapeFormat, `\c`},
		{"unknown", `\q`, EscapeFormat, `\q`},
		{"format octal", `\0101`, EscapeFormat, "A"},
		{"format bare zero", `\0`, EscapeFormat, "\x00"},
		{"arg octal marker", `\041`, EscapeArg, "!"},
		{"arg bare zero", `\0`, EscapeArg, "\x00"},
		{"arg zero then digits", `\0101`, EscapeArg, "A"},
		{"arg one digit octal", `\1`, EscapeArg, "\x01"},
		{"arg three digit octal", `\123`, EscapeArg, "S"},
		{"arg octal literal eight", `\8`, EscapeArg, `\8`},
		{"hex", `\x41\x42`, EscapeArg, "AB"},
		{"hex short", `\xG`, EscapeArg, `\xG`},
	}
	for _, tc := range tests {
		got := ExpandEscapes(tc.in, tc.mode)
		if got != tc.want {
			t.Errorf("%s: ExpandEscapes(%q) = %q, want %q", tc.name, tc.in, got, tc.want)
		}
	}
}
