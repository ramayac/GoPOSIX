// Package uncompress implements the POSIX-compliant uncompress utility.
package uncompress

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/hotei/dcompress"
	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
)

var spec = common.FlagSpec{
	Defs: []common.FlagDef{
		{Short: "c", Long: "stdout", Type: common.FlagBool},
		{Short: "f", Long: "force", Type: common.FlagBool},
		{Short: "k", Long: "keep", Type: common.FlagBool},
		{Short: "q", Long: "quiet", Type: common.FlagBool},
		{Short: "h", Long: "help", Type: common.FlagBool},
		{Long: "json", Type: common.FlagBool},
	},
}

func init() {
	dispatch.Register(dispatch.Command{
		Name:  "uncompress",
		Usage: "Decompress .Z LZW compressed files",
		Run:   run,
	})
}

// ExtractedFileInfo contains information about a single extracted file for JSON output.
type ExtractedFileInfo struct {
	Source      string `json:"source"`
	Destination string `json:"destination,omitempty"`
	BytesResult int64  `json:"bytesResult"`
	Error       string `json:"error,omitempty"`
}

// UncompressResult represents the JSON output format.
type UncompressResult struct {
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
			common.RenderError("uncompress", 1, "FLAG_ERROR", err.Error(), true, stderr)
		} else {
			fmt.Fprintf(stderr, "uncompress: %v\n", err)
		}
		return 1
	}

	if flags.Has("h") || flags.Has("help") {
		helpText := "Usage: uncompress [-cfkq] [FILE]...\n\n" +
			"Decompress .Z FILEs (default: stdin to stdout).\n\n" +
			"Options:\n" +
			"  -c, --stdout   Write to standard output\n" +
			"  -f, --force    Force overwrite of output files\n" +
			"  -k, --keep     Keep (don't delete) input files\n" +
			"  -q, --quiet    Suppress non-critical error messages"
		common.Render("uncompress", struct {
			Help string `json:"help"`
		}{Help: helpText}, jsonMode, stdout, func() {
			fmt.Fprintln(stdout, helpText)
		})
		return 0
	}

	stdoutMode := flags.Has("c") || flags.Has("stdout")
	forceMode := flags.Has("f") || flags.Has("force")
	keepMode := flags.Has("k") || flags.Has("keep")
	quietMode := flags.Has("q") || flags.Has("quiet")

	files := flags.Positional

	// Default: stdin to stdout
	if len(files) == 0 || (len(files) == 1 && files[0] == "-") {
		var dcReader io.Reader
		err = func() (errRet error) {
			defer func() {
				if r := recover(); r != nil {
					errRet = fmt.Errorf("corrupted data")
				}
			}()
			var rdr io.Reader
			rdr, errRet = dcompress.NewReader(stdin)
			if errRet != nil {
				return errRet
			}
			dcReader = rdr
			return nil
		}()

		if err != nil {
			if !quietMode {
				fmt.Fprintf(stderr, "uncompress: corrupted data\n")
			}
			if jsonMode {
				common.RenderError("uncompress", 1, "DECOMPRESS_ERROR", err.Error(), true, stderr)
			}
			return 1
		}

		var written int64
		err = func() (errRet error) {
			defer func() {
				if r := recover(); r != nil {
					errRet = fmt.Errorf("corrupted data")
				}
			}()
			var wr int64
			wr, errRet = io.Copy(stdout, dcReader)
			written = wr
			return errRet
		}()

		if err != nil {
			if !quietMode {
				fmt.Fprintf(stderr, "uncompress: corrupted data\n")
			}
			if jsonMode {
				common.RenderError("uncompress", 1, "DECOMPRESS_ERROR", err.Error(), true, stderr)
			}
			return 1
		}

		if jsonMode {
			common.Render("uncompress", UncompressResult{
				Files: []ExtractedFileInfo{
					{Source: "-", Destination: "-", BytesResult: written},
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
			if !quietMode {
				fmt.Fprintf(stderr, "uncompress: %s: No such file or directory\n", file)
			}
			results = append(results, ExtractedFileInfo{
				Source: file,
				Error:  "No such file or directory",
			})
			continue
		}

		if info.IsDir() {
			exitCode = 1
			if !quietMode {
				fmt.Fprintf(stderr, "uncompress: %s: Is a directory\n", file)
			}
			results = append(results, ExtractedFileInfo{
				Source: file,
				Error:  "Is a directory",
			})
			continue
		}

		// Suffix resolution
		var destName string
		lowered := strings.ToLower(file)
		if strings.HasSuffix(lowered, ".z") {
			destName = file[:len(file)-2]
		} else {
			exitCode = 1
			if !quietMode {
				fmt.Fprintf(stderr, "uncompress: %s: unknown suffix - ignored\n", file)
			}
			results = append(results, ExtractedFileInfo{
				Source: file,
				Error:  "unknown suffix - ignored",
			})
			continue
		}

		err = func() (errRet error) {
			defer func() {
				if r := recover(); r != nil {
					errRet = fmt.Errorf("corrupted data")
				}
			}()

			srcFile, err := os.Open(absPath)
			if err != nil {
				return err
			}
			defer srcFile.Close()

			dcReader, err := dcompress.NewReader(srcFile)
			if err != nil {
				return fmt.Errorf("corrupted data")
			}

			if stdoutMode {
				written, err := io.Copy(stdout, dcReader)
				if err != nil {
					return fmt.Errorf("corrupted data")
				}
				results = append(results, ExtractedFileInfo{
					Source:      file,
					Destination: "-",
					BytesResult: written,
				})
				return nil
			}

			absDestPath := destName
			if !filepath.IsAbs(absDestPath) {
				absDestPath = filepath.Join(cwd, destName)
			}

			if _, err := os.Stat(absDestPath); err == nil && !forceMode {
				return fmt.Errorf("can't open '%s': File exists", destName)
			}

			destFile, err := os.OpenFile(absDestPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
			if err != nil {
				return err
			}
			defer destFile.Close()

			written, err := io.Copy(destFile, dcReader)
			if err != nil {
				destFile.Close()
				os.Remove(absDestPath)
				return fmt.Errorf("corrupted data")
			}

			results = append(results, ExtractedFileInfo{
				Source:      file,
				Destination: destName,
				BytesResult: written,
			})

			if !keepMode {
				os.Remove(absPath)
			}

			return nil
		}()


		if err != nil {
			exitCode = 1
			if !quietMode {
				fmt.Fprintf(stderr, "uncompress: %v\n", err)
			}
			results = append(results, ExtractedFileInfo{
				Source: file,
				Error:  err.Error(),
			})
		}
	}

	if jsonMode {
		common.Render("uncompress", UncompressResult{Files: results}, true, stdout, nil)
	}

	return exitCode
}
