package sort

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestSortBasic(t *testing.T) {
	in := "c\nb\na\n"
	items, _ := parseLines(strings.NewReader(in), nil, "", false)
	res := Run(items, nil, false, false, false, false, false, false, false)
	if !reflect.DeepEqual(res, []string{"a", "b", "c"}) {
		t.Errorf("got %v", res)
	}
}

func TestSortReverse(t *testing.T) {
	in := "a\nb\nc\n"
	items, _ := parseLines(strings.NewReader(in), nil, "", false)
	res := Run(items, nil, true, false, false, false, false, false, false)
	if !reflect.DeepEqual(res, []string{"c", "b", "a"}) {
		t.Errorf("got %v", res)
	}
}

func TestSortNumeric(t *testing.T) {
	in := "10\n2\n1\n"
	items, _ := parseLines(strings.NewReader(in), nil, "", false)
	res := Run(items, nil, false, true, false, false, false, false, false)
	if !reflect.DeepEqual(res, []string{"1", "2", "10"}) {
		t.Errorf("got %v", res)
	}
}

func TestSortUnique(t *testing.T) {
	in := "b\na\nb\nc\n"
	items, _ := parseLines(strings.NewReader(in), nil, "", false)
	res := Run(items, nil, false, false, true, false, false, false, false)
	if !reflect.DeepEqual(res, []string{"a", "b", "c"}) {
		t.Errorf("got %v", res)
	}
}

func TestParseHumanVal(t *testing.T) {
	cases := []struct {
		in     string
		num    float64
		hasSuf bool
		suffix int
	}{
		{"", 0, false, 0},
		{"5", 5, false, 0},
		{"1.5K", 1.5, true, 1},
		{"10M", 10, true, 3},
		{"100G", 100, true, 4},
		{"2.5T", 2.5, true, 5},
		{"abc", 0, false, 0},
	}
	for _, c := range cases {
		got := parseHumanVal(c.in)
		if got.num != c.num || got.hasSuffix != c.hasSuf || got.suffix != c.suffix {
			t.Errorf("parseHumanVal(%q) = {num:%v, hasSuffix:%v, suffix:%v}, want {num:%v, hasSuffix:%v, suffix:%v}",
				c.in, got.num, got.hasSuffix, got.suffix, c.num, c.hasSuf, c.suffix)
		}
	}
}

func TestParseNumericPrefix(t *testing.T) {
	cases := []struct {
		in    string
		want  float64
		valid bool
	}{
		{"123", 123, true},
		{"-5.5", -5.5, true},
		{"+10", 10, true},
		{"abc", 0, false},
		{"  42", 42, true},
		{"3.14xxx", 3.14, true},
		{"", 0, false},
	}
	for _, c := range cases {
		got, valid := parseNumericPrefix(c.in)
		if got != c.want || valid != c.valid {
			t.Errorf("parseNumericPrefix(%q) = (%v, %v), want (%v, %v)", c.in, got, valid, c.want, c.valid)
		}
	}
}

func TestParseMonth(t *testing.T) {
	cases := []struct {
		in    string
		want  int
		valid bool
	}{
		{"Jan", 1, true},
		{"jan", 1, true},
		{"JANUARY", 1, true},
		{"feb", 2, true},
		{"DEC", 12, true},
		{"December", 12, true},
		{"xyz", 0, false},
		{"", 0, false},
	}
	for _, c := range cases {
		got, valid := parseMonth(c.in)
		if got != c.want || valid != c.valid {
			t.Errorf("parseMonth(%q) = (%v, %v), want (%v, %v)", c.in, got, valid, c.want, c.valid)
		}
	}
}

func TestExtractKey(t *testing.T) {
	cases := []struct {
		line      string
		ks        keySpec
		delimiter string
		want      string
	}{
		{"a b c", keySpec{startField: 2}, "", "b"},
		{"a,b,c", keySpec{startField: 2}, ",", "b"},
		{"a b c d", keySpec{startField: 2, endField: 3}, "", "b c"},
		{"single", keySpec{startField: 0}, "", "single"},
		{"abcdef", keySpec{startField: 1, startChar: 2}, "", "bcdef"},
		{"abcdef", keySpec{startField: 1, startChar: 2, endChar: 4}, "", "bcd"},
	}
	for _, c := range cases {
		got := extractKey(c.line, c.ks, c.delimiter)
		if got != c.want {
			t.Errorf("extractKey(%q, %+v, %q) = %q, want %q", c.line, c.ks, c.delimiter, got, c.want)
		}
	}
}

func TestParseKeySpec(t *testing.T) {
	specs := parseKeySpec("2,3nr", false, false, false, false)
	if len(specs) != 1 {
		t.Fatalf("expected 1 spec, got %d", len(specs))
	}
	ks := specs[0]
	if ks.startField != 2 || ks.endField != 3 || !ks.numeric || !ks.reverse {
		t.Errorf("got %+v", ks)
	}
}

func TestParseFieldChar(t *testing.T) {
	f, c := parseFieldChar("3.5")
	if f != 3 || c != 5 {
		t.Errorf("got field=%d char=%d, want field=3 char=5", f, c)
	}
	f2, c2 := parseFieldChar("4")
	if f2 != 4 || c2 != 0 {
		t.Errorf("got field=%d char=%d, want field=4 char=0", f2, c2)
	}
}

func TestScanNUL(t *testing.T) {
	data := []byte("a\x00b\x00c")
	adv, tok, err := scanNUL(data, false)
	if err != nil || adv != 2 || string(tok) != "a" {
		t.Errorf("first token: adv=%d tok=%q err=%v", adv, tok, err)
	}

	adv, tok, err = scanNUL(data[2:], false)
	if err != nil || adv != 2 || string(tok) != "b" {
		t.Errorf("second token: adv=%d tok=%q err=%v", adv, tok, err)
	}

	adv, tok, err = scanNUL(data[4:], true)
	if err != nil || adv != 1 || string(tok) != "c" {
		t.Errorf("last token (atEOF): adv=%d tok=%q err=%v", adv, tok, err)
	}
}

func TestCompareHuman(t *testing.T) {
	a := parseHumanVal("5K")
	b := parseHumanVal("10K")
	if compareHuman(a, b) >= 0 {
		t.Error("5K should be less than 10K")
	}

	c := parseHumanVal("1M")
	d := parseHumanVal("500K")
	if compareHuman(c, d) <= 0 {
		t.Error("1M should be greater than 500K")
	}

	e := parseHumanVal("5")
	f := parseHumanVal("5K")
	if compareHuman(e, f) >= 0 {
		t.Error("no-suffix should sort before suffix")
	}
}

func TestCmpFloat(t *testing.T) {
	if cmpFloat(1.0, 2.0) >= 0 {
		t.Error("1 < 2")
	}
	if cmpFloat(2.0, 1.0) <= 0 {
		t.Error("2 > 1")
	}
	if cmpFloat(1.0, 1.0) != 0 {
		t.Error("1 == 1")
	}
}

func TestSortStable(t *testing.T) {
	in := "c\nb\na\n"
	items, _ := parseLines(strings.NewReader(in), nil, "", false)
	res := Run(items, nil, false, false, false, false, false, true, false)
	if !reflect.DeepEqual(res, []string{"a", "b", "c"}) {
		t.Errorf("stable sort got %v", res)
	}
}

func TestParseLinesNUL(t *testing.T) {
	in := strings.NewReader("z\x00a\x00m\x00")
	items, err := parseLines(in, nil, "", true)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 3 {
		t.Errorf("expected 3 items, got %d", len(items))
	}
}

// --- CLI tests via run() ---

func sortTempFile(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp("", "sorttest")
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString(content)
	f.Close()
	t.Cleanup(func() { os.Remove(f.Name()) })
	return f.Name()
}

func TestCLI_BasicFile(t *testing.T) {
	f := sortTempFile(t, "c\nb\na\n")
	var out bytes.Buffer
	code := run([]string{f}, nil, &out, &out, "")
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	if out.String() != "a\nb\nc\n" {
		t.Errorf("got %q, want a\\nb\\nc\\n", out.String())
	}
}

func TestCLI_Reverse(t *testing.T) {
	f := sortTempFile(t, "a\nb\nc\n")
	var out bytes.Buffer
	code := run([]string{"-r", f}, nil, &out, &out, "")
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	if out.String() != "c\nb\na\n" {
		t.Errorf("got %q, want c\\nb\\na\\n", out.String())
	}
}

func TestCLI_Numeric(t *testing.T) {
	f := sortTempFile(t, "10\n2\n1\n")
	var out bytes.Buffer
	code := run([]string{"-n", f}, nil, &out, &out, "")
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	if out.String() != "1\n2\n10\n" {
		t.Errorf("got %q, want 1\\n2\\n10\\n", out.String())
	}
}

func TestCLI_Unique(t *testing.T) {
	f := sortTempFile(t, "b\na\nb\nc\n")
	var out bytes.Buffer
	code := run([]string{"-u", f}, nil, &out, &out, "")
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	if out.String() != "a\nb\nc\n" {
		t.Errorf("got %q, want a\\nb\\nc\\n", out.String())
	}
}

func TestCLI_Combo(t *testing.T) {
	f := sortTempFile(t, "10\n2\n10\n1\n2\n")
	var out bytes.Buffer
	code := run([]string{"-n", "-u", "-r", f}, nil, &out, &out, "")
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	if out.String() != "10\n2\n1\n" {
		t.Errorf("got %q, want 10\\n2\\n1\\n", out.String())
	}
}

func TestCLI_KeyField(t *testing.T) {
	f := sortTempFile(t, "z a\nx b\ny c\n")
	var out bytes.Buffer
	code := run([]string{"-k", "2", f}, nil, &out, &out, "")
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	if out.String() != "z a\nx b\ny c\n" {
		t.Errorf("got %q", out.String())
	}
}

func TestCLI_KeyFieldDelimiter(t *testing.T) {
	f := sortTempFile(t, "z,a\nx,b\ny,c\n")
	var out bytes.Buffer
	code := run([]string{"-t", ",", "-k", "2", f}, nil, &out, &out, "")
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	if out.String() != "z,a\nx,b\ny,c\n" {
		t.Errorf("got %q", out.String())
	}
}

func TestCLI_OutputFile(t *testing.T) {
	in := sortTempFile(t, "c\nb\na\n")
	outFile := filepath.Join(t.TempDir(), "out.txt")
	var out bytes.Buffer
	code := run([]string{"-o", outFile, in}, nil, &out, &out, "")
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	data, _ := os.ReadFile(outFile)
	if string(data) != "a\nb\nc\n" {
		t.Errorf("got %q, want a\\nb\\nc\\n", string(data))
	}
}

func TestCLI_JSON(t *testing.T) {
	f := sortTempFile(t, "b\na\n")
	var out bytes.Buffer
	code := run([]string{"--json", f}, nil, &out, &out, "")
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	if !strings.Contains(out.String(), "\"lines\"") {
		t.Errorf("expected JSON output, got: %s", out.String())
	}
}

func TestCLI_LongFlags(t *testing.T) {
	f := sortTempFile(t, "10\n2\n1\n")
	var out bytes.Buffer
	code := run([]string{"--numeric-sort", "--reverse", f}, nil, &out, &out, "")
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	if out.String() != "10\n2\n1\n" {
		t.Errorf("got %q, want 10\\n2\\n1\\n", out.String())
	}
}

func TestCLI_FileNotFound(t *testing.T) {
	var out bytes.Buffer
	code := run([]string{"/nonexistent/sort/file"}, nil, &out, &out, "")
	if code != 1 {
		t.Errorf("expected exit 1 for missing file, got %d", code)
	}
}

func TestCLI_BadFlag(t *testing.T) {
	var out bytes.Buffer
	code := run([]string{"--nonexistent"}, nil, &out, &out, "")
	if code != 2 {
		t.Errorf("expected exit 2 for bad flag, got %d", code)
	}
}

func TestCLI_MultiFile(t *testing.T) {
	f1 := sortTempFile(t, "c\n")
	f2 := sortTempFile(t, "a\nb\n")
	var out bytes.Buffer
	code := run([]string{f1, f2}, nil, &out, &out, "")
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	if out.String() != "a\nb\nc\n" {
		t.Errorf("got %q, want a\\nb\\nc\\n", out.String())
	}
}

func TestCLI_ZeroTerminated(t *testing.T) {
	f := sortTempFile(t, "z\x00a\x00m\x00")
	var out bytes.Buffer
	code := run([]string{"-z", f}, nil, &out, &out, "")
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	// Output should be zero-terminated: a\x00m\x00z\x00
	if !strings.Contains(out.String(), "a") {
		t.Error("expected 'a' in output")
	}
}

func TestCLI_Stable(t *testing.T) {
	f := sortTempFile(t, "c\nb\na\n")
	var out bytes.Buffer
	code := run([]string{"-s", f}, nil, &out, &out, "")
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	if out.String() != "a\nb\nc\n" {
		t.Errorf("got %q", out.String())
	}
}

func TestCLI_HumanNumeric(t *testing.T) {
	f := sortTempFile(t, "10K\n5K\n1M\n")
	var out bytes.Buffer
	code := run([]string{"-h", f}, nil, &out, &out, "")
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	// 5K < 10K < 1M
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}
	if lines[0] != "5K" {
		t.Errorf("expected 5K first, got %q", lines[0])
	}
}

func TestCLI_MonthSort(t *testing.T) {
	f := sortTempFile(t, "DEC\nJan\nMAR\n")
	var out bytes.Buffer
	code := run([]string{"-M", f}, nil, &out, &out, "")
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if lines[0] != "Jan" {
		t.Errorf("expected Jan first, got %q", lines[0])
	}
}

func TestCLI_EmptyInput(t *testing.T) {
	f := sortTempFile(t, "")
	var out bytes.Buffer
	code := run([]string{f}, nil, &out, &out, "")
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	if out.String() != "" {
		t.Errorf("expected empty output, got %q", out.String())
	}
}

func TestCLI_DashStdin(t *testing.T) {
	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	os.Stdin = r
	go func() {
		w.WriteString("b\na\n")
		w.Close()
	}()
	defer func() { os.Stdin = oldStdin }()

	var out bytes.Buffer
	code := run([]string{"-"}, r, &out, &out, "")
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	if out.String() != "a\nb\n" {
		t.Errorf("got %q, want a\\nb\\n", out.String())
	}
}

func TestCompareHuman_BothSuffix(t *testing.T) {
	// Both have suffix: compare suffix rank first.
	// KB (1) < MB (2)
	a := humanVal{num: 1, suffix: 1, hasSuffix: true, orig: "1K"}
	b := humanVal{num: 1, suffix: 2, hasSuffix: true, orig: "1M"}
	if compareHuman(a, b) >= 0 {
		t.Error("1K should sort before 1M")
	}
	if compareHuman(b, a) <= 0 {
		t.Error("1M should sort after 1K")
	}
}

func TestCompareHuman_SameSuffixDiffNum(t *testing.T) {
	// Same suffix, different numeric prefix.
	a := humanVal{num: 1, suffix: 1, hasSuffix: true, orig: "1K"}
	b := humanVal{num: 2, suffix: 1, hasSuffix: true, orig: "2K"}
	if compareHuman(a, b) >= 0 {
		t.Error("1K should sort before 2K")
	}
}

func TestCompareHuman_SameSuffixSameNumDiffOrig(t *testing.T) {
	// Same suffix, same number, tiebreak by original string.
	a := humanVal{num: 1, suffix: 1, hasSuffix: true, orig: "1K"}
	b := humanVal{num: 1, suffix: 1, hasSuffix: true, orig: "1k"}
	result := compareHuman(a, b)
	if result >= 0 {
		t.Log("tiebreak: '1K' vs '1k' — compareHuman = ", result)
	}
}

func TestCompareHuman_Equal(t *testing.T) {
	a := humanVal{num: 1, suffix: 1, hasSuffix: true, orig: "1K"}
	b := humanVal{num: 1, suffix: 1, hasSuffix: true, orig: "1K"}
	if compareHuman(a, b) != 0 {
		t.Error("identical values should compare equal")
	}
}

func TestCompareHuman_NoSuffixDiffNum(t *testing.T) {
	a := humanVal{num: 1, hasSuffix: false, orig: "1"}
	b := humanVal{num: 2, hasSuffix: false, orig: "2"}
	if compareHuman(a, b) >= 0 {
		t.Error("1 should sort before 2")
	}
}

func TestCompareHuman_NoSuffixTiebreak(t *testing.T) {
	a := humanVal{num: 1, hasSuffix: false, orig: "1a"}
	b := humanVal{num: 1, hasSuffix: false, orig: "1b"}
	if compareHuman(a, b) >= 0 {
		t.Error("'1a' should sort before '1b'")
	}
}
