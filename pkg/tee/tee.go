// Package tee implements the POSIX tee utility.
package tee

import (
	"fmt"
	"io"
	"os"

	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
)

// TeeResult is the structured result for --json mode.
type TeeResult struct {
	BytesWritten int64    `json:"bytesWritten"`
	Files        []string `json:"files"`
}

var spec = common.FlagSpec{
	Defs: []common.FlagDef{
		{Short: "a", Long: "append", Type: common.FlagBool},
		{Long: "json", Type: common.FlagBool},
	},
}

// countingWriter wraps an io.Writer and counts bytes written.
type countingWriter struct {
	w     io.Writer
	count int64
}

func (c *countingWriter) Write(p []byte) (int, error) {
	n, err := c.w.Write(p)
	c.count += int64(n)
	return n, err
}

func run(args []string, stdin io.Reader, stdout io.Writer) int {
	return teeRun(args, stdout, os.Stderr, os.Stdin)
}

func teeRun(args []string, stdout io.Writer, errOut io.Writer, stdin io.Reader) int {
	flags, err := common.ParseFlags(args, spec)
	if err != nil {
		fmt.Fprintf(errOut, "tee: %v\n", err)
		return 2
	}
	appendMode := flags.Has("a")
	jsonMode := flags.Has("json")

	var writers []io.Writer
	var filePaths []string

	// In json mode, discard stdout output (captured in result)
	var stdoutCapture io.Writer
	if jsonMode {
		stdoutCapture = io.Discard
	} else {
		stdoutCapture = stdout
	}

	writers = append(writers, stdoutCapture)

	var files []*os.File
	exitCode := 0

	for _, path := range flags.Positional {
		fileFlags := os.O_WRONLY | os.O_CREATE
		if appendMode {
			fileFlags |= os.O_APPEND
		} else {
			fileFlags |= os.O_TRUNC
		}
		f, err := os.OpenFile(path, fileFlags, 0666)
		if err != nil {
			if jsonMode {
				common.RenderError("tee", 1, "OPEN", fmt.Sprintf("%s: %v", path, err), true, stdout)
			} else {
				fmt.Fprintf(errOut, "tee: %s: %v\n", path, err)
			}
			exitCode = 1
			continue
		}
		files = append(files, f)
		filePaths = append(filePaths, path)
		writers = append(writers, f)
	}

	multiWriter := io.MultiWriter(writers...)

	var totalWritten int64
	if jsonMode {
		cw := &countingWriter{w: multiWriter}
		_, err = io.Copy(cw, stdin)
		totalWritten = cw.count
	} else {
		_, err = io.Copy(multiWriter, stdin)
	}

	if err != nil {
		if jsonMode {
			common.RenderError("tee", 1, "WRITE", err.Error(), true, stdout)
		} else {
			fmt.Fprintf(errOut, "tee: %v\n", err)
		}
		exitCode = 1
	}

	for _, f := range files {
		f.Close()
	}

	if jsonMode && exitCode == 0 {
		common.Render("tee", TeeResult{
			BytesWritten: totalWritten,
			Files:        filePaths,
		}, true, stdout, func() {})
	}

	return exitCode
}

func init() {
	dispatch.Register(dispatch.Command{
		Name:           "tee",
		Usage:          "Read from standard input and write to standard output and files",
		Run:            run,
		RunWithStreams: teeRun,
	})
}
