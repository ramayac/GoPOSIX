// Package wget implements the POSIX-compliant network downloader utility.
package wget

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
)

// WgetResult represents the data returned in --json mode.
type WgetResult struct {
	URL             string `json:"url"`
	OutputFile      string `json:"output_file"`
	BytesDownloaded int64  `json:"bytes_downloaded"`
	StatusCode      int    `json:"status_code"`
}

var spec = common.FlagSpec{
	Defs: []common.FlagDef{
		{Short: "q", Long: "quiet", Type: common.FlagBool},
		{Short: "O", Type: common.FlagValue},
		{Short: "P", Type: common.FlagValue},
		{Long: "json", Type: common.FlagBool},
	},
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer, cwd string) int {
	flags, err := common.ParseFlags(args, spec)
	if err != nil {
		fmt.Fprintf(stderr, "wget: %v\n", err)
		return 2
	}

	jsonMode := flags.Has("json")
	quietMode := flags.Has("quiet") || jsonMode

	posArgs := flags.Positional
	if len(posArgs) != 1 {
		fmt.Fprintln(stderr, "wget: missing URL")
		return 1
	}

	rawURL := posArgs[0]
	u, err := url.Parse(rawURL)
	if err != nil {
		fmt.Fprintf(stderr, "wget: invalid URL: %v\n", err)
		return 1
	}

	// Default scheme to http if empty
	if u.Scheme == "" {
		u.Scheme = "http"
		rawURL = u.String()
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		fmt.Fprintf(stderr, "wget: unsupported scheme: %s\n", u.Scheme)
		return 1
	}

	// Infer default filename
	defaultFilename := "index.html"
	if u.Path != "" && !strings.HasSuffix(u.Path, "/") {
		defaultFilename = filepath.Base(u.Path)
	}

	var outPath string
	writeToStdout := false

	// Resolve output path
	if flags.Has("O") {
		oVal := flags.Get("O")
		if oVal == "-" {
			writeToStdout = true
			outPath = "-"
		} else {
			outPath = oVal
			if !filepath.IsAbs(outPath) && cwd != "" {
				outPath = filepath.Join(cwd, outPath)
			}
		}
	} else {
		// Resolve prefix directory -P
		prefixDir := "."
		if flags.Has("P") {
			prefixDir = flags.Get("P")
		}
		if !filepath.IsAbs(prefixDir) && cwd != "" {
			prefixDir = filepath.Join(cwd, prefixDir)
		}

		// Ensure directory exists
		if err := os.MkdirAll(prefixDir, 0755); err != nil {
			fmt.Fprintf(stderr, "wget: failed to create directory %s: %v\n", prefixDir, err)
			return 1
		}

		outPath = filepath.Join(prefixDir, defaultFilename)
	}

	if !quietMode {
		fmt.Fprintf(stderr, "Downloading %s...\n", rawURL)
	}

	// Perform HTTP Request
	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		fmt.Fprintf(stderr, "wget: failed to create request: %v\n", err)
		return 1
	}
	req.Header.Set("User-Agent", "Wget/1.20 (GoPOSIX)")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintf(stderr, "wget: request failed: %v\n", err)
		return 1
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		fmt.Fprintf(stderr, "wget: server returned status: %s\n", resp.Status)
		return 1
	}

	// Write to destination
	var writer io.Writer
	var f *os.File
	if writeToStdout {
		writer = stdout
	} else {
		f, err = os.Create(outPath)
		if err != nil {
			fmt.Fprintf(stderr, "wget: failed to create file %s: %v\n", outPath, err)
			return 1
		}
		defer f.Close()
		writer = f
	}

	nBytes, err := io.Copy(writer, resp.Body)
	if err != nil {
		fmt.Fprintf(stderr, "wget: download failed: %v\n", err)
		return 1
	}

	if !quietMode {
		if writeToStdout {
			fmt.Fprintf(stderr, "Download complete (%d bytes written to stdout).\n", nBytes)
		} else {
			fmt.Fprintf(stderr, "Download complete. Saved %d bytes to %s\n", nBytes, outPath)
		}
	}

	// Standard output representation
	resolvedOutPath := outPath
	if !writeToStdout && cwd != "" {
		// Return relative output file in JSON data for portability in tests
		if rel, err := filepath.Rel(cwd, outPath); err == nil {
			resolvedOutPath = rel
		}
	}

	result := WgetResult{
		URL:             rawURL,
		OutputFile:      resolvedOutPath,
		BytesDownloaded: nBytes,
		StatusCode:      resp.StatusCode,
	}

	if jsonMode {
		common.Render("wget", result, true, stdout, func() {})
	}

	return 0
}

func init() {
	dispatch.Register(dispatch.Command{
		Name:  "wget",
		Usage: "Retrieve files via HTTP or HTTPS",
		Run:   run,
	})
}
