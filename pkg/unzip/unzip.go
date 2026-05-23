// Package unzip implements the POSIX-compliant unzip utility.
package unzip

import (
	"archive/zip"
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

	r, err := zip.OpenReader(absZipPath)
	if err != nil {
		if jsonMode {
			common.RenderError("unzip", 1, "OPEN_ERROR", err.Error(), true, stderr)
		} else {
			fmt.Fprintf(stderr, "unzip: corrupted data\n")
			fmt.Fprintf(stderr, "unzip: inflate error\n")
		}
		return 1
	}
	defer r.Close()

	destDir := cwd
	if flags.Has("d") {
		destDir = flags.Get("d")
		if !filepath.IsAbs(destDir) {
			destDir = filepath.Join(cwd, destDir)
		}
	}

	stdoutMode := flags.Has("p")
	quietMode := flags.Has("q")
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

			if stdoutMode {
				_, err = io.Copy(stdout, rc)
				return err
			}

			destPath := filepath.Clean(filepath.Join(destDir, f.Name))

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

			if !quietMode && !jsonMode {
				fmt.Fprintf(stdout, "  inflating: %s\n", f.Name)
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
				fmt.Fprintf(stderr, "unzip: error extracting %s: %v\n", f.Name, err)
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
