// Package wc implements the POSIX wc utility.
package wc

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"unicode"
	"unicode/utf8"

	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
)

// WcResult is the --json output for a file.
type WcResult struct {
	Lines         int `json:"lines"`
	Words         int `json:"words"`
	Bytes         int `json:"bytes"`
	Chars         int `json:"chars"`
	MaxLineLength int `json:"maxLineLength"`
}

var spec = common.FlagSpec{
	Defs: []common.FlagDef{
		{Short: "l", Long: "lines", Type: common.FlagBool},
		{Short: "w", Long: "words", Type: common.FlagBool},
		{Short: "c", Long: "bytes", Type: common.FlagBool},
		{Short: "m", Long: "chars", Type: common.FlagBool},
		{Short: "L", Long: "max-line-length", Type: common.FlagBool},
		{Long: "json", Type: common.FlagBool},
	},
}

// Count reads from r and returns the counts.
func Count(r io.Reader) (WcResult, error) {
	var res WcResult
	buf := make([]byte, 32*1024)
	inWord := false

	for {
		n, err := r.Read(buf)
		if n > 0 {
			res.Bytes += n
			chunk := buf[:n]

			res.Lines += bytes.Count(chunk, []byte{'\n'})

			res.Chars += utf8.RuneCount(chunk) // Simplified, actual needs state between chunks

			for _, r := range string(chunk) {
				if unicode.IsSpace(r) {
					inWord = false
				} else if !inWord {
					inWord = true
					res.Words++
				}
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return res, err
		}
	}
	return res, nil
}

// Simple reliable counter using bufio.Scanner for words and lines, but it might be slower.
func CountScanner(r io.Reader) (WcResult, error) {
	var res WcResult
	scanner := bufio.NewScanner(r)
	scanner.Split(bufio.ScanBytes) // We process byte by byte for accurate counts

	inWord := false

	for scanner.Scan() {
		b := scanner.Bytes()[0]
		res.Bytes++
		if b == '\n' {
			res.Lines++
		}

		r := rune(b) // This is naïve for chars, assumes ASCII for the word splitting part. Proper implementation uses a better reader.
		if unicode.IsSpace(r) {
			inWord = false
		} else if !inWord {
			inWord = true
			res.Words++
		}
	}

	// Real char count needs proper rune parsing
	// Since we need posix wc, let's use bufio.Reader
	return res, scanner.Err()
}

// Proper count implementation
func CountProper(r io.Reader) (WcResult, error) {
	var res WcResult
	var reader *bufio.Reader
	if br, ok := r.(*bufio.Reader); ok {
		reader = br
	} else {
		reader = bufio.NewReaderSize(r, 64*1024)
	}
	inWord := false
	curLineLen := 0

	for {
		buf, err := reader.Peek(64 * 1024)
		n := len(buf)
		if n == 0 {
			if err != nil {
				if err == io.EOF {
					break
				}
				return res, err
			}
			continue
		}

		i := 0
		for i < n {
			b := buf[i]
			if b < 0x80 {
				// ASCII fast path
				res.Bytes++
				res.Chars++
				i++

				if b == '\n' {
					res.Lines++
					if curLineLen > res.MaxLineLength {
						res.MaxLineLength = curLineLen
					}
					curLineLen = 0
					inWord = false
					continue
				}

				curLineLen++
				isSpace := b == ' ' || b == '\t' || b == '\r' || b == '\f' || b == '\v'
				if isSpace {
					inWord = false
				} else if !inWord {
					inWord = true
					res.Words++
				}
			} else {
				// UTF-8 slow path
				if n-i < 4 && err != io.EOF {
					break
				}

				rn, size := utf8.DecodeRune(buf[i:])
				res.Bytes += size
				res.Chars++
				i += size

				if rn == '\n' {
					res.Lines++
					if curLineLen > res.MaxLineLength {
						res.MaxLineLength = curLineLen
					}
					curLineLen = 0
					inWord = false
					continue
				}

				curLineLen++
				if unicode.IsSpace(rn) {
					inWord = false
				} else if !inWord {
					inWord = true
					res.Words++
				}
			}
		}

		_, _ = reader.Discard(i)
	}

	// Check last line's length (file may not end with newline)
	if curLineLen > res.MaxLineLength {
		res.MaxLineLength = curLineLen
	}

	return res, nil
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer, cwd string) int {
	flags, err := common.ParseFlags(args, spec)
	if err != nil {
		fmt.Fprintf(stderr, "wc: %v\n", err)
		return 2
	}
	jsonMode := flags.Has("json")

	showLines := flags.Has("l")
	showWords := flags.Has("w")
	showBytes := flags.Has("c")
	showChars := flags.Has("m")
	showMaxLine := flags.Has("L")

	// Default POSIX behavior: if no flags are given, print lines, words, bytes
	if !showLines && !showWords && !showBytes && !showChars && !showMaxLine {
		showLines, showWords, showBytes = true, true, true
	}

	paths := flags.Positional
	if len(paths) == 0 {
		paths = append(paths, "-")
	}

	var total WcResult
	var jsonResults map[string]WcResult
	if jsonMode {
		jsonResults = make(map[string]WcResult)
	}

	exitCode := 0
	for _, p := range paths {
		var f io.Reader
		if p == "-" {
			f = stdin
		} else {
			file, err := os.Open(p)
			if err != nil {
				fmt.Fprintf(stderr, "wc: %s: %v\n", p, err)
				exitCode = 1
				continue
			}
			defer file.Close()
			f = file
		}

		res, err := CountProper(f)
		if err != nil {
			fmt.Fprintf(stderr, "wc: %s: %v\n", p, err)
			exitCode = 1
			continue
		}

		total.Lines += res.Lines
		total.Words += res.Words
		total.Bytes += res.Bytes
		total.Chars += res.Chars

		if jsonMode {
			jsonResults[p] = res
		} else {
			printCount(res, p, showLines, showWords, showBytes, showChars, showMaxLine, stdout)
		}
	}

	if len(paths) > 1 {
		if jsonMode {
			jsonResults["total"] = total
		} else {
			printCount(total, "total", showLines, showWords, showBytes, showChars, showMaxLine, stdout)
		}
	}

	if jsonMode {
		// Output json results
		// If single file, unwrap
		if len(paths) == 1 {
			common.Render("wc", jsonResults[paths[0]], true, stdout, func() {})
		} else {
			common.Render("wc", jsonResults, true, stdout, func() {})
		}
	}

	return exitCode
}

func printCount(res WcResult, name string, showLines, showWords, showBytes, showChars, showMaxLine bool, stdout io.Writer) {
	line := ""
	if showMaxLine {
		line += fmt.Sprintf(" %d", res.MaxLineLength)
	}
	if showLines {
		line += fmt.Sprintf(" %d", res.Lines)
	}
	if showWords {
		line += fmt.Sprintf(" %d", res.Words)
	}
	if showChars {
		line += fmt.Sprintf(" %d", res.Chars)
	} else if showBytes {
		line += fmt.Sprintf(" %d", res.Bytes)
	}
	if name != "-" {
		line += fmt.Sprintf(" %s", name)
	}
	if line != "" {
		fmt.Fprintln(stdout, line[1:]) // trim leading space
	}
}

func init() {
	dispatch.Register(dispatch.Command{Name: "wc", Usage: "Print newline, word, and byte counts", Run: run})
}
