// Package bzcat implements the POSIX-compliant bzcat utility.
package bzcat

import (
	"compress/bzip2"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
)

var spec = common.FlagSpec{
	Defs: []common.FlagDef{
		{Short: "h", Long: "help", Type: common.FlagBool},
		{Long: "json", Type: common.FlagBool},
	},
}

func init() {
	dispatch.Register(dispatch.Command{
		Name:  "bzcat",
		Usage: "Decompress bzip2 files to standard output",
		Run:   run,
	})
}

// ExtractedFileInfo contains information about a single extracted file for JSON output.
type ExtractedFileInfo struct {
	Source      string `json:"source"`
	BytesResult int64  `json:"bytesResult"`
	Error       string `json:"error,omitempty"`
}

// BzcatResult represents the JSON output format.
type BzcatResult struct {
	Files []ExtractedFileInfo `json:"files"`
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer, cwd string) int {
	jsonMode := false
	for _, arg := range args {
		if arg == "--json" {
			jsonMode = true
			break
		}
	}

	flags, err := common.ParseFlags(args, spec)
	if err != nil {
		if jsonMode {
			common.RenderError("bzcat", 1, "FLAG_ERROR", err.Error(), true, stderr)
		} else {
			fmt.Fprintf(stderr, "bzcat: %v\n", err)
		}
		return 1
	}

	if flags.Has("h") || flags.Has("help") {
		helpText := "Usage: bzcat [FILE]...\n\n" +
			"Decompress FILEs to standard output.\n\n" +
			"Options:\n" +
			"  -h, --help     Print help"
		common.Render("bzcat", struct {
			Help string `json:"help"`
		}{Help: helpText}, jsonMode, stdout, func() {
			fmt.Fprintln(stdout, helpText)
		})
		return 0
	}

	files := flags.Positional

	// Default: stdin to stdout
	if len(files) == 0 || (len(files) == 1 && files[0] == "-") {
		bzReader := bzip2.NewReader(stdin)
		written, err := io.Copy(stdout, bzReader)
		if err != nil {
			fmt.Fprintf(stderr, "bzcat: corrupted data\n")
			if jsonMode {
				common.RenderError("bzcat", 1, "DECOMPRESS_ERROR", err.Error(), true, stderr)
			}
			return 1
		}
		if jsonMode {
			common.Render("bzcat", BzcatResult{
				Files: []ExtractedFileInfo{
					{Source: "-", BytesResult: written},
				},
			}, true, stdout, nil)
		}
		return 0
	}

	var results []ExtractedFileInfo
	exitCode := 0

	for _, file := range files {
		absPath := file
		if !filepath.IsAbs(absPath) {
			absPath = filepath.Join(cwd, file)
		}

		info, err := os.Stat(absPath)
		if err != nil {
			exitCode = 1
			fmt.Fprintf(stderr, "bzcat: %s: No such file or directory\n", file)
			results = append(results, ExtractedFileInfo{
				Source: file,
				Error:  "No such file or directory",
			})
			continue
		}

		if info.IsDir() {
			exitCode = 1
			fmt.Fprintf(stderr, "bzcat: %s: Is a directory\n", file)
			results = append(results, ExtractedFileInfo{
				Source: file,
				Error:  "Is a directory",
			})
			continue
		}

		err = func() error {
			srcFile, err := os.Open(absPath)
			if err != nil {
				return err
			}
			defer srcFile.Close()

			bzReader := bzip2.NewReader(srcFile)
			written, err := io.Copy(stdout, bzReader)
			if err != nil {
				return err
			}

			results = append(results, ExtractedFileInfo{
				Source:      file,
				BytesResult: written,
			})
			return nil
		}()

		if err != nil {
			exitCode = 1
			fmt.Fprintf(stderr, "bzcat: %v\n", err)
			results = append(results, ExtractedFileInfo{
				Source: file,
				Error:  err.Error(),
			})
		}
	}

	if jsonMode {
		common.Render("bzcat", BzcatResult{Files: results}, true, stdout, nil)
	}

	return exitCode
}
