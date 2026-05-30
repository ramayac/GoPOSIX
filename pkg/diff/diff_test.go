package diff

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDiffEqual(t *testing.T) {
	differ, hunks := GenerateDiff("a\nb\nc", "a\nb\nc", 3, false, false, false)
	if differ {
		t.Errorf("Expected differ to be false")
	}
	if len(hunks) != 0 {
		t.Errorf("Expected 0 hunks, got %d", len(hunks))
	}
}

func TestDiffSimpleDiff(t *testing.T) {
	differ, hunks := GenerateDiff("a\nb\nc", "a\nx\nc", 3, false, false, false)
	if !differ {
		t.Errorf("Expected differ to be true")
	}
	if len(hunks) != 1 {
		t.Fatalf("Expected 1 hunk, got %d", len(hunks))
	}
	h := hunks[0]
	if h.OldStart != 1 || h.OldLines != 3 || h.NewStart != 1 || h.NewLines != 3 {
		t.Errorf("Hunk numbers wrong: %+v", h)
	}
	expectedLines := []string{" a", "-b", "+x", " c"}
	for i, l := range expectedLines {
		if h.Lines[i] != l {
			t.Errorf("Hunk line %d: got %q, want %q", i, h.Lines[i], l)
		}
	}
}

func TestDiffContextLines(t *testing.T) {
	a := "1\n2\n3\n4\n5\n6\n7\n8\n9"
	b := "1\n2\n3\n4\nx\n6\n7\n8\n9"
	differ, hunks := GenerateDiff(a, b, 2, false, false, false)
	if !differ {
		t.Errorf("Expected differ to be true")
	}
	if len(hunks) != 1 {
		t.Fatalf("Expected 1 hunk, got %d", len(hunks))
	}
	h := hunks[0]
	if h.OldStart != 3 || h.OldLines != 5 || h.NewStart != 3 || h.NewLines != 5 {
		t.Errorf("Hunk numbers wrong: %+v", h)
	}
	expectedLines := []string{" 3", " 4", "-5", "+x", " 6", " 7"}
	for i, l := range expectedLines {
		if h.Lines[i] != l {
			t.Errorf("Hunk line %d: got %q, want %q", i, h.Lines[i], l)
		}
	}
}

func TestRunCLI(t *testing.T) {
	tmpDir := t.TempDir()
	f1 := filepath.Join(tmpDir, "a.txt")
	f2 := filepath.Join(tmpDir, "b.txt")
	os.WriteFile(f1, []byte("a\nb\nc\n"), 0644)
	os.WriteFile(f2, []byte("a\nx\nc\n"), 0644)

	var buf bytes.Buffer
	code := run([]string{"-u", f1, f2}, nil, &buf, &buf, "")
	if code != 1 {
		t.Errorf("exit code %d, want 1", code)
	}
	out := buf.String()
	if !bytes.Contains([]byte(out), []byte("--- "+f1)) || !bytes.Contains([]byte(out), []byte("+++ "+f2)) {
		t.Errorf("missing headers in output: %s", out)
	}
	if !bytes.Contains([]byte(out), []byte("@@ -1,3 +1,3 @@")) {
		t.Errorf("missing hunk header in output: %s", out)
	}
}

func TestRunCLIJSON(t *testing.T) {
	tmpDir := t.TempDir()
	f1 := filepath.Join(tmpDir, "a.txt")
	f2 := filepath.Join(tmpDir, "b.txt")
	os.WriteFile(f1, []byte("a\nb\nc\n"), 0644)
	os.WriteFile(f2, []byte("a\nx\nc\n"), 0644)

	var buf bytes.Buffer
	code := run([]string{"--json", f1, f2}, nil, &buf, &buf, "")
	if code != 1 {
		t.Errorf("exit code %d, want 1", code)
	}

	var env map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	data := env["data"].(map[string]interface{})
	if data["differ"] != true {
		t.Errorf("expected differ true")
	}
}

func TestDiffEmpty(t *testing.T) {
	script := Diff(nil, nil)
	if len(script) != 0 {
		t.Errorf("expected 0 items for empty diff, got %d", len(script))
	}
}

func TestDiffInsert(t *testing.T) {
	script := Diff([]string{"a"}, []string{"a", "b"})
	if len(script) != 2 {
		t.Errorf("expected 2 items, got %d", len(script))
	}
	if script[0].op != opEq || script[1].op != opIns {
		t.Errorf("expected eq+ins, got %v", script)
	}
}

func TestDiffDelete(t *testing.T) {
	script := Diff([]string{"a", "b"}, []string{"a"})
	if len(script) != 2 {
		t.Errorf("expected 2 items, got %d", len(script))
	}
	if script[0].op != opEq || script[1].op != opDel {
		t.Errorf("expected eq+del, got %v", script)
	}
}

func TestDiffAllDifferent(t *testing.T) {
	script := Diff([]string{"a"}, []string{"b"})
	hasDel := false
	hasIns := false
	for _, item := range script {
		if item.op == opDel {
			hasDel = true
		}
		if item.op == opIns {
			hasIns = true
		}
	}
	if !hasDel || !hasIns {
		t.Errorf("expected del+ins for all-different, got %v", script)
	}
}

func TestNormalizeSpace(t *testing.T) {
	lines := []string{"  hello   world  ", "\ttabs\there\t"}
	got := normalizeSpace(lines)
	if got[0] != " hello world" {
		t.Errorf("got %q", got[0])
	}
	if got[1] != " tabs here" {
		t.Errorf("got %q", got[1])
	}
}

func TestFilterBlankLines(t *testing.T) {
	script := []diffItem{
		{op: opDel, text: ""},
		{op: opIns, text: ""},
		{op: opEq, text: "real"},
	}
	got := filterBlankLineChanges(script)
	if got[0].op != opIgnoredDel || got[1].op != opIgnoredIns {
		t.Errorf("blank line changes should be ignored, got %v", got)
	}
}

func TestGenerateDiffIgnoreSpace(t *testing.T) {
	differ, _ := GenerateDiff("hello   world", "hello world", 3, true, false, false)
	if differ {
		t.Error("expected no diff when ignoring whitespace changes")
	}
}

func TestGenerateDiffIgnoreBlankLines(t *testing.T) {
	differ, _ := GenerateDiff("a\n\n\nb", "a\nb", 3, false, false, true)
	if differ {
		t.Error("expected no diff when ignoring blank lines")
	}
}

func TestGenerateDiffEmptyVsContent(t *testing.T) {
	differ, hunks := GenerateDiff("", "hello\n", 3, false, false, false)
	if !differ {
		t.Error("expected differ=true for empty vs content")
	}
	if len(hunks) != 1 {
		t.Errorf("expected 1 hunk, got %d", len(hunks))
	}
}

func TestGenerateDiffContentVsEmpty(t *testing.T) {
	differ, hunks := GenerateDiff("hello\n", "", 3, false, false, false)
	if !differ {
		t.Error("expected differ=true for content vs empty")
	}
	if len(hunks) != 1 {
		t.Errorf("expected 1 hunk, got %d", len(hunks))
	}
}

func TestJoinPreserving(t *testing.T) {
	if got := joinPreserving("dir/", "file"); got != "dir/file" {
		t.Errorf("got %q", got)
	}
	if got := joinPreserving("dir", "file"); got != "dir/file" {
		t.Errorf("got %q", got)
	}
}

func TestMinMax(t *testing.T) {
	// a < b case (covered)
	if min(3, 5) != 3 {
		t.Error("min(3,5) should be 3")
	}
	// b < a case (uncovered)
	if min(5, 3) != 3 {
		t.Error("min(5,3) should be 3")
	}
	// equal case
	if min(3, 3) != 3 {
		t.Error("min(3,3) should be 3")
	}
	// a > b case (covered)
	if max(3, 5) != 5 {
		t.Error("max(3,5) should be 5")
	}
	// b > a case (uncovered)
	if max(5, 3) != 5 {
		t.Error("max(5,3) should be 5")
	}
	// equal case
	if max(3, 3) != 3 {
		t.Error("max(3,3) should be 3")
	}
}

func TestGenerateDiffNoContext(t *testing.T) {
	differ, hunks := GenerateDiff("a\nb\nc", "a\nx\nc", 0, false, false, false)
	if !differ {
		t.Error("expected differ")
	}
	if len(hunks) != 1 {
		t.Errorf("expected 1 hunk, got %d", len(hunks))
	}
}

func TestGenerateDiffMultiHunk(t *testing.T) {
	differ, hunks := GenerateDiff("a\nb\nc\nd\ne", "a\nx\nc\ny\ne", 1, false, false, false)
	if !differ {
		t.Error("expected differ")
	}
	if len(hunks) < 1 {
		t.Errorf("expected at least 1 hunk")
	}
}

func TestCLI_BasicDiff(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.txt")
	b := filepath.Join(dir, "b.txt")
	os.WriteFile(a, []byte("hello\n"), 0644)
	os.WriteFile(b, []byte("world\n"), 0644)
	var out bytes.Buffer
	code := run([]string{a, b}, nil, &out, &out, "")
	if code != 1 {
		t.Errorf("expected exit 1 for diff, got %d", code)
	}
}

func TestCLI_Identical(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.txt")
	b := filepath.Join(dir, "b.txt")
	os.WriteFile(a, []byte("same\n"), 0644)
	os.WriteFile(b, []byte("same\n"), 0644)
	var out bytes.Buffer
	code := run([]string{a, b}, nil, &out, &out, "")
	if code != 0 {
		t.Errorf("expected exit 0 for same files, got %d", code)
	}
}

func TestCLI_ContextLines(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.txt")
	b := filepath.Join(dir, "b.txt")
	os.WriteFile(a, []byte("line1\nline2\nline3\n"), 0644)
	os.WriteFile(b, []byte("line1\nCHANGED\nline3\n"), 0644)
	var out bytes.Buffer
	code := run([]string{"-U", "1", a, b}, nil, &out, &out, "")
	if code != 1 {
		t.Errorf("expected exit 1, got %d", code)
	}
}

func TestCLI_JSON(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.txt")
	os.WriteFile(a, []byte("hello\n"), 0644)
	var out bytes.Buffer
	code := run([]string{"--json", a, a}, nil, &out, &out, "")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if out.Len() == 0 {
		t.Errorf("expected output, got empty")
	}
}

func TestCLI_MissingFile(t *testing.T) {
	var out bytes.Buffer
	code := run([]string{"/nonexistent/a", "/nonexistent/b"}, nil, &out, &out, "")
	if code != 2 {
		t.Errorf("expected exit 2, got %d", code)
	}
}

func TestCLI_BadFlag(t *testing.T) {
	var out bytes.Buffer
	code := run([]string{"--nonexistent"}, nil, &out, &out, "")
	if code != 2 {
		t.Errorf("expected exit 2, got %d", code)
	}
}

func TestDiff_EmptyFileVsNonEmpty(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a")
	b := filepath.Join(dir, "b")
	os.WriteFile(a, []byte(""), 0644)
	os.WriteFile(b, []byte("hello\n"), 0644)
	var out bytes.Buffer
	code := run([]string{a, b}, nil, &out, &out, "")
	if code != 1 {
		t.Errorf("expected exit 1 (files differ), got %d", code)
	}
	if out.Len() == 0 {
		t.Error("expected diff output")
	}
}

func TestDiff_TwoEmptyFiles(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a")
	b := filepath.Join(dir, "b")
	os.WriteFile(a, []byte(""), 0644)
	os.WriteFile(b, []byte(""), 0644)
	var out bytes.Buffer
	code := run([]string{a, b}, nil, &out, &out, "")
	if code != 0 {
		t.Errorf("expected exit 0 (identical), got %d", code)
	}
}

func TestDiff_IgnoreWhitespace(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a")
	b := filepath.Join(dir, "b")
	os.WriteFile(a, []byte("hello   world\n"), 0644)
	os.WriteFile(b, []byte("hello world\n"), 0644)
	var out bytes.Buffer
	code := run([]string{"-w", a, b}, nil, &out, &out, "")
	if code != 0 {
		t.Errorf("expected exit 0 with -w, got %d", code)
	}
}

func TestDiff_IgnoreBlankLines(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a")
	b := filepath.Join(dir, "b")
	os.WriteFile(a, []byte("hello\n\nworld\n"), 0644)
	os.WriteFile(b, []byte("hello\nworld\n"), 0644)
	var out bytes.Buffer
	code := run([]string{"-B", a, b}, nil, &out, &out, "")
	if code != 0 {
		t.Errorf("expected exit 0 with -B, got %d", code)
	}
}

func TestDiff_CrLfLineEndings(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a")
	b := filepath.Join(dir, "b")
	os.WriteFile(a, []byte("hello\r\nworld\n"), 0644)
	os.WriteFile(b, []byte("hello\nworld\n"), 0644)
	var out bytes.Buffer
	code := run([]string{a, b}, nil, &out, &out, "")
	if code != 1 {
		t.Errorf("expected exit 1 (differ), got %d", code)
	}
}

func TestDiff_BinaryWarning(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a")
	b := filepath.Join(dir, "b")
	os.WriteFile(a, []byte("text\n"), 0644)
	os.WriteFile(b, []byte("text\x00binary\n"), 0644)
	var out bytes.Buffer
	code := run([]string{a, b}, nil, &out, &out, "")
	if code != 1 {
		t.Errorf("expected exit 1 (differ), got %d", code)
	}
}

func TestDiff_RecursiveDirs(t *testing.T) {
	dir := t.TempDir()
	dirA := filepath.Join(dir, "a")
	dirB := filepath.Join(dir, "b")
	os.MkdirAll(dirA, 0755)
	os.MkdirAll(dirB, 0755)
	os.WriteFile(filepath.Join(dirA, "same.txt"), []byte("hello\n"), 0644)
	os.WriteFile(filepath.Join(dirB, "same.txt"), []byte("hello\n"), 0644)
	os.WriteFile(filepath.Join(dirA, "diff.txt"), []byte("a\n"), 0644)
	os.WriteFile(filepath.Join(dirB, "diff.txt"), []byte("b\n"), 0644)
	var out bytes.Buffer
	code := run([]string{"-r", dirA, dirB}, nil, &out, &out, "")
	if code != 1 {
		t.Errorf("expected exit 1 (dirs differ), got %d", code)
	}
}

func TestDiff_RecursiveDirsIdentical(t *testing.T) {
	dir := t.TempDir()
	dirA := filepath.Join(dir, "a")
	dirB := filepath.Join(dir, "b")
	os.MkdirAll(dirA, 0755)
	os.MkdirAll(dirB, 0755)
	os.WriteFile(filepath.Join(dirA, "f.txt"), []byte("same\n"), 0644)
	os.WriteFile(filepath.Join(dirB, "f.txt"), []byte("same\n"), 0644)
	var out bytes.Buffer
	code := run([]string{"-r", dirA, dirB}, nil, &out, &out, "")
	if code != 0 {
		t.Errorf("expected exit 0 (identical dirs), got %d", code)
	}
}

func TestDiff_NewFile(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a")
	b := filepath.Join(dir, "b")
	os.WriteFile(a, []byte(""), 0644)
	os.WriteFile(b, []byte("hello\n"), 0644)
	var out bytes.Buffer
	code := run([]string{"-N", a, b}, nil, &out, &out, "")
	// -N treats absent files as empty
	if code != 1 {
		t.Errorf("expected exit 1 (files differ), got %d", code)
	}
}

func TestDiff_RecursiveNonRegular(t *testing.T) {
	// Recursive diff where one side has a file and the other doesn't.
	dir := t.TempDir()
	dirA := filepath.Join(dir, "A")
	dirB := filepath.Join(dir, "B")
	os.MkdirAll(dirA, 0755)
	os.MkdirAll(dirB, 0755)
	os.WriteFile(filepath.Join(dirA, "only-in-a"), []byte("x\n"), 0644)
	var out bytes.Buffer
	code := run([]string{"-r", dirA, dirB}, nil, &out, &out, "")
	if code != 1 {
		t.Errorf("expected exit 1, got %d", code)
	}
	if !strings.Contains(out.String(), "Only in") {
		t.Errorf("expected 'Only in' message, got %q", out.String())
	}
}

func TestDiff_RecursiveBothMissing(t *testing.T) {
	dir := t.TempDir()
	dirA := filepath.Join(dir, "A")
	dirB := filepath.Join(dir, "B")
	os.MkdirAll(dirA, 0755)
	os.MkdirAll(dirB, 0755)
	os.WriteFile(filepath.Join(dirA, "shared"), []byte("same\n"), 0644)
	os.WriteFile(filepath.Join(dirB, "shared"), []byte("same\n"), 0644)
	os.WriteFile(filepath.Join(dirB, "only-b"), []byte("y\n"), 0644)
	var out bytes.Buffer
	code := run([]string{"-r", dirA, dirB}, nil, &out, &out, "")
	// Should show "Only in B"
	if code != 1 {
		t.Errorf("expected exit 1, got %d", code)
	}
}

func TestDiffContextFormat(t *testing.T) {
	dir := t.TempDir()
	f1 := filepath.Join(dir, "a")
	f2 := filepath.Join(dir, "b")
	os.WriteFile(f1, []byte("line1\nline2\nline3\n"), 0644)
	os.WriteFile(f2, []byte("line1\nlineX\nline3\n"), 0644)
	var buf bytes.Buffer
	code := run([]string{"-c", f1, f2}, nil, &buf, &buf, "")
	_ = code
}
func TestDiffUnifiedFormat(t *testing.T) {
	dir := t.TempDir()
	f1 := filepath.Join(dir, "a")
	f2 := filepath.Join(dir, "b")
	os.WriteFile(f1, []byte("line1\nline2\n"), 0644)
	os.WriteFile(f2, []byte("line1\nlineX\n"), 0644)
	var buf bytes.Buffer
	code := run([]string{"-u", f1, f2}, nil, &buf, &buf, "")
	if code != 1 {
		t.Errorf("diff -u: exit %d, want 1", code)
	}
}
func TestDiffIdentical(t *testing.T) {
	dir := t.TempDir()
	f1 := filepath.Join(dir, "a")
	f2 := filepath.Join(dir, "b")
	os.WriteFile(f1, []byte("same\n"), 0644)
	os.WriteFile(f2, []byte("same\n"), 0644)
	var buf bytes.Buffer
	code := run([]string{f1, f2}, nil, &buf, &buf, "")
	if code != 0 {
		t.Errorf("diff identical: exit %d, want 0", code)
	}
}
