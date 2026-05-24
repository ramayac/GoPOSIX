// Package unzip implements the POSIX-compliant unzip utility.
package unzip

import (
	"archive/zip"
	"encoding/binary"
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
		{Short: "p", Long: "stdout", Type: common.FlagBool},
		{Short: "q", Long: "quiet", Type: common.FlagBool},
		{Short: "o", Long: "overwrite", Type: common.FlagBool},
		{Short: "d", Long: "dir", Type: common.FlagValue},
		{Short: "l", Long: "list", Type: common.FlagBool},
		{Short: "t", Long: "test", Type: common.FlagBool},
		{Short: "h", Long: "help", Type: common.FlagBool},
		{Long: "json", Type: common.FlagBool},
	},
}

func init() {
	dispatch.Register(dispatch.Command{
		Name:  "unzip",
		Usage: "Extract files from a ZIP archive",
		Run:   run,
	})
}

// UnzippedFileInfo represents a file in the zip archive for JSON output.
type UnzippedFileInfo struct {
	Name           string `json:"name"`
	Size           int64  `json:"size"`
	CompressedSize int64  `json:"compressedSize"`
	IsDir          bool   `json:"isDir"`
	Error          string `json:"error,omitempty"`
}

// UnzipResult represents the JSON response structure.
type UnzipResult struct {
	Archive string             `json:"archive"`
	Files   []UnzippedFileInfo `json:"files"`
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
			common.RenderError("unzip", 1, "FLAG_ERROR", err.Error(), true, stderr)
		} else {
			fmt.Fprintf(stderr, "unzip: %v\n", err)
		}
		return 1
	}

	if flags.Has("h") || flags.Has("help") {
		helpText := "Usage: unzip [-pqolth] [-d DIR] ZIPFILE [FILE...]\n\n" +
			"Extract files from a ZIP archive.\n\n" +
			"Options:\n" +
			"  -p             Extract to stdout (no progress messages)\n" +
			"  -q             Quiet mode\n" +
			"  -o             Overwrite existing files without asking\n" +
			"  -d DIR         Extract into DIR\n" +
			"  -l             List archive contents\n" +
			"  -t             Test archive integrity\n" +
			"  -h             Print help"
		common.Render("unzip", struct {
			Help string `json:"help"`
		}{Help: helpText}, jsonMode, stdout, func() {
			fmt.Fprintln(stdout, helpText)
		})
		return 0
	}

	if len(flags.Positional) == 0 {
		if jsonMode {
			common.RenderError("unzip", 1, "MISSING_ARGUMENT", "missing ZIPFILE", true, stderr)
		} else {
			fmt.Fprintln(stderr, "unzip: missing ZIPFILE")
		}
		return 1
	}

	zipPath := flags.Positional[0]
	absZipPath := zipPath
	if !filepath.IsAbs(absZipPath) {
		absZipPath = filepath.Join(cwd, zipPath)
	}

	// Filter matches: positional arguments after ZIPFILE
	filters := flags.Positional[1:]

	// Determine modes early — needed for error path output decisions.
	quietMode := flags.Has("q")
	stdoutMode := flags.Has("p")

	r, err := zip.OpenReader(absZipPath)
	if err != nil {
		// Scan for filenames even in corrupted archives so we can emit
		// header/inflating lines and slash-prefix warnings (BusyBox compat).
		firstFile, slashWarn := scanCorruptedZip(absZipPath)
		firstFile = sanitizeFilename(firstFile)
		if slashWarn && !quietMode && !jsonMode {
			fmt.Fprintf(stderr, "unzip: removing leading '/' from member names\n")
		}
		if !quietMode && !stdoutMode && !jsonMode && firstFile != "" {
			fmt.Fprintf(stdout, "Archive:  %s\n", zipPath)
			fmt.Fprintf(stdout, "  inflating: %s\n", firstFile)
		}
		if jsonMode {
			common.RenderError("unzip", 1, "OPEN_ERROR", err.Error(), true, stderr)
		} else {
			fmt.Fprintf(stderr, "unzip: corrupted data\n")
			fmt.Fprintf(stderr, "unzip: inflate error\n")
		}
		return 1
	}
	defer r.Close()

	// If archive opens but has zero entries while the file is non-trivial,
	// it's a corrupted zip — report it as corrupted data.
	if len(r.File) == 0 {
		fi, statErr := os.Stat(absZipPath)
		if statErr == nil && fi.Size() > 128 {
			// File has data but no parseable entries — corrupted.
			firstFile, slashWarn := scanCorruptedZip(absZipPath)
			firstFile = sanitizeFilename(firstFile)
			if slashWarn && !quietMode && !jsonMode {
				fmt.Fprintf(stderr, "unzip: removing leading '/' from member names\n")
			}
			if !quietMode && !stdoutMode && !jsonMode && firstFile != "" {
				fmt.Fprintf(stdout, "Archive:  %s\n", zipPath)
				fmt.Fprintf(stdout, "  inflating: %s\n", firstFile)
			}
			if !jsonMode {
				fmt.Fprintf(stderr, "unzip: corrupted data\n")
				fmt.Fprintf(stderr, "unzip: inflate error\n")
			}
			return 1
		}
	}

	destDir := cwd
	if flags.Has("d") {
		destDir = flags.Get("d")
		if !filepath.IsAbs(destDir) {
			destDir = filepath.Join(cwd, destDir)
		}
	}

	overwriteMode := flags.Has("o")
	listMode := flags.Has("l")
	testMode := flags.Has("t")

	// Filter utility
	shouldExtract := func(name string) bool {
		if len(filters) == 0 {
			return true
		}
		for _, filter := range filters {
			if name == filter {
				return true
			}
		}
		return false
	}

	var results []UnzippedFileInfo

	if listMode {
		if !jsonMode {
			fmt.Fprintf(stdout, "  Length      Date    Time    Name\n")
			fmt.Fprintf(stdout, "---------  ---------- -----   ----\n")
		}
		var totalSize int64
		var count int
		for _, f := range r.File {
			if !shouldExtract(f.Name) {
				continue
			}
			totalSize += int64(f.UncompressedSize64)
			count++
			results = append(results, UnzippedFileInfo{
				Name:           f.Name,
				Size:           int64(f.UncompressedSize64),
				CompressedSize: int64(f.CompressedSize64),
				IsDir:          f.FileInfo().IsDir(),
			})

			if !jsonMode {
				modTime := f.Modified
				fmt.Fprintf(stdout, "%9d  %04d-%02d-%02d %02d:%02d   %s\n",
					f.UncompressedSize64,
					modTime.Year(), modTime.Month(), modTime.Day(),
					modTime.Hour(), modTime.Minute(),
					f.Name,
				)
			}
		}
		if !jsonMode {
			fmt.Fprintf(stdout, "---------                     -------\n")
			label := "files"
			if count == 1 {
				label = "file"
			}
			fmt.Fprintf(stdout, "%9d                     %d %s\n", totalSize, count, label)
		}
		if jsonMode {
			common.Render("unzip", UnzipResult{Archive: zipPath, Files: results}, true, stdout, nil)
		}
		return 0
	}

	if testMode {
		exitCode := 0
		for _, f := range r.File {
			if !shouldExtract(f.Name) {
				continue
			}
			err := func() error {
				rc, err := f.Open()
				if err != nil {
					return err
				}
				defer rc.Close()
				_, err = io.Copy(io.Discard, rc)
				return err
			}()

			errStr := ""
			if err != nil {
				exitCode = 1
				errStr = err.Error()
				if !quietMode {
					fmt.Fprintf(stdout, "  testing: %s   failed\n", f.Name)
				}
			} else {
				if !quietMode {
					fmt.Fprintf(stdout, "  testing: %s   OK\n", f.Name)
				}
			}

			results = append(results, UnzippedFileInfo{
				Name:           f.Name,
				Size:           int64(f.UncompressedSize64),
				CompressedSize: int64(f.CompressedSize64),
				IsDir:          f.FileInfo().IsDir(),
				Error:          errStr,
			})
		}
		if !quietMode && exitCode == 0 {
			fmt.Fprintf(stdout, "No errors detected in compressed data of %s.\n", zipPath)
		}
		if jsonMode {
			common.Render("unzip", UnzipResult{Archive: zipPath, Files: results}, true, stdout, nil)
		}
		return exitCode
	}

	if !quietMode && !stdoutMode && !jsonMode {
		fmt.Fprintf(stdout, "Archive:  %s\n", zipPath)
	}

	strippedSlash := false
	exitCode := 0
	for _, f := range r.File {
		if !shouldExtract(f.Name) {
			continue
		}

		// Strip leading '/' from member names (security convention, per POSIX).
		// This must happen before any mode-specific handling so stdout/list/test
		// paths also trigger the warning.
		if strings.HasPrefix(f.Name, "/") {
			if !strippedSlash && !quietMode && !jsonMode {
				fmt.Fprintf(stderr, "unzip: removing leading '/' from member names\n")
				strippedSlash = true
			}
		}

		// Print inflating progress for non-quiet, non-stdout, non-json modes.
		if !quietMode && !stdoutMode && !jsonMode {
			fmt.Fprintf(stdout, "  inflating: %s\n", sanitizeFilename(f.Name))
		}

		err := func() error {
			rc, err := f.Open()
			if err != nil {
				return err
			}
			defer rc.Close()

			if stdoutMode {
				_, err = io.Copy(stdout, rc)
				return err
			}

			destPath := filepath.Clean(filepath.Join(destDir, f.Name))

			// Strip leading '/' from member names (security convention)
			if strings.HasPrefix(f.Name, "/") {
				destPath = filepath.Clean(filepath.Join(destDir, f.Name[1:]))
			}

			// Security guard: prevent directory traversal attacks
			if !strings.HasPrefix(destPath, destDir) {
				return fmt.Errorf("prevented directory traversal: %s", f.Name)
			}

			if f.FileInfo().IsDir() {
				return os.MkdirAll(destPath, f.Mode())
			}

			// Create parent directory
			if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
				return err
			}

			// File overwrite check
			if _, err := os.Stat(destPath); err == nil && !overwriteMode {
				// skip
				return nil
			}

			out, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				return err
			}
			defer out.Close()

			_, err = io.Copy(out, rc)
			return err
		}()

		errStr := ""
		if err != nil {
			exitCode = 1
			errStr = err.Error()
			if !quietMode {
				// Detect decompression errors and emit BusyBox-compatible messages.
				// Go's archive/zip surfaces flate/lzma/checksum errors through
				// io.Copy or f.Open when compressed data is corrupted.
				errMsg := err.Error()
				isDecompressErr := strings.Contains(errMsg, "flate") ||
					strings.Contains(errMsg, "checksum") ||
					strings.Contains(errMsg, "unexpected EOF") ||
					strings.Contains(errMsg, "invalid") ||
					strings.Contains(errMsg, "unsupported compression")
				if isDecompressErr {
					fmt.Fprintf(stderr, "unzip: corrupted data\n")
					fmt.Fprintf(stderr, "unzip: inflate error\n")
				} else {
					fmt.Fprintf(stderr, "unzip: error extracting %s: %v\n", f.Name, err)
				}
			}
		}

		results = append(results, UnzippedFileInfo{
			Name:           f.Name,
			Size:           int64(f.UncompressedSize64),
			CompressedSize: int64(f.CompressedSize64),
			IsDir:          f.FileInfo().IsDir(),
			Error:          errStr,
		})
	}

	if jsonMode {
		common.Render("unzip", UnzipResult{Archive: zipPath, Files: results}, true, stdout, nil)
	}

	return exitCode
}

// scanCorruptedZip scans a zip file for local file headers in corrupted
// archives. Returns the first filename found and whether any filename
// starts with '/'. Used to emit BusyBox-compatible output even when
// Go's archive/zip can't parse the file normally.
func scanCorruptedZip(zipPath string) (firstFile string, slashWarn bool) {
	f, err := os.Open(zipPath)
	if err != nil {
		return "", false
	}
	defer f.Close()

	// Read up to 64KB — enough to find local file headers near the start.
	buf := make([]byte, 65536)
	n, err := io.ReadFull(f, buf)
	if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
		return "", false
	}
	buf = buf[:n]

	// Look for local file header signatures: PK\x03\x04 (standard) or
	// PK\x03\x00 (some corrupted variants).
	for i := 0; i < len(buf)-30; i++ {
		if buf[i] != 'P' || buf[i+1] != 'K' || buf[i+2] != 0x03 {
			continue
		}
		sigByte := buf[i+3]
		if sigByte != 0x04 && sigByte != 0x00 {
			continue
		}
		// Local file header: offset 26 = filename length (uint16 LE)
		if i+30 > len(buf) {
			break
		}
		nameLen := binary.LittleEndian.Uint16(buf[i+26 : i+28])
		extraLen := binary.LittleEndian.Uint16(buf[i+28 : i+30])
		nameStart := i + 30
		nameEnd := nameStart + int(nameLen)
		if nameEnd > len(buf) || nameLen == 0 {
			continue
		}
		name := string(buf[nameStart:nameEnd])
		if name[0] == '/' {
			slashWarn = true
			name = name[1:] // strip leading '/' for display
		}
		if firstFile == "" && name != "" {
			firstFile = name
		}
		// Skip past this header to continue scanning.
		i = nameEnd + int(extraLen) - 1
	}
	return firstFile, slashWarn
}

// sanitizeFilename replaces control characters in filenames
// with '?' to match BusyBox's display of binary/corrupted filenames.
// Extended ASCII (0x80+) and printable ASCII are left untouched.
func sanitizeFilename(name string) string {
	if name == "" {
		return name
	}
	b := []byte(name)
	for i, c := range b {
		if c < 0x20 || c == 0x7f {
			b[i] = '?'
		}
	}
	return string(b)
}
