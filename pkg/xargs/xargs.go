package xargs

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
)

var spec = common.FlagSpec{
	Defs: []common.FlagDef{
		{Short: "E", Long: "eof-str", Type: common.FlagValue},
		{Short: "e", Long: "eof-str-compat", Type: common.FlagOptionalValue},
		{Short: "s", Long: "max-chars", Type: common.FlagValue},
		{Short: "n", Long: "max-args", Type: common.FlagValue},
		{Short: "I", Long: "replace", Type: common.FlagValue},
		{Short: "t", Long: "verbose", Type: common.FlagBool},
		{Long: "json", Type: common.FlagBool},
		{Short: "0", Long: "null", Type: common.FlagBool},
	},
}

type ExecResult struct {
	Command  string `json:"command"`
	ExitCode int    `json:"exitCode"`
}

func run(args []string, out io.Writer) int {
	return xargsRun(args, out, os.Stderr, os.Stdin)
}

func xargsRun(args []string, out io.Writer, errOut io.Writer, stdin io.Reader) int {
	flags, err := common.ParseFlags(args, spec)
	if err != nil {
		fmt.Fprintf(errOut, "xargs: %v\n", err)
		return 1
	}

	baseCmd := "echo"
	if len(flags.Positional) > 0 {
		baseCmd = flags.Positional[0]
	}

	cmdArgs := []string{}
	if len(flags.Positional) > 1 {
		cmdArgs = flags.Positional[1:]
	}

	maxSize := 2048 // default max command line size
	if val := flags.Get("s"); val != "" {
		fmt.Sscanf(val, "%d", &maxSize)
	}

	maxArgs := 0
	if val := flags.Get("n"); val != "" {
		fmt.Sscanf(val, "%d", &maxArgs)
	}

	trace := flags.Has("t")

	replaceStr := flags.Get("I")

	eofStr := ""
	hasEOF := false
	if flags.Has("E") {
		eofStr = flags.Get("E")
		hasEOF = eofStr != ""
	} else if flags.Has("e") {
		eofStr = flags.Get("e")
		hasEOF = eofStr != ""
	}

	baseSize := len(baseCmd) + 1
	for _, a := range cmdArgs {
		baseSize += len(a) + 1
	}

	nullDelim := flags.Has("0")

	scanner := bufio.NewScanner(stdin)
	// When -I is used, read entire lines (not words).
	// When -0 is used, split on NUL bytes.
	if nullDelim {
		scanner.Split(scanNulls)
	} else if replaceStr != "" {
		scanner.Split(bufio.ScanLines)
	} else {
		scanner.Split(bufio.ScanWords)
	}

	var batches [][]string
	var currentBatch []string
	currentSize := baseSize

	for scanner.Scan() {
		word := scanner.Text()
		if hasEOF && word == eofStr {
			break
		}

		// -I replace-str: replace occurrences of replaceStr in cmdArgs with word.
		// Empty lines and whitespace-only lines are SKIPPED in -I mode.
		// Leading whitespace is stripped from each line.
		if replaceStr != "" {
			word = strings.TrimLeft(word, " \t\n\r\v\f")
			if word == "" {
				continue
			}
			replacedArgs := make([]string, len(cmdArgs))
			for i, a := range cmdArgs {
				replacedArgs[i] = strings.ReplaceAll(a, replaceStr, word)
			}
			batches = append(batches, replacedArgs)
			continue
		}

		sizeLimitHit := maxSize > 0 && currentSize+len(word)+1 > maxSize && len(currentBatch) > 0
		argsLimitHit := maxArgs > 0 && len(currentBatch) >= maxArgs

		if sizeLimitHit || argsLimitHit {
			batches = append(batches, currentBatch)
			currentBatch = nil
			currentSize = baseSize
		}
		currentBatch = append(currentBatch, word)
		currentSize += len(word) + 1
	}

	if len(currentBatch) > 0 {
		batches = append(batches, currentBatch)
	}
	if len(batches) == 0 {
		batches = append(batches, []string{}) // xargs runs once if no args
	}

	exitCode := 0
	var results []ExecResult

	for _, batch := range batches {
		var args []string
		if replaceStr != "" {
			// In -I mode, batch already contains the full replaced args.
			args = batch
		} else {
			args = append([]string{}, cmdArgs...)
			args = append(args, batch...)
		}

		cmd := exec.Command(baseCmd, args...)
		cmd.Stdout = out
		cmd.Stderr = errOut

		if trace {
			traceStr := baseCmd
			for _, a := range args {
				traceStr += " " + a
			}
			fmt.Fprintln(errOut, traceStr)
		}

		err = cmd.Run()
		code := 0
		if err != nil {
			code = 123
			if exitError, ok := err.(*exec.ExitError); ok {
				code = exitError.ExitCode()
			}
			exitCode = 123 // POSIX says exit 123 if any invocation returns 1-125
		}

		results = append(results, ExecResult{
			Command:  baseCmd,
			ExitCode: code,
		})

		// POSIX: if command exits with 255, xargs stops immediately
		if code == 255 {
			exitCode = 124
			break
		}
	}

	if flags.Has("json") {
		common.Render("xargs", results, true, out, func() {})
	}

	return exitCode
}

// scanNulls is a bufio.SplitFunc that splits on NUL (\0) bytes.
func scanNulls(data []byte, atEOF bool) (advance int, token []byte, err error) {
	for i, b := range data {
		if b == 0 {
			return i + 1, data[:i], nil
		}
	}
	if atEOF && len(data) > 0 {
		return len(data), data, nil
	}
	return 0, nil, nil
}

func init() {
	dispatch.Register(dispatch.Command{Name: "xargs", Usage: "Build and execute command lines from standard input", Run: run})
}
