// Package bunzip2 implements the POSIX-compliant bunzip2 utility.
package bunzip2

import (
	"compress/bzip2"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

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
		Name:  "bunzip2",
		Usage: "Decompress bzip2 compressed files",
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

// Bunzip2Result represents the JSON output format.
type Bunzip2Result struct {
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
			common.RenderError("bunzip2", 1, "FLAG_ERROR", err.Error(), true, stderr)
		} else {
			fmt.Fprintf(stderr, "bunzip2: %v\n", err)
		}
		return 1
	}

	if flags.Has("h") || flags.Has("help") {
		helpText := "Usage: bunzip2 [-cfkq] [FILE]...\n\n" +
			"Decompress FILEs (default: stdin to stdout).\n\n" +
			"Options:\n" +
			"  -c, --stdout   Write to standard output\n" +
			"  -f, --force    Force overwrite of output files\n" +
			"  -k, --keep     Keep (don't delete) input files\n" +
			"  -q, --quiet    Suppress non-critical error messages"
		common.Render("bunzip2", struct {
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
		bzReader := bzip2.NewReader(stdin)
		written, err := io.Copy(stdout, bzReader)
		if err != nil {
			if !quietMode {
				fmt.Fprintf(stderr, "bunzip2: corrupted data\n")
			}
			if jsonMode {
				common.RenderError("bunzip2", 1, "DECOMPRESS_ERROR", err.Error(), true, stderr)
			}
			return 1
		}
		if jsonMode {
			common.Render("bunzip2", Bunzip2Result{
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
				fmt.Fprintf(stderr, "bunzip2: %s: No such file or directory\n", file)
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
				fmt.Fprintf(stderr, "bunzip2: %s: Is a directory\n", file)
			}
			results = append(results, ExtractedFileInfo{
				Source: file,
				Error:  "Is a directory",
			})
			continue
		}

		// Determine destination name & validate suffix
		var destName string
		lowered := strings.ToLower(file)
		if strings.HasSuffix(lowered, ".bz2") {
			destName = file[:len(file)-4]
		} else if strings.HasSuffix(lowered, ".tbz2") {
			destName = file[:len(file)-5] + ".tar"
		} else if strings.HasSuffix(lowered, ".tbz") {
			destName = file[:len(file)-4] + ".tar"
		} else {
			// Suffix not recognized
			exitCode = 1
			if !quietMode {
				fmt.Fprintf(stderr, "bunzip2: %s: unknown suffix - ignored\n", file)
			}
			results = append(results, ExtractedFileInfo{
				Source: file,
				Error:  "unknown suffix - ignored",
			})
			continue
		}

		// Perform decompression
		err = func() error {
			srcFile, err := os.Open(absPath)
			if err != nil {
				return err
			}
			defer srcFile.Close()

			bzReader := bzip2.NewReader(srcFile)

			if stdoutMode {
				written, err := io.Copy(stdout, bzReader)
				if err != nil {
					return err
				}
				results = append(results, ExtractedFileInfo{
					Source:      file,
					Destination: "-",
					BytesResult: written,
				})
				return nil
			}

			// Extract to file
			absDestPath := destName
			if !filepath.IsAbs(absDestPath) {
				absDestPath = filepath.Join(cwd, destName)
			}

			// Check if dest already exists
			if _, err := os.Stat(absDestPath); err == nil && !forceMode {
				return fmt.Errorf("can't open '%s': File exists", destName)
			}

			destFile, err := os.OpenFile(absDestPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
			if err != nil {
				return err
			}
			defer destFile.Close()

			written, err := io.Copy(destFile, bzReader)
			if err != nil {
				// Clean up partially written dest file
				destFile.Close()
				os.Remove(absDestPath)
				return err
			}

			results = append(results, ExtractedFileInfo{
				Source:      file,
				Destination: destName,
				BytesResult: written,
			})

			// If success and not keepMode, delete source
			if !keepMode {
				os.Remove(absPath)
			}

			return nil
		}()

		if err != nil {
			exitCode = 1
			if !quietMode {
				fmt.Fprintf(stderr, "bunzip2: %v\n", err)
			}
			results = append(results, ExtractedFileInfo{
				Source: file,
				Error:  err.Error(),
			})
		}
	}

	if jsonMode {
		common.Render("bunzip2", Bunzip2Result{Files: results}, true, stdout, nil)
	}

	return exitCode
}
