// Package rev implements the POSIX rev utility.
package rev

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
)

// RevResult holds the list of reversed lines for JSON rendering.
type RevResult struct {
	Lines     []string `json:"lines"`
	LineCount int      `json:"line_count"`
}

var spec = common.FlagSpec{
	Defs: []common.FlagDef{
		{Long: "json", Type: common.FlagBool},
	},
}

func reverseRunes(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer, cwd string) int {
	if stdin == nil {
		stdin = os.Stdin
	}
	flags, err := common.ParseFlags(args, spec)
	if err != nil {
		fmt.Fprintf(os.Stderr, "rev: %v\n", err)
		return 1
	}

	jsonMode := flags.Has("json")
	files := flags.Positional
	if len(files) == 0 {
		files = []string{"-"}
	}

	var allLines []string
	exitCode := 0

	for _, file := range files {
		var r io.Reader
		if file == "-" {
			r = stdin
		} else {
			f, err := os.Open(file)
			if err != nil {
				fmt.Fprintf(os.Stderr, "rev: %s: %v\n", file, err)
				exitCode = 1
				continue
			}
			r = f
		}

		br := bufio.NewReader(r)
		for {
			lineBytes, err := br.ReadBytes('\n')
			if len(lineBytes) > 0 {
				cBytes := lineBytes
				if idx := strings.IndexByte(string(lineBytes), 0); idx != -1 {
					cBytes = lineBytes[:idx]
				}

				hasNL := false
				if len(cBytes) > 0 && cBytes[len(cBytes)-1] == '\n' {
					hasNL = true
					cBytes = cBytes[:len(cBytes)-1]
					if len(cBytes) > 0 && cBytes[len(cBytes)-1] == '\r' {
						cBytes = cBytes[:len(cBytes)-1]
					}
				}

				revStr := reverseRunes(string(cBytes))
				allLines = append(allLines, revStr)

				if !jsonMode {
					if hasNL {
						fmt.Fprintln(stdout, revStr)
					} else {
						fmt.Fprint(stdout, revStr)
					}
				}
			}
			if err != nil {
				if err != io.EOF {
					fmt.Fprintf(os.Stderr, "rev: %v\n", err)
					exitCode = 1
				}
				break
			}
		}

		if file != "-" {
			if closer, ok := r.(io.Closer); ok {
				closer.Close()
			}
		}
	}

	if jsonMode {
		common.Render("rev", RevResult{Lines: allLines, LineCount: len(allLines)}, true, stdout, func() {})
	}

	return exitCode
}

func init() {
	dispatch.Register(dispatch.Command{
		Name:  "rev",
		Usage: "Reverse lines of a file or files",
		Run:   run,
	})
}
