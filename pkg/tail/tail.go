// Package tail implements the POSIX tail utility.
package tail

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
)

// TailResult is the --json output for tail.
type TailResult struct {
	Lines     []string `json:"lines"`
	LineCount int      `json:"lineCount"`
}

var spec = common.FlagSpec{
	Defs: []common.FlagDef{
		{Short: "n", Long: "lines", Type: common.FlagValue},
		{Short: "c", Long: "bytes", Type: common.FlagValue},
		{Short: "f", Long: "follow", Type: common.FlagBool},
		{Long: "json", Type: common.FlagBool},
	},
}

func Run(r io.Reader, w io.Writer, linesCount int, bytesCount int, fromStart bool) ([]string, error) {
	if bytesCount > 0 {
		data, err := io.ReadAll(r)
		if err != nil {
			return nil, err
		}
		if fromStart {
			skip := bytesCount - 1
			if skip < 0 {
				skip = 0
			}
			if skip > len(data) {
				skip = len(data)
			}
			data = data[skip:]
		} else {
			if bytesCount > len(data) {
				bytesCount = len(data)
			}
			data = data[len(data)-bytesCount:]
		}
		w.Write(data)
		return []string{string(data)}, nil
	}

	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024)
	var lines []string

	if fromStart {
		skip := linesCount - 1
		if skip < 0 {
			skip = 0
		}
		lineNum := 0
		for scanner.Scan() {
			if lineNum >= skip {
				lines = append(lines, scanner.Text())
				fmt.Fprintln(w, scanner.Text())
			}
			lineNum++
		}
	} else {
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
			if len(lines) > linesCount {
				lines = lines[1:]
			}
		}

		for _, line := range lines {
			fmt.Fprintln(w, line)
		}
	}

	return lines, scanner.Err()
}

func run(args []string, stdin io.Reader, stdout io.Writer) int {
	// Preprocess: convert traditional "-N" (where N is a number) to "-n N"
	cleanArgs := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		a := args[i]
		if len(a) >= 2 && a[0] == '-' && a[1] >= '0' && a[1] <= '9' {
			// -N → -n N
			cleanArgs = append(cleanArgs, "-n", a[1:])
		} else {
			cleanArgs = append(cleanArgs, a)
		}
	}
	args = cleanArgs

	flags, err := common.ParseFlags(args, spec)
	if err != nil {
		fmt.Fprintf(os.Stderr, "tail: %v\n", err)
		return 2
	}
	jsonMode := flags.Has("json")
	follow := flags.Has("f")
	linesCount := 10
	bytesCount := 0
	fromStart := false

	nStr := flags.Get("n")
	cStr := flags.Get("c")

	if len(flags.Positional) > 0 && strings.HasPrefix(flags.Positional[0], "+") {
		nStr = flags.Positional[0]
		flags.Positional = flags.Positional[1:]
	}

	if cStr != "" {
		if strings.HasPrefix(cStr, "+") {
			fromStart = true
			cStr = cStr[1:]
		}
		n, err := strconv.Atoi(cStr)
		if err != nil || n < 0 {
			fmt.Fprintf(os.Stderr, "tail: illegal byte count -- %s\n", cStr)
			return 2
		}
		bytesCount = n
	} else if nStr != "" {
		if strings.HasPrefix(nStr, "+") {
			fromStart = true
			nStr = nStr[1:]
		}
		n, err := strconv.Atoi(nStr)
		if err != nil || n < 0 {
			fmt.Fprintf(os.Stderr, "tail: illegal line count -- %s\n", nStr)
			return 2
		}
		linesCount = n
	}

	var readers []string
	if len(flags.Positional) == 0 {
		readers = append(readers, "-")
	} else {
		readers = flags.Positional
	}

	exitCode := 0
	var allLines []string
	for i, path := range readers {
		if len(readers) > 1 {
			name := path
			if path == "-" {
				name = "standard input"
			}
			header := fmt.Sprintf("==> %s <==", name)
			if !jsonMode {
				if i > 0 {
					fmt.Fprintln(stdout)
				}
				fmt.Fprintln(stdout, header)
			}
		}

		var f *os.File
		if path == "-" {
			f = os.Stdin
		} else {
			file, err := os.Open(path)
			if err != nil {
				fmt.Fprintf(os.Stderr, "tail: %s: %v\n", path, err)
				exitCode = 1
				continue
			}
			f = file
		}

		writer := stdout
		if jsonMode {
			writer = io.Discard
		}

		var w io.Writer = writer
		if jsonMode {
			w = io.Discard
		}

		lines, err := Run(f, w, linesCount, bytesCount, fromStart)
		if err != nil {
			fmt.Fprintf(os.Stderr, "tail: %v\n", err)
			exitCode = 1
		}
		if jsonMode {
			allLines = append(allLines, lines...)
		}

		if follow && path != "-" {
			// Naive polling follow for non-stdin
			for {
				time.Sleep(500 * time.Millisecond)
				scanner := bufio.NewScanner(f)
				for scanner.Scan() {
					fmt.Fprintln(writer, scanner.Text())
				}
			}
		}

		if path != "-" {
			f.Close()
		}
	}

	if jsonMode {
		common.Render("tail", TailResult{Lines: allLines, LineCount: len(allLines)}, true, stdout, func() {})
	}

	return exitCode
}

func init() {
	dispatch.Register(dispatch.Command{Name: "tail", Usage: "Output the last part of files", Run: run})
}
