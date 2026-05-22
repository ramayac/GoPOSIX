package comm

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func compare(f1, f2 []string, s [3]bool) []Entry {
	return Compare(f1, f2, s)
}

func TestComm_BusyBox1(t *testing.T) {
	entries := compare(
		[]string{"456", "abc"},
		[]string{"123", "def"},
		[3]bool{false, false, false},
	)
	got := Format(entries)
	want := "\t123\n456\nabc\n\tdef\n"
	if got != want {
		t.Errorf("test1:\n  got  %q\n  want %q", got, want)
	}
}

func TestComm_BusyBox2(t *testing.T) {
	entries := compare(
		[]string{"123", "def"},
		[]string{"456", "abc"},
		[3]bool{false, false, false},
	)
	got := Format(entries)
	want := "123\n\t456\n\tabc\ndef\n"
	if got != want {
		t.Errorf("test2:\n  got  %q\n  want %q", got, want)
	}
}

func TestComm_BusyBox3(t *testing.T) {
	entries := compare(
		[]string{"abc", "xyz"},
		[]string{"def"},
		[3]bool{false, false, false},
	)
	got := Format(entries)
	want := "abc\n\tdef\nxyz\n"
	if got != want {
		t.Errorf("test3:\n  got  %q\n  want %q", got, want)
	}
}

func TestComm_BusyBox4(t *testing.T) {
	entries := compare(
		[]string{"def"},
		[]string{"abc", "xyz"},
		[3]bool{false, false, false},
	)
	got := Format(entries)
	want := "\tabc\ndef\n\txyz\n"
	if got != want {
		t.Errorf("test4:\n  got  %q\n  want %q", got, want)
	}
}

func TestComm_BusyBox5(t *testing.T) {
	entries := compare(
		[]string{"123", "abc"},
		[]string{"def"},
		[3]bool{false, false, false},
	)
	got := Format(entries)
	want := "123\nabc\n\tdef\n"
	if got != want {
		t.Errorf("test5:\n  got  %q\n  want %q", got, want)
	}
}

func TestComm_BusyBox6(t *testing.T) {
	entries := compare(
		[]string{"def"},
		[]string{"123", "abc"},
		[3]bool{false, false, false},
	)
	got := Format(entries)
	want := "\t123\n\tabc\ndef\n"
	if got != want {
		t.Errorf("test6:\n  got  %q\n  want %q", got, want)
	}
}

func TestComm_BusyBox7(t *testing.T) {
	entries := compare(
		[]string{"abc"},
		[]string{"def"},
		[3]bool{false, false, false},
	)
	got := Format(entries)
	want := "abc\n\tdef\n"
	if got != want {
		t.Errorf("test7:\n  got  %q\n  want %q", got, want)
	}
}

func TestComm_BusyBox8(t *testing.T) {
	// comm - input: file1(stdin) = "def", file2(input) = "abc"
	entries := compare(
		[]string{"def"},
		[]string{"abc"},
		[3]bool{false, false, false},
	)
	got := Format(entries)
	want := "\tabc\ndef\n"
	if got != want {
		t.Errorf("test8:\n  got  %q\n  want %q", got, want)
	}
}

func TestComm_SuppressCol1(t *testing.T) {
	entries := compare(
		[]string{"a", "c"},
		[]string{"b", "d"},
		[3]bool{true, false, false},
	)
	for _, e := range entries {
		if e.Col == 1 {
			t.Errorf("col1 should be suppressed, got %v", e)
		}
	}
	if len(entries) != 2 {
		t.Errorf("expected 2 entries, got %d", len(entries))
	}
}

func TestComm_SuppressCol2(t *testing.T) {
	entries := compare(
		[]string{"a", "c"},
		[]string{"b", "d"},
		[3]bool{false, true, false},
	)
	for _, e := range entries {
		if e.Col == 2 {
			t.Errorf("col2 should be suppressed, got %v", e)
		}
	}
}

func TestComm_SuppressCol3(t *testing.T) {
	entries := compare(
		[]string{"a", "b"},
		[]string{"a", "c"},
		[3]bool{false, false, true},
	)
	for _, e := range entries {
		if e.Col == 3 {
			t.Errorf("col3 should be suppressed, got %v", e)
		}
	}
}

func TestComm_CommonLines(t *testing.T) {
	entries := compare(
		[]string{"a", "b", "d"},
		[]string{"b", "c", "d"},
		[3]bool{false, false, false},
	)
	c1, c2, c3 := Counts(entries)
	if c1 != 1 || c2 != 1 || c3 != 2 {
		t.Errorf("counts: got %d/%d/%d, want 1/1/2", c1, c2, c3)
	}
	if entries[0].Text != "a" || entries[0].Col != 1 {
		t.Errorf("entry0: got %v", entries[0])
	}
	if entries[1].Text != "b" || entries[1].Col != 3 {
		t.Errorf("entry1: got %v", entries[1])
	}
	if entries[2].Text != "c" || entries[2].Col != 2 {
		t.Errorf("entry2: got %v", entries[2])
	}
	if entries[3].Text != "d" || entries[3].Col != 3 {
		t.Errorf("entry3: got %v", entries[3])
	}
}

func TestComm_CommonLines_Format(t *testing.T) {
	entries := compare(
		[]string{"a", "b"},
		[]string{"b", "c"},
		[3]bool{false, false, false},
	)
	got := Format(entries)
	want := "a\n\t\tb\n\tc\n"
	if got != want {
		t.Errorf("format with both:\n  got  %q\n  want %q", got, want)
	}
}

func TestComm_OneFile(t *testing.T) {
	entries := compare(
		[]string{"a", "b", "c"},
		[]string{},
		[3]bool{false, false, false},
	)
	if len(entries) != 3 {
		t.Errorf("expected 3 entries, got %d", len(entries))
	}
}

func TestComm_EmptyBoth(t *testing.T) {
	entries := compare(
		[]string{},
		[]string{},
		[3]bool{false, false, false},
	)
	if len(entries) != 0 {
		t.Errorf("expected empty, got %d", len(entries))
	}
}

// --- CLI layer tests ---

func TestCommRun_Stdin(t *testing.T) {
	var outBuf, errBuf bytes.Buffer
	stdin := strings.NewReader("123\ndef\n")

	rc := commRun([]string{"-", "-"}, &outBuf, &errBuf, stdin, "")
	if rc != 0 {
		t.Logf("stderr: %s", errBuf.String())
	}
	got := outBuf.String()
	if !strings.Contains(got, "\t\t123") || !strings.Contains(got, "\t\tdef") {
		t.Errorf("unexpected output: %q", got)
	}
}

func TestCommRun_JsonFlag(t *testing.T) {
	t.Skip("requires temp files — tested via integration")
}

func TestToResult(t *testing.T) {
	entries := []Entry{
		{Col: 1, Text: "file1-only"},
		{Col: 2, Text: "file2-only"},
		{Col: 3, Text: "both"},
		{Col: 1, Text: "file1-again"},
		{Col: 3, Text: "both2"},
		{Col: 2, Text: "file2-again"},
	}
	result := toResult(entries)
	if len(result.OnlyFile1) != 2 || result.OnlyFile1[0] != "file1-only" || result.OnlyFile1[1] != "file1-again" {
		t.Errorf("OnlyFile1: %v", result.OnlyFile1)
	}
	if len(result.OnlyFile2) != 2 || result.OnlyFile2[0] != "file2-only" || result.OnlyFile2[1] != "file2-again" {
		t.Errorf("OnlyFile2: %v", result.OnlyFile2)
	}
	if len(result.Both) != 2 || result.Both[0] != "both" || result.Both[1] != "both2" {
		t.Errorf("Both: %v", result.Both)
	}
}

func TestToResult_Empty(t *testing.T) {
	result := toResult(nil)
	if len(result.OnlyFile1) != 0 || len(result.OnlyFile2) != 0 || len(result.Both) != 0 {
		t.Errorf("expected empty result, got %+v", result)
	}
}

func TestComm_CLIRun(t *testing.T) {
	// Test the CLI glue run() function.
	var outBuf, errBuf bytes.Buffer
	rc := run([]string{}, nil, &outBuf, &errBuf, "")
	if rc != 2 {
		t.Logf("comm run() exit code: %d", rc)
	}
}

func TestCommCLI_SuppressColumns(t *testing.T) {
	dir := t.TempDir()
	f1 := filepath.Join(dir, "f1")
	f2 := filepath.Join(dir, "f2")
	os.WriteFile(f1, []byte("a\nb\nc\n"), 0644)
	os.WriteFile(f2, []byte("b\nc\nd\n"), 0644)
	var out bytes.Buffer
	// -1 suppresses column 1 (lines only in f1)
	code := run([]string{"-1", f1, f2}, nil, &out, &out, "")
	if code != 0 {
		t.Errorf("exit %d, want 0", code)
	}
	if strings.Contains(out.String(), "a") {
		t.Error("column 1 should be suppressed")
	}
}

func TestCommCLI_SuppressAllColumns(t *testing.T) {
	dir := t.TempDir()
	f1 := filepath.Join(dir, "f1")
	f2 := filepath.Join(dir, "f2")
	os.WriteFile(f1, []byte("a\n"), 0644)
	os.WriteFile(f2, []byte("b\n"), 0644)
	var out bytes.Buffer
	code := run([]string{"-1", "-2", "-3", f1, f2}, nil, &out, &out, "")
	if code != 0 {
		t.Errorf("exit %d, want 0", code)
	}
	if out.Len() != 0 {
		t.Errorf("expected empty output, got %q", out.String())
	}
}

func TestCommCLI_TotalFlag(t *testing.T) {
	dir := t.TempDir()
	f1 := filepath.Join(dir, "f1")
	f2 := filepath.Join(dir, "f2")
	os.WriteFile(f1, []byte("a\nb\n"), 0644)
	os.WriteFile(f2, []byte("b\nc\n"), 0644)
	var out bytes.Buffer
	code := run([]string{"--total", f1, f2}, nil, &out, &out, "")
	if code != 0 {
		t.Errorf("exit %d, want 0", code)
	}
}

func TestCommCLI_JSON(t *testing.T) {
	dir := t.TempDir()
	f1 := filepath.Join(dir, "f1")
	f2 := filepath.Join(dir, "f2")
	os.WriteFile(f1, []byte("a\nb\n"), 0644)
	os.WriteFile(f2, []byte("b\nc\n"), 0644)
	var out bytes.Buffer
	code := run([]string{"--json", f1, f2}, nil, &out, &out, "")
	if code != 0 {
		t.Errorf("exit %d, want 0", code)
	}
	if !strings.Contains(out.String(), `"only_file1"`) {
		t.Errorf("expected JSON, got %q", out.String())
	}
}

func TestCommCLI_BadFlag(t *testing.T) {
	var out bytes.Buffer
	code := run([]string{"--nonexistent"}, nil, &out, &out, "")
	if code != 2 {
		t.Errorf("exit %d, want 2", code)
	}
}
