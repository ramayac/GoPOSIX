// Package head implements the POSIX head utility.
package head

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
)

// HeadResult is the --json output for head.
type HeadResult struct {
	Lines     []string `json:"lines"`
	LineCount int      `json:"lineCount"`
}

var spec = common.FlagSpec{
	Defs: []common.FlagDef{
		{Short: "n", Long: "lines", Type: common.FlagValue},
		{Short: "c", Long: "bytes", Type: common.FlagValue},
		{Long: "json", Type: common.FlagBool},
	},
}

// Run reads up to linesCount lines from r and writes to w.
// Returns the read lines for JSON mode.
func Run(r io.Reader, w io.Writer, linesCount int) ([]string, error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024)
	var lines []string
	count := 0

	for scanner.Scan() {
		if count >= linesCount {
			break
		}
		line := scanner.Text()
		fmt.Fprintln(w, line)
		lines = append(lines, line)
		count++
	}
	return lines, scanner.Err()
}

// headInput pairs a reader with its display name for multi-file headers.
type headInput struct {
	r      io.Reader
	name   string
	closer io.Closer // non-nil for files that need closing
}

func run(args []string, out io.Writer) int {
	flags, err := common.ParseFlags(args, spec)
	if err != nil {
		fmt.Fprintf(os.Stderr, "head: %v\n", err)
		return 2
	}
	jsonMode := flags.Has("json")
	linesCount := 10
	byteCount := -1
	negativeCount := false
	if cStr := flags.Get("c"); cStr != "" {
		n, err := strconv.Atoi(cStr)
		if err != nil || n < 0 {
			fmt.Fprintf(os.Stderr, "head: illegal byte count -- %s\n", cStr)
			return 2
		}
		byteCount = n
		linesCount = 0 // -c overrides -n
	} else if nStr := flags.Get("n"); nStr != "" {
		if strings.HasPrefix(nStr, "-") {
			n, err := strconv.Atoi(nStr[1:])
			if err != nil {
				fmt.Fprintf(os.Stderr, "head: illegal line count -- %s\n", nStr)
				return 2
			}
			linesCount = n
			negativeCount = true
		} else {
			n, err := strconv.Atoi(nStr)
			if err != nil || n < 0 {
				fmt.Fprintf(os.Stderr, "head: illegal line count -- %s\n", nStr)
				return 2
			}
			linesCount = n
		}
	}

	// Build list of (reader, displayName) pairs.
	// This correctly handles bare "-" (stdin) interleaved with file paths.
	// Note: flags.Stdin is true when bare "-" appears, AND "-" is added to Positional.
	// We must avoid adding stdin twice.
	exitCode := 0
	var inputs []headInput
	// Only add stdin from the no-args / Stdin case if Positional is empty or
	// the first positional is NOT "-" (which would duplicate).
	if len(flags.Positional) == 0 {
		inputs = append(inputs, headInput{r: os.Stdin, name: "standard input"})
	}
	for _, path := range flags.Positional {
		if path == "-" {
			inputs = append(inputs, headInput{r: os.Stdin, name: "standard input"})
			continue
		}
		f, err := os.Open(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "head: %s: %v\n", path, err)
			exitCode = 1
			continue
		}
		inputs = append(inputs, headInput{r: f, name: path, closer: f})
	}

	var allLines []string
	for i, in := range inputs {
		if len(inputs) > 1 {
			if !jsonMode {
				if i > 0 {
					fmt.Println()
				}
				fmt.Printf("==> %s <==\n", in.name)
			}
		}

		var w io.Writer = os.Stdout
		if jsonMode {
			w = io.Discard
		}

		var lines []string
		var errR error
		if byteCount >= 0 {
			lines, errR = runBytes(in.r, w, byteCount)
		} else if negativeCount {
			lines, errR = runNegative(in.r, w, linesCount)
		} else {
			lines, errR = Run(in.r, w, linesCount)
		}
		if errR != nil {
			fmt.Fprintf(os.Stderr, "head: %v\n", errR)
			exitCode = 1
		}
		if jsonMode {
			allLines = append(allLines, lines...)
		}
		// Close per-file handles immediately to avoid fd exhaustion.
		if in.closer != nil {
			in.closer.Close()
		}
	}

	if jsonMode {
		common.Render("head", HeadResult{Lines: allLines, LineCount: len(allLines)}, true, out, func() {})
	}

	return exitCode
}

// runNegative prints all lines except the last skipLast lines.
func runNegative(r io.Reader, w io.Writer, skipLast int) ([]string, error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024)
	var all []string
	for scanner.Scan() {
		all = append(all, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return all, err
	}
	limit := len(all) - skipLast
	if limit < 0 {
		limit = 0
	}
	var lines []string
	for i := 0; i < limit; i++ {
		fmt.Fprintln(w, all[i])
		lines = append(lines, all[i])
	}
	return lines, nil
}

// runBytes reads up to n bytes from r and writes to w.
func runBytes(r io.Reader, w io.Writer, n int) ([]string, error) {
	buf := make([]byte, n)
	total, err := io.ReadFull(r, buf)
	if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
		return nil, err
	}
	if total > 0 {
		w.Write(buf[:total])
	}
	return nil, nil
}

func init() {
	dispatch.Register(dispatch.Command{Name: "head", Usage: "Output the first part of files", Run: run})
}
