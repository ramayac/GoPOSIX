// Package grep implements the POSIX grep utility.
package grep

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
)

// GrepMatch represents a single matched line.
type GrepMatch struct {
	File    string   `json:"file,omitempty"`
	Line    int      `json:"line"`
	Text    string   `json:"text"`
	Matches []string `json:"matches"`
}

var spec = common.FlagSpec{
	Defs: []common.FlagDef{
		{Short: "i", Long: "ignore-case", Type: common.FlagBool},
		{Short: "v", Long: "invert-match", Type: common.FlagBool},
		{Short: "c", Long: "count", Type: common.FlagBool},
		{Short: "n", Long: "line-number", Type: common.FlagBool},
		{Short: "l", Long: "files-with-matches", Type: common.FlagBool},
		{Short: "L", Long: "files-without-match", Type: common.FlagBool},
		{Short: "o", Long: "only-matching", Type: common.FlagBool},
		{Short: "r", Long: "recursive", Type: common.FlagBool},
		{Short: "R", Long: "dereference-recursive", Type: common.FlagBool},
		{Short: "q", Long: "quiet", Type: common.FlagBool},
		{Short: "s", Long: "no-messages", Type: common.FlagBool},
		{Short: "H", Long: "with-filename", Type: common.FlagBool},
		{Short: "h", Long: "no-filename", Type: common.FlagBool},
		{Short: "f", Long: "file", Type: common.FlagValue},
		{Short: "e", Long: "regexp", Type: common.FlagValue},
		{Short: "E", Long: "extended-regexp", Type: common.FlagBool},
		{Short: "F", Long: "fixed-strings", Type: common.FlagBool},
		{Short: "w", Long: "word-regexp", Type: common.FlagBool},
		{Short: "x", Long: "line-regexp", Type: common.FlagBool},
		{Short: "A", Long: "after-context", Type: common.FlagValue},
		{Short: "B", Long: "before-context", Type: common.FlagValue},
		{Short: "C", Long: "context", Type: common.FlagValue},
		{Short: "a", Long: "text", Type: common.FlagBool},
		{Long: "json", Type: common.FlagBool},
	},
}

func Run(r io.Reader, filename string, re *regexp.Regexp, fixedPatterns []string, invert, fixed, lineRegexp bool) ([]GrepMatch, error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024)
	var matches []GrepMatch
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		text := scanner.Text()

		var matchFound bool
		var substrings []string

		if fixed {
			for _, pat := range fixedPatterns {
				var found bool
				if lineRegexp {
					found = text == pat
				} else {
					found = strings.Contains(text, pat)
				}
				if found {
					matchFound = true
					if !invert {
						substrings = append(substrings, pat)
					}
				}
			}
		} else if re != nil {
			if lineRegexp {
				matchFound = re.MatchString(text)
			} else {
				locs := re.FindAllString(text, -1)
				for _, loc := range locs {
					if loc != "" {
						substrings = append(substrings, loc)
					}
				}
				matchFound = len(locs) > 0
			}
		}

		if invert {
			matchFound = !matchFound
			substrings = nil
		}

		if matchFound {
			matches = append(matches, GrepMatch{
				File:    filename,
				Line:    lineNum,
				Text:    text,
				Matches: substrings,
			})
		}
	}
	return matches, scanner.Err()
}

func run(args []string, stdin io.Reader, stdout io.Writer) int {
	return grepRun(args, stdout, os.Stderr, os.Stdin)
}

// grepRun is the testable core of the grep CLI. All output goes to stdout or errOut,
// and stdin is read from stdinR.
func grepRun(args []string, stdout, errOut io.Writer, stdinR io.Reader) int {
	flags, err := common.ParseFlags(args, spec)
	if err != nil {
		fmt.Fprintf(errOut, "grep: %v\n", err)
		return 2
	}

	var paths []string
	var patterns []string
	ePats := flags.GetAll("e")
	fPats := flags.GetAll("f")

	quiet := flags.Has("q")
	suppressErrors := flags.Has("s")

	if len(ePats) > 0 || len(fPats) > 0 {
		paths = flags.Positional
	} else if len(flags.Positional) > 0 {
		ePats = append(ePats, flags.Positional[0])
		paths = flags.Positional[1:]
	}

	if len(ePats) == 0 && len(fPats) == 0 {
		if !suppressErrors {
			fmt.Fprintln(errOut, "grep: missing pattern")
		}
		return 2
	}

	jsonMode := flags.Has("json")
	invert := flags.Has("v")
	ignoreCase := flags.Has("i")
	countMode := flags.Has("c")
	lineNum := flags.Has("n")
	filesWithMatches := flags.Has("l")
	filesWithoutMatch := flags.Has("L")
	fixed := flags.Has("F")
	wordRegexp := flags.Has("w")
	lineRegexp := flags.Has("x")
	onlyMatching := flags.Has("o")
	recursive := flags.Has("r") || flags.Has("R")

	// Context flags
	var afterCtx, beforeCtx int
	if ctxStr := flags.Get("A"); ctxStr != "" {
		afterCtx, _ = strconv.Atoi(ctxStr)
		if afterCtx < 0 {
			afterCtx = 0
		}
	}
	if ctxStr := flags.Get("B"); ctxStr != "" {
		beforeCtx, _ = strconv.Atoi(ctxStr)
		if beforeCtx < 0 {
			beforeCtx = 0
		}
	}
	if ctxStr := flags.Get("C"); ctxStr != "" {
		n, _ := strconv.Atoi(ctxStr)
		if n < 0 {
			n = 0
		}
		afterCtx = n
		beforeCtx = n
	}
	hasContext := afterCtx > 0 || beforeCtx > 0

	for _, p := range ePats {
		for _, sp := range strings.Split(p, "\n") {
			if sp != "" {
				patterns = append(patterns, sp)
			}
		}
	}

	for _, fp := range fPats {
		var b []byte
		var err error
		if fp == "-" {
			b, err = io.ReadAll(stdinR)
		} else {
			b, err = os.ReadFile(fp)
		}

		if err == nil {
			for _, p := range strings.Split(strings.TrimSuffix(string(b), "\n"), "\n") {
				if p != "" {
					patterns = append(patterns, p)
				}
			}
		} else if !suppressErrors {
			fmt.Fprintf(errOut, "grep: %v\n", err)
			return 2
		}
	}

	var re *regexp.Regexp
	var fixedPatterns []string

	var pattern string
	if fixed {
		if ignoreCase || wordRegexp {
			var reParts []string
			for _, p := range patterns {
				escaped := regexp.QuoteMeta(p)
				if wordRegexp {
					escaped = "\\b" + escaped + "\\b"
				}
				reParts = append(reParts, escaped)
			}
			pattern = strings.Join(reParts, "|")
			if ignoreCase {
				pattern = "(?i)" + pattern
			}
			fixed = false
		} else {
			fixedPatterns = patterns
		}
	}

	if len(patterns) == 0 {
		// match nothing, re remains nil
	} else if !fixed {
		pattern = strings.Join(patterns, "|")
		if wordRegexp && !strings.Contains(pattern, "\\b") {
			if pattern == "^" || pattern == "$" {
				pattern = `a^`
			} else {
				pattern = "\\b(" + pattern + ")\\b"
			}
		}
		if lineRegexp {
			pattern = "^(" + pattern + ")$"
		}
		if ignoreCase && !strings.HasPrefix(pattern, "(?i)") {
			pattern = "(?i)" + pattern
		}

		compiled, err := regexp.Compile(pattern)
		if err != nil {
			if !suppressErrors {
				fmt.Fprintf(errOut, "grep: invalid regex: %v\n", err)
			}
			return 2
		}
		re = compiled
	}

	var readers []string
	var exitCode int = 1
	if len(paths) == 0 {
		readers = append(readers, "-")
	} else if recursive {
		for _, root := range paths {
			stat, err := os.Stat(root)
			if err != nil {
				if !suppressErrors {
					fmt.Fprintf(errOut, "grep: %s: %v\n", root, err)
				}
				exitCode = 2
				continue
			}
			if stat.IsDir() {
				walkRoot := root
				if info, err := os.Lstat(root); err == nil && info.Mode()&os.ModeSymlink != 0 {
					if !strings.HasSuffix(walkRoot, "/") {
						walkRoot += "/"
					}
				}
				filepath.Walk(walkRoot, func(p string, info os.FileInfo, err error) error {
					if err != nil {
						return nil
					}
					if !info.IsDir() {
						readers = append(readers, p)
					}
					return nil
				})
			} else {
				readers = append(readers, root)
			}
		}
	} else {
		readers = paths
	}

	var allMatches []GrepMatch

	// Determine filename prefix policy: -H forces prefix, -h suppresses it.
	// Without either, prefix is shown when multiple files are searched.
	// With -r (recursive), always show prefix (could expand to many files).
	printPrefix := flags.Has("H")
	noPrefix := flags.Has("h")
	if !noPrefix && !printPrefix {
		printPrefix = len(paths) > 1 || recursive
	}

	for _, path := range readers {
		retCode := func() int {
			var r io.Reader
			var fname string
			var f *os.File
			if path == "-" {
				r = stdinR
				fname = "(standard input)"
			} else {
				var err error
				f, err = os.Open(path)
				if err != nil {
					if !suppressErrors {
						fmt.Fprintf(errOut, "grep: %s: %v\n", path, err)
					}
					exitCode = 2
					return 0 // continue loop
				}
				defer f.Close()
				r = f
				fname = path
			}

			// 256MB total input cap to prevent OOM on large files
			r = io.LimitReader(r, 256*1024*1024)

			isBinary := false
			if !flags.Has("a") {
				// Read up to 8192 bytes to check for NUL bytes
				prefixBuf := make([]byte, 8192)
				n, readErr := io.ReadFull(r, prefixBuf)
				if readErr != nil && readErr != io.EOF && readErr != io.ErrUnexpectedEOF {
					if !suppressErrors {
						fmt.Fprintf(errOut, "grep: %v\n", readErr)
					}
					exitCode = 2
					return 0 // continue loop
				}
				prefix := prefixBuf[:n]
				for _, b := range prefix {
					if b == 0 {
						isBinary = true
						break
					}
				}
				// Reconstruct reader
				r = io.MultiReader(bytes.NewReader(prefix), r)
			}

			if isBinary {
				matches, err := Run(r, fname, re, fixedPatterns, invert, fixed, lineRegexp)
				if err != nil {
					if !suppressErrors {
						fmt.Fprintf(errOut, "grep: %v\n", err)
					}
					exitCode = 2
				}
				if len(matches) > 0 {
					if exitCode != 2 {
						exitCode = 0
					}
					if quiet {
						return 1 // return 0 from grepRun
					}
					if jsonMode {
						allMatches = append(allMatches, GrepMatch{
							File: fname,
							Text: fmt.Sprintf("Binary file %s matches", fname),
						})
					} else {
						if filesWithMatches {
							fmt.Fprintln(stdout, fname)
						} else if !filesWithoutMatch {
							fmt.Fprintf(stdout, "Binary file %s matches\n", fname)
						}
					}
				} else {
					if filesWithoutMatch && !jsonMode {
						fmt.Fprintln(stdout, fname)
					}
				}
				return 0 // continue loop
			}

			// Context mode: scan with context instead of using Run().
			if hasContext && !countMode && !filesWithMatches && !filesWithoutMatch && !onlyMatching {
				ctxMatches := scanWithContext(r, re, fixedPatterns, invert, fixed, lineRegexp, beforeCtx, afterCtx)
				if jsonMode {
					for _, cm := range ctxMatches {
						allMatches = append(allMatches, GrepMatch{
							File: fname,
							Line: cm.line,
							Text: cm.text,
						})
					}
					if len(ctxMatches) > 0 {
						if exitCode != 2 {
							exitCode = 0
						}
						if quiet {
							return 1 // return 0 from grepRun
						}
					}
					return 0 // continue loop
				}
				if len(ctxMatches) > 0 {
					if exitCode != 2 {
						exitCode = 0
					}
					if quiet {
						return 1 // return 0 from grepRun
					}
				}
				for _, cm := range ctxMatches {
					prefix := ""
					if printPrefix {
						prefix = fname + ":"
					}
					if lineNum {
						prefix += fmt.Sprintf("%d:", cm.line)
					}
					if cm.separator {
						fmt.Fprintln(stdout, "--")
					}
					fmt.Fprintf(stdout, "%s%s%s\n", prefix, cm.matchMarker, cm.text)
				}
				return 0 // continue loop
			}

			matches, err := Run(r, fname, re, fixedPatterns, invert, fixed, lineRegexp)
			if err != nil {
				if !suppressErrors {
					fmt.Fprintf(errOut, "grep: %v\n", err)
				}
				exitCode = 2
			}

			if jsonMode {
				allMatches = append(allMatches, matches...)
				if len(matches) > 0 {
					if exitCode != 2 {
						exitCode = 0
					}
					if quiet {
						return 1 // return 0 from grepRun
					}
				}
				return 0 // continue loop
			}

			if filesWithMatches {
				if len(matches) > 0 {
					fmt.Fprintln(stdout, fname)
					if exitCode != 2 {
						exitCode = 0
					}
					if quiet {
						return 1 // return 0 from grepRun
					}
				}
				return 0 // continue loop
			}

			if filesWithoutMatch {
				if len(matches) == 0 {
					fmt.Fprintln(stdout, fname)
					if exitCode != 2 {
						exitCode = 0
					}
					if quiet {
						return 1 // return 0 from grepRun
					}
				}
				return 0 // continue loop
			}

			if len(matches) > 0 {
				if exitCode != 2 {
					exitCode = 0
				}
				if quiet {
					return 1 // return 0 from grepRun
				}
			}

			if countMode {
				prefix := ""
				if printPrefix {
					prefix = fname + ":"
				}
				fmt.Fprintf(stdout, "%s%d\n", prefix, len(matches))
				return 0 // continue loop
			}

			for _, m := range matches {
				prefix := ""
				if printPrefix {
					prefix = fname + ":"
				}
				if lineNum {
					prefix += fmt.Sprintf("%d:", m.Line)
				}
				if onlyMatching {
					for _, sub := range m.Matches {
						fmt.Fprintf(stdout, "%s%s\n", prefix, sub)
					}
				} else {
					fmt.Fprintf(stdout, "%s%s\n", prefix, m.Text)
				}
			}
			return 0
		}()
		if retCode == 1 {
			return 0
		}
	}

	if jsonMode {
		common.Render("grep", allMatches, true, stdout, func() {})
	}

	return exitCode
}

// ctxLine holds a line from the context-aware scanner.
type ctxLine struct {
	line        int
	text        string
	matchMarker string // ":" for match lines, "-" for context lines
	separator   bool   // true if this is a group separator
}

// scanWithContext scans r and returns matches with surrounding context lines.
// Groups of matches separated by more than (afterCtx+beforeCtx) lines get a "--" separator.
func scanWithContext(r io.Reader, re *regexp.Regexp, fixedPatterns []string, invert, fixed, lineRegexp bool, beforeCtx, afterCtx int) []ctxLine {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024)
	var allLines []string
	for scanner.Scan() {
		allLines = append(allLines, scanner.Text())
	}

	n := len(allLines)
	if n == 0 {
		return nil
	}

	// Determine which lines match.
	matched := make([]bool, n)
	for i, text := range allLines {
		var found bool
		if fixed {
			for _, pat := range fixedPatterns {
				if lineRegexp {
					found = text == pat
				} else {
					found = strings.Contains(text, pat)
				}
				if found {
					break
				}
			}
		} else if re != nil {
			if lineRegexp {
				found = re.MatchString(text)
			} else {
				found = re.MatchString(text)
			}
		}
		if invert {
			found = !found
		}
		matched[i] = found
	}

	// Build the set of lines to print (match + context).
	printLine := make([]bool, n)
	for i := 0; i < n; i++ {
		if matched[i] {
			printLine[i] = true
			for j := 1; j <= beforeCtx && i-j >= 0; j++ {
				printLine[i-j] = true
			}
			for j := 1; j <= afterCtx && i+j < n && !matched[i+j]; j++ {
				printLine[i+j] = true
			}
		}
	}

	// Build output with separators between non-contiguous groups.
	var result []ctxLine
	lastPrinted := -999
	for i := 0; i < n; i++ {
		if !printLine[i] {
			continue
		}
		// Insert separator if there's a gap.
		if len(result) > 0 && i-lastPrinted > 1 {
			result = append(result, ctxLine{separator: true})
		}
		marker := "-"
		if matched[i] {
			marker = ":"
		}
		result = append(result, ctxLine{
			line:        i + 1,
			text:        allLines[i],
			matchMarker: marker,
		})
		lastPrinted = i
	}

	return result
}

func init() {
	dispatch.Register(dispatch.Command{Name: "grep", Usage: "Print lines matching a pattern", Run: run})
	dispatch.Register(dispatch.Command{Name: "egrep", Usage: "Print lines matching an extended regex pattern", Run: egrepRun})
	dispatch.Register(dispatch.Command{Name: "fgrep", Usage: "Print lines matching fixed strings", Run: fgrepRun})
}

// egrepRun prepends -E and delegates to the main grep CLI.
func egrepRun(args []string, stdout io.Writer) int {
	return run(append([]string{"-E"}, args...), stdout)
}

// fgrepRun prepends -F and delegates to the main grep CLI.
func fgrepRun(args []string, stdout io.Writer) int {
	return run(append([]string{"-F"}, args...), stdout)
}
