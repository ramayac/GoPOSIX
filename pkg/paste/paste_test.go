package paste

import (
	"bufio"
	"strings"
	"testing"
)

func scannersFromStrings(inputs ...string) []*bufio.Scanner {
	var scs []*bufio.Scanner
	for _, s := range inputs {
		scs = append(scs, bufio.NewScanner(strings.NewReader(s)))
	}
	return scs
}

func TestPaste_Basic(t *testing.T) {
	scs := scannersFromStrings("foo1\nfoo2\nfoo3\n", "bar1\nbar2\nbar3\n")
	records := Merge(scs, false)
	got := Format(records, []string{"\t"})
	want := "foo1\tbar1\nfoo2\tbar2\nfoo3\tbar3\n"
	if got != want {
		t.Errorf("basic:\n  got  %q\n  want %q", got, want)
	}
}

func TestPaste_MultiStdin(t *testing.T) {
	data := "line1\nline2\nline3\nline4\nline5\nline6\n"
	scs := []*bufio.Scanner{
		bufio.NewScanner(strings.NewReader(data)),
		bufio.NewScanner(strings.NewReader(data)),
		bufio.NewScanner(strings.NewReader(data)),
	}
	records := Merge(scs, false)
	got := Format(records, []string{"\t"})
	want := "line1\tline1\tline1\nline2\tline2\tline2\nline3\tline3\tline3\nline4\tline4\tline4\nline5\tline5\tline5\nline6\tline6\tline6\n"
	if got != want {
		t.Errorf("multi-stdin:\n  got  %q\n  want %q", got, want)
	}
}

func TestPaste_Serial(t *testing.T) {
	scs := scannersFromStrings("foo1\nfoo2\nfoo3\n", "bar1\nbar2\nbar3\n")
	records := Merge(scs, true)
	got := Format(records, []string{"\t"})
	want := "foo1\tfoo2\tfoo3\nbar1\tbar2\tbar3\n"
	if got != want {
		t.Errorf("serial:\n  got  %q\n  want %q", got, want)
	}
}

func TestPaste_Serial_AlternatingDelims(t *testing.T) {
	scs := scannersFromStrings("foo1\nbar1\nfoo2\nbar2\nfoo3\n")
	records := Merge(scs, true)
	got := Format(records, []string{"\t", "\n"})
	want := "foo1\tbar1\nfoo2\tbar2\nfoo3\n"
	if got != want {
		t.Errorf("serial alt delims:\n  got  %q\n  want %q", got, want)
	}
}

func TestPaste_NulDelimiter(t *testing.T) {
	// -d '\0' means empty delimiter (no separator)
	scs := scannersFromStrings("this is the ", "first line\n")
	records := Merge(scs, false)
	got := Format(records, []string{""}) // empty delimiter = join directly
	want := "this is the first line\n"
	if got != want {
		t.Errorf("NUL as empty:\n  got  %q\n  want %q", got, want)
	}
}

func TestPaste_CustomDelimiter(t *testing.T) {
	scs := scannersFromStrings("a\nb\n", "1\n2\n")
	records := Merge(scs, false)
	got := Format(records, []string{":"})
	want := "a:1\nb:2\n"
	if got != want {
		t.Errorf("custom delim:\n  got  %q\n  want %q", got, want)
	}
}

func TestParseDelimiters(t *testing.T) {
	tests := []struct {
		in   string
		want []string
	}{
		{"\\t", []string{"\t"}},
		{"\\n", []string{"\n"}},
		{"\\0", []string{""}},
		{"\\\\", []string{"\\"}},
		{":", []string{":"}},
		{"\\t\\n", []string{"\t", "\n"}},
		{"abc", []string{"a", "b", "c"}},
		{"\\t\\0\\n", []string{"\t", "", "\n"}},
	}
	for _, tc := range tests {
		got := parseDelimiters(tc.in)
		if len(got) != len(tc.want) {
			t.Errorf("parseDelimiters(%q) len: got %d, want %d", tc.in, len(got), len(tc.want))
			continue
		}
		for i := range got {
			if got[i] != tc.want[i] {
				t.Errorf("parseDelimiters(%q)[%d]: got %q, want %q", tc.in, i, got[i], tc.want[i])
			}
		}
	}
}

func TestPaste_UnequalLengths(t *testing.T) {
	scs := scannersFromStrings("a\nb\nc\n", "1\n2\n")
	records := Merge(scs, false)
	got := Format(records, []string{"\t"})
	want := "a\t1\nb\t2\nc\n"
	if got != want {
		t.Errorf("unequal:\n  got  %q\n  want %q", got, want)
	}
}

func TestPaste_Empty(t *testing.T) {
	records := Merge(nil, false)
	if len(records) != 0 {
		t.Errorf("empty: got %d records", len(records))
	}
}

func TestPaste_SingleFile(t *testing.T) {
	scs := scannersFromStrings("a\nb\nc\n")
	records := Merge(scs, false)
	got := Format(records, []string{"\t"})
	want := "a\nb\nc\n"
	if got != want {
		t.Errorf("single:\n  got  %q\n  want %q", got, want)
	}
}
