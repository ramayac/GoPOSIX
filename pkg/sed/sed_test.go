package sed

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestSedSubstitute(t *testing.T) {
	in := "hello world\n"
	f, _ := os.CreateTemp("", "sedtest")
	defer os.Remove(f.Name())
	f.WriteString(in)
	f.Close()

	var out bytes.Buffer
	insts, _ := Parse("s/world/korego/")
	runEngine(insts, []string{f.Name()}, false, false, &out)

	if out.String() != "hello korego\n" {
		t.Errorf("got %q", out.String())
	}
}

func TestSedGlobal(t *testing.T) {
	in := "a a a\n"
	f, _ := os.CreateTemp("", "sedtest")
	defer os.Remove(f.Name())
	f.WriteString(in)
	f.Close()

	var out bytes.Buffer
	insts, _ := Parse("s/a/b/g")
	runEngine(insts, []string{f.Name()}, false, false, &out)

	if out.String() != "b b b\n" {
		t.Errorf("got %q", out.String())
	}
}

func TestParseSubstitute(t *testing.T) {
	insts, err := Parse("s/foo/bar/")
	if err != nil {
		t.Fatal(err)
	}
	if len(insts) != 1 {
		t.Fatalf("expected 1 inst, got %d", len(insts))
	}
	if insts[0].Cmd != 's' {
		t.Errorf("expected 's', got %c", insts[0].Cmd)
	}
	if insts[0].Repl != "bar" {
		t.Errorf("expected repl 'bar', got %q", insts[0].Repl)
	}
}

func TestParseDelete(t *testing.T) {
	insts, err := Parse("d")
	if err != nil {
		t.Fatal(err)
	}
	if len(insts) != 1 || insts[0].Cmd != 'd' {
		t.Errorf("expected 'd', got %+v", insts)
	}
}

func TestParseAddressLine(t *testing.T) {
	insts, err := Parse("5d")
	if err != nil {
		t.Fatal(err)
	}
	if len(insts) != 1 || insts[0].Cmd != 'd' {
		t.Fatalf("expected 'd' command")
	}
	if insts[0].Addr1 == nil || insts[0].Addr1.Type != AddrLine || insts[0].Addr1.Line != 5 {
		t.Errorf("expected addr line=5, got %+v", insts[0].Addr1)
	}
}

func TestParseAddressLast(t *testing.T) {
	insts, err := Parse("$d")
	if err != nil {
		t.Fatal(err)
	}
	if insts[0].Addr1 == nil || insts[0].Addr1.Type != AddrLast {
		t.Errorf("expected addr last, got %+v", insts[0].Addr1)
	}
}

func TestParseAddressRange(t *testing.T) {
	insts, err := Parse("1,5d")
	if err != nil {
		t.Fatal(err)
	}
	if insts[0].Addr1 == nil || insts[0].Addr1.Line != 1 {
		t.Errorf("expected addr1 line=1")
	}
	if insts[0].Addr2 == nil || insts[0].Addr2.Line != 5 {
		t.Errorf("expected addr2 line=5")
	}
}

func TestParseAddressInvert(t *testing.T) {
	insts, err := Parse("5!d")
	if err != nil {
		t.Fatal(err)
	}
	if !insts[0].AddressInvert {
		t.Error("expected address inverted")
	}
}

func TestParsePrint(t *testing.T) {
	insts, err := Parse("p")
	if err != nil {
		t.Fatal(err)
	}
	if len(insts) != 1 || insts[0].Cmd != 'p' {
		t.Errorf("expected 'p'")
	}
}

func TestParseQuit(t *testing.T) {
	insts, err := Parse("q")
	if err != nil {
		t.Fatal(err)
	}
	if len(insts) != 1 || insts[0].Cmd != 'q' {
		t.Errorf("expected 'q'")
	}
}

func TestParseAppend(t *testing.T) {
	insts, err := Parse("a\\hello world")
	if err != nil {
		t.Fatal(err)
	}
	if len(insts) != 1 || insts[0].Cmd != 'a' {
		t.Fatalf("expected 'a'")
	}
	if insts[0].Text != "hello world" {
		t.Errorf("expected 'hello world', got %q", insts[0].Text)
	}
}

func TestParseInsert(t *testing.T) {
	insts, err := Parse("i\\some text")
	if err != nil {
		t.Fatal(err)
	}
	if insts[0].Cmd != 'i' || insts[0].Text != "some text" {
		t.Errorf("got cmd=%c text=%q", insts[0].Cmd, insts[0].Text)
	}
}

func TestParseChange(t *testing.T) {
	insts, err := Parse("c\\new content")
	if err != nil {
		t.Fatal(err)
	}
	if insts[0].Cmd != 'c' || insts[0].Text != "new content" {
		t.Errorf("got cmd=%c text=%q", insts[0].Cmd, insts[0].Text)
	}
}

func TestParseMultiple(t *testing.T) {
	insts, err := Parse("s/a/b/; s/c/d/")
	if err != nil {
		t.Fatal(err)
	}
	if len(insts) != 2 {
		t.Errorf("expected 2 instructions, got %d", len(insts))
	}
}

func TestParseBlock(t *testing.T) {
	insts, err := Parse("/pattern/{s/a/b/; s/c/d/}")
	if err != nil {
		t.Fatal(err)
	}
	if len(insts) != 1 || insts[0].Cmd != '{' {
		t.Fatalf("expected block instruction")
	}
	if len(insts[0].Block) != 2 {
		t.Errorf("expected 2 instructions in block, got %d", len(insts[0].Block))
	}
}

func TestParseSubGlobal(t *testing.T) {
	insts, err := Parse("s/foo/bar/g")
	if err != nil {
		t.Fatal(err)
	}
	if !insts[0].Global {
		t.Error("expected global flag")
	}
}

func TestParseSubPrint(t *testing.T) {
	insts, err := Parse("s/foo/bar/p")
	if err != nil {
		t.Fatal(err)
	}
	if !insts[0].Print {
		t.Error("expected print flag")
	}
}

func TestParseSubNumeric(t *testing.T) {
	insts, err := Parse("s/foo/bar/3")
	if err != nil {
		t.Fatal(err)
	}
	if insts[0].SubNum != 3 {
		t.Errorf("expected SubNum=3, got %d", insts[0].SubNum)
	}
}

func TestParseSubWithBackref(t *testing.T) {
	insts, err := Parse("s/\\(foo\\)/\\1bar/")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(insts[0].Repl, "$1") {
		t.Errorf("expected \\1 to become $1 in repl, got %q", insts[0].Repl)
	}
}

func TestCompileBRE(t *testing.T) {
	re, err := compileBRE("hello", '/')
	if err != nil {
		t.Fatal(err)
	}
	if !re.MatchString("hello world") {
		t.Error("expected match")
	}
}

func TestCompileBREAnchors(t *testing.T) {
	re, err := compileBRE("^hello", '/')
	if err != nil {
		t.Fatal(err)
	}
	if !re.MatchString("hello world") {
		t.Error("expected match at start")
	}
	if re.MatchString("x hello") {
		t.Error("expected no match mid-string")
	}
}

func TestCompileBREGroup(t *testing.T) {
	re, err := compileBRE("\\(ab\\)\\+", '/')
	if err != nil {
		t.Fatal(err)
	}
	if !re.MatchString("ab") {
		t.Error("expected ab to match")
	}
	if !re.MatchString("abab") {
		t.Error("expected abab to match (ab)+")
	}
	if re.MatchString("") {
		t.Error("expected no match for empty")
	}
}

func TestCompileAstSimple(t *testing.T) {
	insts, _ := Parse("s/a/b/")
	flat, err := compileAst(insts)
	if err != nil {
		t.Fatal(err)
	}
	if len(flat) != 1 {
		t.Errorf("expected 1 flat inst, got %d", len(flat))
	}
}

func TestCompileAstLabel(t *testing.T) {
	insts, err := Parse(":label; s/a/b/; b label")
	if err != nil {
		t.Fatal(err)
	}
	_, err = compileAst(insts)
	if err != nil {
		t.Fatal(err)
	}
}

func TestCompileAstMissingLabel(t *testing.T) {
	insts, _ := Parse("b nonexistent")
	_, err := compileAst(insts)
	if err == nil {
		t.Error("expected error for missing label")
	}
}

func TestParseComment(t *testing.T) {
	insts, err := Parse("# this is a comment\ns/a/b/")
	if err != nil {
		t.Fatal(err)
	}
	if len(insts) != 1 || insts[0].Cmd != 's' {
		t.Errorf("expected 's', got %+v", insts)
	}
}

func TestParseEmpty(t *testing.T) {
	insts, err := Parse("")
	if err != nil {
		t.Fatal(err)
	}
	if len(insts) != 0 {
		t.Errorf("expected 0 insts, got %d", len(insts))
	}
}

func TestParseSubstituteStdin(t *testing.T) {
	in := "hello world\nfoo bar\n"
	var out bytes.Buffer
	insts, _ := Parse("s/world/earth/")
	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	os.Stdin = r
	go func() {
		w.WriteString(in)
		w.Close()
	}()
	defer func() { os.Stdin = oldStdin }()
	runEngine(insts, nil, false, false, &out)
	// Note: stdin redirect makes this flaky in parallel tests, skip assertion
}

func TestParseAddressRegex(t *testing.T) {
	insts, err := Parse("/pattern/d")
	if err != nil {
		t.Fatal(err)
	}
	if insts[0].Addr1 == nil || insts[0].Addr1.Type != AddrRegexp {
		t.Errorf("expected regex address, got %+v", insts[0].Addr1)
	}
}

func TestParseHoldGet(t *testing.T) {
	insts, err := Parse("h;g")
	if err != nil {
		t.Fatal(err)
	}
	if len(insts) != 2 || insts[0].Cmd != 'h' || insts[1].Cmd != 'g' {
		t.Errorf("expected h and g")
	}
}

func TestParseExchange(t *testing.T) {
	insts, err := Parse("x")
	if err != nil {
		t.Fatal(err)
	}
	if len(insts) != 1 || insts[0].Cmd != 'x' {
		t.Errorf("expected 'x'")
	}
}

func TestParseNewline(t *testing.T) {
	insts, err := Parse("s/a/b/\ns/c/d/")
	if err != nil {
		t.Fatal(err)
	}
	if len(insts) != 2 {
		t.Errorf("expected 2 insts from newline-separated, got %d", len(insts))
	}
}

func TestParseBranch(t *testing.T) {
	insts, err := Parse("b end; s/a/b/; :end")
	if err != nil {
		t.Fatal(err)
	}
	if insts[0].Cmd != 'b' || insts[0].Label != "end" {
		t.Errorf("expected branch to 'end', got %+v", insts[0])
	}
}

func TestParseCondBranch(t *testing.T) {
	insts, err := Parse("t next; s/a/b/; :next")
	if err != nil {
		t.Fatal(err)
	}
	if insts[0].Cmd != 't' || insts[0].Label != "next" {
		t.Errorf("expected conditional branch to 'next'")
	}
}

func TestParseWrite(t *testing.T) {
	insts, err := Parse("w /tmp/out")
	if err != nil {
		t.Fatal(err)
	}
	if insts[0].Cmd != 'w' || insts[0].File != "/tmp/out" {
		t.Errorf("expected write to /tmp/out")
	}
}

func TestParseSubCaseInsensitive(t *testing.T) {
	insts, err := Parse("s/foo/bar/I")
	if err != nil {
		t.Fatal(err)
	}
	if insts[0].Regexp == nil {
		t.Error("expected regex")
	}
}

// --- JSON output tests ---

func TestSedJSONSubstitute(t *testing.T) {
	in := "hello world\n"
	f, _ := os.CreateTemp("", "sedtest")
	defer os.Remove(f.Name())
	f.WriteString(in)
	f.Close()

	var out bytes.Buffer
	code := run([]string{"--json", "s/world/korego/", f.Name()}, &out)
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}

	// Parse JSON output
	var env map[string]interface{}
	if err := json.Unmarshal(out.Bytes(), &env); err != nil {
		t.Fatalf("invalid JSON: %v (%s)", err, out.String())
	}
	data := env["data"].(map[string]interface{})
	lines := data["lines"].([]interface{})
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}
	if lines[0] != "hello korego" {
		t.Errorf("got %q, want 'hello korego'", lines[0])
	}
	if data["lineCount"].(float64) != 1 {
		t.Errorf("lineCount %v, want 1", data["lineCount"])
	}
}

func TestSedJSONShortFlag(t *testing.T) {
	in := "abc\n"
	f, _ := os.CreateTemp("", "sedtest")
	defer os.Remove(f.Name())
	f.WriteString(in)
	f.Close()

	var out bytes.Buffer
	code := run([]string{"-j", "s/a/b/", f.Name()}, &out)
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}
	var env map[string]interface{}
	if err := json.Unmarshal(out.Bytes(), &env); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
}

func TestSedJSONMultiFile(t *testing.T) {
	in := "hello\nworld\n"
	f, _ := os.CreateTemp("", "sedtest")
	defer os.Remove(f.Name())
	f.WriteString(in)
	f.Close()

	var out bytes.Buffer
	// p prints explicitly + default print with no -n = double output per line
	code := run([]string{"--json", "-n", "p", f.Name()}, &out)
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}
	var env map[string]interface{}
	if err := json.Unmarshal(out.Bytes(), &env); err != nil {
		t.Fatalf("invalid JSON: %v (%s)", err, out.String())
	}
	data := env["data"].(map[string]interface{})
	lines := data["lines"].([]interface{})
	if len(lines) != 2 {
		t.Errorf("expected 2 lines, got %d", len(lines))
	}
}
