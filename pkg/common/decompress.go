// Package common — shared decompression core for the file-decompression
// family (unlzma, bunzip2, uncompress, bzcat).
//
// DecompressMode implements the common flag parsing, stdin handling,
// suffix resolution, file walking, output-file management, and JSON
// rendering. Each utility supplies its reader constructor, suffixes,
// help text, and error-message style.
package common

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// DecompSuffix maps one input-file extension to the output name.
type DecompSuffix struct {
	Ext    string // ".bz2"
	Cut    int    // bytes to strip from the input name
	Append string // suffix to add, e.g. ".tar" for .tbz/.tbz2
}

// DecompressSpec describes one decompression utility.
type DecompressSpec struct {
	ProgName  string // message prefix and --json command name
	HelpText  string
	Suffixes  []DecompSuffix // empty means no suffix check (cat mode)
	NewReader func(io.Reader) (io.Reader, error)
	// CorruptMsg is the stderr message for decompression failures.
	CorruptMsg string
	// Quietable enables the -q flag (suppresses non-critical messages).
	Quietable bool
	// CatMode always writes to stdout and never touches output files.
	CatMode bool
	// RecoverPanics converts library panics into errors (uncompress).
	RecoverPanics bool
	// SilenceLog suppresses library log output (uncompress).
	SilenceLog bool
	// Flags is the utility's own flag spec (bzcat has no -c/-f/-k/-q).
	Flags FlagSpec
}

// DecompFileInfo is one extracted file in --json output.
type DecompFileInfo struct {
	Source      string `json:"source"`
	Destination string `json:"destination,omitempty"`
	BytesResult int64  `json:"bytesResult"`
	Error       string `json:"error,omitempty"`
}

// DecompressMode runs the shared decompression logic. Returns the exit code.
func DecompressMode(spec DecompressSpec, args []string, stdin io.Reader, stdout, stderr io.Writer, cwd string) int {
	if spec.SilenceLog {
		orig := log.Writer()
		log.SetOutput(io.Discard)
		defer log.SetOutput(orig)
	}

	jsonMode := false
	for _, arg := range args {
		if arg == "--json" {
			jsonMode = true
			break
		}
	}

	flags, err := ParseFlags(args, spec.Flags)
	if err != nil {
		return RenderFlagError(spec.ProgName, args, err, stderr, 1)
	}

	if flags.Has("h") || flags.Has("help") {
		Render(spec.ProgName, struct {
			Help string `json:"help"`
		}{Help: spec.HelpText}, jsonMode, stdout, func() {
			fmt.Fprintln(stdout, spec.HelpText)
		})
		return 0
	}

	stdoutMode := spec.CatMode || flags.Has("c") || flags.Has("stdout")
	forceMode := flags.Has("f") || flags.Has("force")
	keepMode := flags.Has("k") || flags.Has("keep")
	quietMode := spec.Quietable && (flags.Has("q") || flags.Has("quiet"))

	files := flags.Positional

	// Default: stdin to stdout.
	if len(files) == 0 || (len(files) == 1 && files[0] == "-") {
		reader, err := newReaderRecovering(spec, stdin)
		var written int64
		if err == nil {
			written, err = copyRecovering(stdout, reader, spec.RecoverPanics)
		}
		if err != nil {
			if !quietMode {
				fmt.Fprintf(stderr, "%s: %s\n", spec.ProgName, spec.CorruptMsg)
			}
			if jsonMode {
				RenderError(spec.ProgName, 1, "DECOMPRESS_ERROR", err.Error(), true, stderr)
			}
			return 1
		}
		if jsonMode {
			entry := DecompFileInfo{Source: "-", BytesResult: written}
			if !spec.CatMode {
				entry.Destination = "-"
			}
			Render(spec.ProgName, struct {
				Files []DecompFileInfo `json:"files"`
			}{Files: []DecompFileInfo{entry}}, true, stdout, nil)
		}
		return 0
	}

	var results []DecompFileInfo
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
				fmt.Fprintf(stderr, "%s: %s: No such file or directory\n", spec.ProgName, file)
			}
			results = append(results, DecompFileInfo{Source: file, Error: "No such file or directory"})
			continue
		}

		if info.IsDir() {
			exitCode = 1
			if !quietMode {
				fmt.Fprintf(stderr, "%s: %s: Is a directory\n", spec.ProgName, file)
			}
			results = append(results, DecompFileInfo{Source: file, Error: "Is a directory"})
			continue
		}

		// Suffix resolution.
		destName := ""
		if len(spec.Suffixes) > 0 {
			lowered := strings.ToLower(file)
			ok := false
			for _, s := range spec.Suffixes {
				if strings.HasSuffix(lowered, s.Ext) {
					destName = file[:len(file)-s.Cut] + s.Append
					ok = true
					break
				}
			}
			if !ok {
				exitCode = 1
				if !quietMode {
					fmt.Fprintf(stderr, "%s: %s: unknown suffix - ignored\n", spec.ProgName, file)
				}
				results = append(results, DecompFileInfo{Source: file, Error: "unknown suffix - ignored"})
				continue
			}
		}

		err = func() (retErr error) {
			if spec.RecoverPanics {
				defer func() {
					if r := recover(); r != nil {
						retErr = fmt.Errorf("corrupted data")
					}
				}()
			}

			srcFile, err := os.Open(absPath)
			if err != nil {
				return err
			}
			defer srcFile.Close()

			reader, err := spec.NewReader(srcFile)
			if err != nil {
				return fmt.Errorf("%s", spec.CorruptMsg)
			}

			if stdoutMode {
				written, err := io.Copy(stdout, reader)
				if err != nil {
					if spec.CatMode {
						return err
					}
					return fmt.Errorf("%s", spec.CorruptMsg)
				}
				entry := DecompFileInfo{Source: file, BytesResult: written}
				if !spec.CatMode {
					entry.Destination = "-"
				}
				results = append(results, entry)
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

			written, err := io.Copy(destFile, reader)
			if err != nil {
				destFile.Close()
				os.Remove(absDestPath)
				return fmt.Errorf("%s", spec.CorruptMsg)
			}

			results = append(results, DecompFileInfo{
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
				fmt.Fprintf(stderr, "%s: %v\n", spec.ProgName, err)
			}
			results = append(results, DecompFileInfo{Source: file, Error: err.Error()})
		}
	}

	if jsonMode {
		Render(spec.ProgName, struct {
			Files []DecompFileInfo `json:"files"`
		}{Files: results}, true, stdout, nil)
	}

	return exitCode
}

// copyRecovering copies src to dst. When recoverPanics is set, a panic in the
// copy becomes a "corrupted data" error (dcompress panics on bad input).
func copyRecovering(dst io.Writer, src io.Reader, recoverPanics bool) (written int64, err error) {
	if recoverPanics {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("corrupted data")
			}
		}()
	}
	return io.Copy(dst, src)
}

// newReaderRecovering builds the decompression reader. When recoverPanics is
// set, a panic in the constructor becomes a "corrupted data" error.
func newReaderRecovering(spec DecompressSpec, r io.Reader) (reader io.Reader, err error) {
	if spec.RecoverPanics {
		defer func() {
			if rec := recover(); rec != nil {
				err = fmt.Errorf("corrupted data")
			}
		}()
	}
	return spec.NewReader(r)
}
