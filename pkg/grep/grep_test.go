package grep

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/ramayac/goposix/pkg/common"
)

// =============================================================================
// Run() library-layer tests (kept from original)
// =============================================================================

func TestGrepBasic(t *testing.T) {
	in := "hello\nworld\nhello world\n"
	matches, _ := Run(strings.NewReader(in), "file", nil, []string{"hello"}, false, true, false)
	if len(matches) != 2 {
		t.Errorf("expected 2 matches, got %d", len(matches))
	}
}

func TestGrepInvert(t *testing.T) {
	in := "hello\nworld\nhello world\n"
	matches, _ := Run(strings.NewReader(in), "file", nil, []string{"hello"}, true, true, false)
	if len(matches) != 1 || matches[0].Text != "world" {
		t.Errorf("expected 1 match 'world', got %v", matches)
	}
}

func TestGrepRegex(t *testing.T) {
	in := "hello\nworld\nhello world\n"
	re := regexp.MustCompile(`^w`)
	matches, _ := Run(strings.NewReader(in), "file", re, nil, false, false, false)
	if len(matches) != 1 {
		t.Errorf("expected 1 match, got %d", len(matches))
	}
	if matches[0].Text != "world" {
		t.Errorf("expected 'world', got %q", matches[0].Text)
	}
}

func TestGrepRegexIgnoreCase(t *testing.T) {
	in := "Hello\nWORLD\n"
	re := regexp.MustCompile(`(?i)hello`)
	matches, _ := Run(strings.NewReader(in), "file", re, nil, false, false, false)
	if len(matches) != 1 {
		t.Errorf("expected 1 match, got %d", len(matches))
	}
}

func TestGrepLineRegexp(t *testing.T) {
	in := "hello\nworld\nhello world\n"
	re := regexp.MustCompile(`^hello$`)
	matches, _ := Run(strings.NewReader(in), "file", re, nil, false, false, false)
	if len(matches) != 1 {
		t.Errorf("expected 1 exact line match, got %d", len(matches))
	}
}

func TestGrepWordRegexp(t *testing.T) {
	in := "hello\nworld\nhelloworld\n"
	re := regexp.MustCompile(`\bhello\b`)
	matches, _ := Run(strings.NewReader(in), "file", re, nil, false, false, false)
	if len(matches) != 1 {
		t.Errorf("expected 1 word-boundary match, got %d", len(matches))
	}
	if matches[0].Text != "hello" {
		t.Errorf("expected 'hello', got %q", matches[0].Text)
	}
}

func TestGrepEmptyInput(t *testing.T) {
	matches, err := Run(strings.NewReader(""), "file", nil, []string{"x"}, false, true, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Errorf("expected 0 matches, got %d", len(matches))
	}
}

func TestGrepNoMatch(t *testing.T) {
	in := "hello\nworld\n"
	matches, _ := Run(strings.NewReader(in), "file", nil, []string{"xyz"}, false, true, false)
	if len(matches) != 0 {
		t.Errorf("expected 0 matches, got %d", len(matches))
	}
}

func TestGrepMultiplePatterns(t *testing.T) {
	in := "apple\nbanana\ncherry\ndate\n"
	matches, _ := Run(strings.NewReader(in), "file", nil, []string{"apple", "cherry"}, false, true, false)
	if len(matches) != 2 {
		t.Errorf("expected 2 matches, got %d", len(matches))
	}
}

func TestGrepLineNumbers(t *testing.T) {
	in := "a\nb\nc\na\n"
	matches, _ := Run(strings.NewReader(in), "file", nil, []string{"a"}, false, true, false)
	if len(matches) != 2 {
		t.Errorf("expected 2 matches, got %d", len(matches))
	}
	if matches[0].Line != 1 {
		t.Errorf("expected line 1, got %d", matches[0].Line)
	}
	if matches[1].Line != 4 {
		t.Errorf("expected line 4, got %d", matches[1].Line)
	}
}

func TestGrepRegexOnlyMatching(t *testing.T) {
	in := "hello world\n"
	re := regexp.MustCompile(`wo\w+`)
	matches, _ := Run(strings.NewReader(in), "file", re, nil, false, false, false)
	if len(matches) != 1 {
		t.Fatalf("expected 1 match")
	}
	if len(matches[0].Matches) != 1 || matches[0].Matches[0] != "world" {
		t.Errorf("expected capture 'world', got %v", matches[0].Matches)
	}
}

func TestGrepFixedIgnoreCase(t *testing.T) {
	in := "HELLO\nworld\n"
	re := regexp.MustCompile(`(?i)hello`)
	matches, _ := Run(strings.NewReader(in), "file", re, nil, false, false, false)
	if len(matches) != 1 {
		t.Errorf("expected 1 case-insensitive match, got %d", len(matches))
	}
}

func TestGrepInvertRegex(t *testing.T) {
	in := "hello\nworld\n"
	re := regexp.MustCompile(`hello`)
	matches, _ := Run(strings.NewReader(in), "file", re, nil, true, false, false)
	if len(matches) != 1 || matches[0].Text != "world" {
		t.Errorf("expected 1 inverted match 'world', got %v", matches)
	}
}

func TestGrepInvertWithMultipleMatches(t *testing.T) {
	in := "a\nb\nc\na\n"
	re := regexp.MustCompile(`a`)
	matches, _ := Run(strings.NewReader(in), "file", re, nil, true, false, false)
	if len(matches) != 2 {
		t.Errorf("expected 2 non-a matches, got %d", len(matches))
	}
}

func TestGrepLineRegexpFixedMode(t *testing.T) {
	in := "exact\nnot exact\n"
	matches, _ := Run(strings.NewReader(in), "file", nil, []string{"exact"}, false, true, true)
	if len(matches) != 1 {
		t.Errorf("expected 1 exact line match, got %d", len(matches))
	}
}

func TestGrepMultipleMatchesPerLine(t *testing.T) {
	re := regexp.MustCompile(`a`)
	in := "banana\n"
	matches, _ := Run(strings.NewReader(in), "file", re, nil, false, false, false)
	if len(matches) != 1 {
		t.Fatalf("expected 1 line match")
	}
	if len(matches[0].Matches) != 3 {
		t.Errorf("expected 3 'a' on line, got %d matches: %v", len(matches[0].Matches), matches[0].Matches)
	}
}

func TestGrepFixedFullLineMatch(t *testing.T) {
	in := "hello\nhello world\n"
	matches, _ := Run(strings.NewReader(in), "file", nil, []string{"hello"}, false, true, true)
	if len(matches) != 1 {
		t.Errorf("expected 1 exact whole-line match, got %d", len(matches))
	}
}

func TestGrepFilename(t *testing.T) {
	in := "hello\n"
	matches, _ := Run(strings.NewReader(in), "/path/to/file.txt", nil, []string{"hello"}, false, true, false)
	if len(matches) != 1 {
		t.Fatalf("expected 1 match")
	}
	if matches[0].File != "/path/to/file.txt" {
		t.Errorf("expected filename, got %q", matches[0].File)
	}
}

// =============================================================================
// CLI layer tests via grepRun (testable core with injected io)
// =============================================================================

// simpleGrep runs grepRun with optional stdin string, returns exitCode, stdout, stderr.
func simpleGrep(args []string, stdin string) (int, string, string) {
	var outBuf, errBuf bytes.Buffer
	stdinR := io.Reader(strings.NewReader(stdin))
	code := grepRun(args, &outBuf, &errBuf, stdinR, "")
	return code, outBuf.String(), errBuf.String()
}

func TestCLI_Basic(t *testing.T) {
	code, out, errStr := simpleGrep([]string{"Alice", "../../testdata/grep/alice.txt"}, "")
	if code != 0 {
		t.Fatalf("exit code %d: %s", code, errStr)
	}
	if !strings.Contains(out, "Alice was beginning") {
		t.Errorf("expected 'Alice was beginning' in output, got: %s", out)
	}
}

func TestCLI_Count(t *testing.T) {
	code, out, _ := simpleGrep([]string{"-c", "Alice", "../../testdata/grep/alice.txt"}, "")
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	if strings.TrimSpace(out) != "2" {
		t.Errorf("expected count 2, got %q", strings.TrimSpace(out))
	}
}

func TestCLI_IgnoreCase(t *testing.T) {
	code, out, _ := simpleGrep([]string{"-i", "alice", "../../testdata/grep/alice.txt"}, "")
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	if !strings.Contains(out, "Alice was beginning") {
		t.Errorf("expected case-insensitive match, got: %s", out)
	}
}

func TestCLI_Invert(t *testing.T) {
	code, out, _ := simpleGrep([]string{"-v", "Alice", "../../testdata/grep/numbers.txt"}, "")
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	// All lines should appear (none contain "Alice")
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 10 {
		t.Errorf("expected 10 inverted-match lines, got %d", len(lines))
	}
}

func TestCLI_LineNumber(t *testing.T) {
	code, out, _ := simpleGrep([]string{"-n", "Alice", "../../testdata/grep/alice.txt"}, "")
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	if !strings.Contains(out, "1:Alice was beginning") {
		t.Errorf("expected line number prefix, got: %s", out)
	}
}

func TestCLI_FilesWithMatches(t *testing.T) {
	code, out, _ := simpleGrep([]string{"-l", "hello", "../../testdata/grep/subdir/file1.txt", "../../testdata/grep/subdir/file2.txt"}, "")
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	// file1.txt has "hello", file2.txt has "goodbye" and "cruel world" (no plain "hello")
	if !strings.Contains(out, "file1.txt") {
		t.Errorf("expected file1.txt in -l output, got: %s", out)
	}
	if strings.Contains(out, "file2.txt") {
		t.Errorf("file2.txt should not appear (-l with 'hello'), got: %s", out)
	}
}

func TestCLI_FilesWithoutMatch(t *testing.T) {
	code, out, _ := simpleGrep([]string{"-L", "xyzzy", "../../testdata/grep/subdir/file1.txt", "../../testdata/grep/subdir/file2.txt"}, "")
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	// Both files should appear since neither contains "xyzzy"
	if !strings.Contains(out, "file1.txt") {
		t.Errorf("expected file1.txt in -L output, got: %s", out)
	}
	if !strings.Contains(out, "file2.txt") {
		t.Errorf("expected file2.txt in -L output, got: %s", out)
	}
}

func TestCLI_FixedStrings(t *testing.T) {
	code, out, _ := simpleGrep([]string{"-F", "was beginning", "../../testdata/grep/alice.txt"}, "")
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	if !strings.Contains(out, "Alice was beginning") {
		t.Errorf("expected fixed-string match, got: %s", out)
	}
}

func TestCLI_WordRegexp(t *testing.T) {
	// "the" appears inside "their" and "there" etc, but -w should only match standalone "the"
	code, out, _ := simpleGrep([]string{"-w", "the", "../../testdata/grep/alice.txt"}, "")
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	// Should NOT match "their" or "there", only "the" as standalone word
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, "their") || strings.Contains(line, "there") {
			t.Errorf("-w should not match 'their'/'there', got: %s", line)
		}
	}
}

func TestCLI_LineRegexp(t *testing.T) {
	code, out, _ := simpleGrep([]string{"-x", "one", "../../testdata/grep/numbers.txt"}, "")
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	if strings.TrimSpace(out) != "one" {
		t.Errorf("expected exact line 'one', got: %q", strings.TrimSpace(out))
	}
}

func TestCLI_LineRegexpNoMatch(t *testing.T) {
	// "on" is not an exact line match for any line
	code, _, errStr := simpleGrep([]string{"-x", "on", "../../testdata/grep/numbers.txt"}, "")
	if code != 1 {
		t.Fatalf("expected exit code 1 (no match), got %d: %s", code, errStr)
	}
}

func TestCLI_Recursive(t *testing.T) {
	code, out, _ := simpleGrep([]string{"-r", "hello", "../../testdata/grep/subdir"}, "")
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	if !strings.Contains(out, "file1.txt") {
		t.Errorf("expected file1.txt in recursive output, got: %s", out)
	}
}

func TestCLI_RecursiveDashR(t *testing.T) {
	// -R is alias for -r
	code, out, _ := simpleGrep([]string{"-R", "hello", "../../testdata/grep/subdir"}, "")
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	if !strings.Contains(out, "file1.txt") {
		t.Errorf("expected file1.txt in -R output, got: %s", out)
	}
}

func TestCLI_FilePattern(t *testing.T) {
	code, out, _ := simpleGrep([]string{"-f", "../../testdata/grep/patterns.txt", "../../testdata/grep/alice.txt"}, "")
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	if !strings.Contains(out, "Alice") && !strings.Contains(out, "Rabbit") {
		t.Errorf("expected matches from patterns file, got: %s", out)
	}
}

func TestCLI_RegexpFlag(t *testing.T) {
	code, out, _ := simpleGrep([]string{"-e", "Alice", "-e", "Rabbit", "../../testdata/grep/alice.txt"}, "")
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	if !strings.Contains(out, "Alice") || !strings.Contains(out, "Rabbit") {
		t.Errorf("expected matches for both patterns, got: %s", out)
	}
}

func TestCLI_Stdin(t *testing.T) {
	code, out, _ := simpleGrep([]string{"hello"}, "hello world\nhello\nworld\n")
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 2 {
		t.Errorf("expected 2 matches from stdin, got %d: %q", len(lines), out)
	}
}

func TestCLI_StdinDash(t *testing.T) {
	// "-" as positional becomes pattern; stdin has "hello world\n" → no "-" → exit 1
	code, _, _ := simpleGrep([]string{"-"}, "hello world\n")
	if code != 1 {
		t.Fatalf("expected exit code 1 (no match), got %d", code)
	}
}

func TestCLI_Json(t *testing.T) {
	code, out, _ := simpleGrep([]string{"--json", "Alice", "../../testdata/grep/alice.txt"}, "")
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	var env common.JSONEnvelope
	if err := json.Unmarshal([]byte(out), &env); err != nil {
		t.Fatalf("invalid JSON: %v\nraw: %s", err, out)
	}
	if env.Command != "grep" {
		t.Errorf("expected command 'grep', got %q", env.Command)
	}
}

func TestCLI_Quiet(t *testing.T) {
	code, out, _ := simpleGrep([]string{"-q", "Alice", "../../testdata/grep/alice.txt"}, "")
	if code != 0 {
		t.Fatalf("expected exit 0 for found match, got %d", code)
	}
	if out != "" {
		t.Errorf("expected no output in quiet mode, got: %q", out)
	}
}

func TestCLI_QuietNoMatch(t *testing.T) {
	code, out, _ := simpleGrep([]string{"-q", "xyzzy", "../../testdata/grep/alice.txt"}, "")
	if code != 1 {
		t.Fatalf("expected exit 1 for no match, got %d", code)
	}
	if out != "" {
		t.Errorf("expected no output in quiet mode, got: %q", out)
	}
}

func TestCLI_MissingPattern(t *testing.T) {
	code, _, errStr := simpleGrep([]string{}, "")
	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
	if !strings.Contains(errStr, "missing pattern") {
		t.Errorf("expected 'missing pattern' error, got: %q", errStr)
	}
}

func TestCLI_FileNotFound(t *testing.T) {
	code, _, errStr := simpleGrep([]string{"x", "nonexistent_file.txt"}, "")
	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
	if !strings.Contains(errStr, "nonexistent_file") {
		t.Errorf("expected file-not-found error, got: %q", errStr)
	}
}

func TestCLI_FileNotFoundSuppress(t *testing.T) {
	// -s suppresses error messages
	code, _, errStr := simpleGrep([]string{"-s", "x", "nonexistent_file.txt"}, "")
	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
	if errStr != "" {
		t.Errorf("expected no stderr with -s, got: %q", errStr)
	}
}

func TestCLI_OnlyMatching(t *testing.T) {
	code, out, _ := simpleGrep([]string{"-o", "[A-Z][a-z]*", "../../testdata/grep/alice.txt"}, "")
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	if !strings.Contains(out, "Alice") {
		t.Errorf("expected 'Alice' in -o output, got: %s", out)
	}
}

func TestCLI_MultipleFiles(t *testing.T) {
	code, out, _ := simpleGrep([]string{"hello", "../../testdata/grep/subdir/file1.txt", "../../testdata/grep/subdir/file2.txt"}, "")
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	// Multi-file output should prefix with filename
	if !strings.Contains(out, "file1.txt:") {
		t.Errorf("expected filename prefix for multi-file, got: %s", out)
	}
}

func TestCLI_WithFilenameFlag(t *testing.T) {
	// -H forces filename prefix even with single file
	code, out, _ := simpleGrep([]string{"-H", "Alice", "../../testdata/grep/alice.txt"}, "")
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	if !strings.Contains(out, "../../testdata/grep/alice.txt:") {
		t.Errorf("expected filename prefix with -H, got: %s", out)
	}
}

func TestCLI_NoFilenameFlag(t *testing.T) {
	// -h suppresses filename prefix even with multiple files
	code, out, _ := simpleGrep([]string{"-h", "hello", "../../testdata/grep/subdir/file1.txt", "../../testdata/grep/subdir/file2.txt"}, "")
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	if strings.Contains(out, "file1.txt:") || strings.Contains(out, "file2.txt:") {
		t.Errorf("expected no filename prefix with -h, got: %s", out)
	}
}

func TestCLI_ExtendedRegexp(t *testing.T) {
	// -E enables extended regex (in Go, regex is always extended, so -E is a noop for syntax)
	code, out, _ := simpleGrep([]string{"-E", "(Alice|Rabbit)", "../../testdata/grep/alice.txt"}, "")
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	if !strings.Contains(out, "Alice") || !strings.Contains(out, "Rabbit") {
		t.Errorf("expected matches with -E, got: %s", out)
	}
}

func TestCLI_FixedIgnoreCase(t *testing.T) {
	code, out, _ := simpleGrep([]string{"-F", "-i", "alice", "../../testdata/grep/alice.txt"}, "")
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	if !strings.Contains(out, "Alice") {
		t.Errorf("expected case-insensitive fixed match, got: %s", out)
	}
}

func TestCLI_FixedWord(t *testing.T) {
	// -F -w: fixed string as whole word only
	code, out, _ := simpleGrep([]string{"-F", "-w", "the", "../../testdata/grep/alice.txt"}, "")
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, "their") || strings.Contains(line, "there") {
			t.Errorf("-Fw should not match 'their'/'there', got: %s", line)
		}
	}
}

func TestCLI_AfterContext(t *testing.T) {
	code, out, _ := simpleGrep([]string{"-A", "1", "seven", "../../testdata/grep/numbers.txt"}, "")
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	if !strings.Contains(out, "seven") {
		t.Errorf("expected match 'seven', got: %s", out)
	}
	if !strings.Contains(out, "eight") {
		t.Errorf("expected after-context 'eight', got: %s", out)
	}
}

func TestCLI_BeforeContext(t *testing.T) {
	code, out, _ := simpleGrep([]string{"-B", "1", "seven", "../../testdata/grep/numbers.txt"}, "")
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	if !strings.Contains(out, "seven") {
		t.Errorf("expected match 'seven', got: %s", out)
	}
	if !strings.Contains(out, "six") {
		t.Errorf("expected before-context 'six', got: %s", out)
	}
}

func TestCLI_Context(t *testing.T) {
	code, out, _ := simpleGrep([]string{"-C", "1", "seven", "../../testdata/grep/numbers.txt"}, "")
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	if !strings.Contains(out, "six") || !strings.Contains(out, "seven") || !strings.Contains(out, "eight") {
		t.Errorf("expected full context around 'seven', got: %s", out)
	}
}

func TestCLI_ContextWithSeparator(t *testing.T) {
	// Two matches far apart should get a '--' separator
	code, out, _ := simpleGrep([]string{"-C", "0", "one", "../../testdata/grep/numbers.txt"}, "")
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	if !strings.Contains(out, "one") {
		t.Errorf("expected match 'one', got: %s", out)
	}
}

func TestCLI_StdinFilePattern(t *testing.T) {
	// -f - reads patterns from stdin
	code, out, _ := simpleGrep([]string{"-f", "-", "../../testdata/grep/alice.txt"}, "Alice\nRabbit\n")
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	if !strings.Contains(out, "Alice") || !strings.Contains(out, "Rabbit") {
		t.Errorf("expected matches from stdin patterns, got: %s", out)
	}
}

func TestCLI_InvalidFlag(t *testing.T) {
	code, _, errStr := simpleGrep([]string{"--nonexistent"}, "")
	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
	if !strings.Contains(errStr, "unknown") {
		t.Errorf("expected unknown flag error, got: %q", errStr)
	}
}

func TestCLI_CountMultiFile(t *testing.T) {
	code, out, _ := simpleGrep([]string{"-c", "hello", "../../testdata/grep/subdir/file1.txt", "../../testdata/grep/subdir/file2.txt"}, "")
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	if !strings.Contains(out, "file1.txt:") {
		t.Errorf("expected filename prefix in multi-file count, got: %s", out)
	}
}

func TestCLI_CountStdin(t *testing.T) {
	code, out, _ := simpleGrep([]string{"-c", "hello"}, "hello\nworld\nhello\n")
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	if strings.TrimSpace(out) != "2" {
		t.Errorf("expected count 2, got %q", strings.TrimSpace(out))
	}
}

func TestCLI_EmptyPatternFile(t *testing.T) {
	// An empty pattern file should produce no matches
	code, out, errStr := simpleGrep([]string{"-f", "-", "../../testdata/grep/alice.txt"}, "\n")
	if code != 1 {
		t.Fatalf("expected exit 1 (no match), got %d: %s", code, errStr)
	}
	_ = out
}

func TestCLI_MultiplePatternFlags(t *testing.T) {
	code, out, _ := simpleGrep([]string{"-e", "Alice", "-e", "Rabbit", "-e", "sister", "../../testdata/grep/alice.txt"}, "")
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	count := strings.Count(out, "\n")
	if count < 4 {
		t.Errorf("expected at least 4 matching lines, got %d", count)
	}
}

func TestCLI_JsonNoMatch(t *testing.T) {
	code, out, _ := simpleGrep([]string{"--json", "xyzzy", "../../testdata/grep/alice.txt"}, "")
	// json mode with no match — should still return valid JSON with empty data
	var env common.JSONEnvelope
	if err := json.Unmarshal([]byte(out), &env); err != nil {
		t.Fatalf("invalid JSON: %v\nraw: %s", err, out)
	}
	_ = code
}

func TestCLI_InvalidRegex(t *testing.T) {
	code, _, errStr := simpleGrep([]string{"[invalid", "../../testdata/grep/alice.txt"}, "")
	if code != 2 {
		t.Fatalf("expected exit code 2 for invalid regex, got %d", code)
	}
	if !strings.Contains(errStr, "invalid regex") {
		t.Errorf("expected invalid regex error, got: %q", errStr)
	}
}

func TestCLI_StdinWithDashArg(t *testing.T) {
	code, out, _ := simpleGrep([]string{"hello", "-"}, "hello world\n")
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	if !strings.Contains(out, "hello world") {
		t.Errorf("expected match from stdin via '-', got: %s", out)
	}
}

func TestCLI_NoArgsStdin(t *testing.T) {
	// No args → reads stdin with missing pattern → should error
	code, _, errStr := simpleGrep([]string{}, "")
	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
	if !strings.Contains(errStr, "missing pattern") {
		t.Errorf("expected missing pattern error, got: %q", errStr)
	}
}

// BusyBox hardening: grep -r on symlink to dir should show filename prefix.
func TestGrepRecursiveOnSymlinkToDir(t *testing.T) {
	// Create test dir structure
	testDir, err := os.MkdirTemp("", "grep-symlink-test-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(testDir)

	fooDir := filepath.Join(testDir, "foo")
	os.Mkdir(fooDir, 0755)
	os.WriteFile(filepath.Join(fooDir, "file"), []byte("bar\n"), 0644)
	os.Symlink("foo", filepath.Join(testDir, "symfoo"))

	var out bytes.Buffer
	code := run([]string{"-r", ".", filepath.Join(testDir, "symfoo")}, nil, &out, &out, "")
	if code != 0 {
		t.Fatalf("grep -r exited with %d, want 0", code)
	}
	if !strings.Contains(out.String(), "file:bar") {
		t.Errorf("expected filename prefix in output, got: %q", out.String())
	}
}

// TestEgrepAlias verifies that prepending -E enables extended regex syntax.
func TestEgrepAlias(t *testing.T) {
	var outBuf, errBuf bytes.Buffer
	stdinR := strings.NewReader("hello\nworld\n")
	// egrepRun prepends -E; simulate same by calling grepRun with -E.
	code := grepRun([]string{"-E", "hel+o", "-"}, &outBuf, &errBuf, stdinR, "")
	if code != 0 {
		t.Fatalf("egrep-style exited with %d, want 0", code)
	}
	// + is a metachar in ERE; should match "hello"
	if outBuf.String() != "hello\n" {
		t.Errorf("expected 'hello\\n', got: %q", outBuf.String())
	}
}

// TestFgrepAlias verifies that prepending -F enables fixed-string matching.
func TestFgrepAlias(t *testing.T) {
	var outBuf, errBuf bytes.Buffer
	stdinR := strings.NewReader("hello.\ngoodbye.\nhelo\n")
	// fgrepRun prepends -F; simulate same by calling grepRun with -F.
	code := grepRun([]string{"-F", ".", "-"}, &outBuf, &errBuf, stdinR, "")
	if code != 0 {
		t.Fatalf("fgrep-style exited with %d, want 0", code)
	}
	// Literal dot match, not regex any-char
	if !strings.Contains(outBuf.String(), "hello.") {
		t.Errorf("expected literal dot match, got: %q", outBuf.String())
	}
}

func TestGrepBinaryHandling(t *testing.T) {
	// Create binary payload with NUL byte: "hello\x00world\n"
	binaryData := "hello\x00world\n"

	// 1. Without -a flag, it should print "Binary file (standard input) matches"
	{
		var outBuf, errBuf bytes.Buffer
		stdinR := strings.NewReader(binaryData)
		code := grepRun([]string{"hello", "-"}, &outBuf, &errBuf, stdinR, "")
		if code != 0 {
			t.Errorf("expected exit code 0, got %d", code)
		}
		got := outBuf.String()
		want := "Binary file (standard input) matches\n"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	}

	// 2. With -a flag, it should treat as text and print the matched line
	{
		var outBuf, errBuf bytes.Buffer
		stdinR := strings.NewReader(binaryData)
		code := grepRun([]string{"-a", "hello", "-"}, &outBuf, &errBuf, stdinR, "")
		if code != 0 {
			t.Errorf("expected exit code 0, got %d", code)
		}
		got := outBuf.String()
		want := "hello\x00world\n"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	}
}
