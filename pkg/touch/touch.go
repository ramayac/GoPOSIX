// Package touch implements the POSIX touch utility.
package touch

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
)

var spec = common.FlagSpec{
	Defs: []common.FlagDef{
		{Short: "c", Long: "no-create", Type: common.FlagBool},
		{Short: "t", Long: "time", Type: common.FlagValue},
		{Short: "r", Long: "reference", Type: common.FlagValue},
		{Short: "d", Long: "date", Type: common.FlagValue},
		{Long: "json", Type: common.FlagBool},
	},
}

// TouchResult is the --json output.
type TouchResult struct {
	Touched []string `json:"touched"`
}

// Run creates files or updates their timestamps.
// If noCreate is true (-c flag), existing files are updated but new files are
// silently skipped (no error).
func Run(paths []string, ts time.Time, noCreate bool) (TouchResult, error) {
	var result TouchResult
	for _, p := range paths {
		if noCreate {
			// -c: don't create, only update existing files.
			info, err := os.Stat(p)
			if err != nil {
				if os.IsNotExist(err) {
					continue // silently skip
				}
				return result, err
			}
			if info.IsDir() {
				// Directories can have their times updated.
				if err := os.Chtimes(p, ts, ts); err != nil {
					return result, err
				}
				result.Touched = append(result.Touched, p)
				continue
			}
		}
		f, err := os.OpenFile(p, os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return result, err
		}
		f.Close()
		if err := os.Chtimes(p, ts, ts); err != nil {
			return result, err
		}
		result.Touched = append(result.Touched, p)
	}
	return result, nil
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer, cwd string) int {
	flags, err := common.ParseFlags(args, spec)
	if err != nil {
		fmt.Fprintf(os.Stderr, "touch: %v\n", err)
		return 2
	}
	jsonMode := flags.Has("json")
	ts := time.Now()

	if ref := flags.Get("r"); ref != "" {
		info, err := os.Stat(ref)
		if err != nil {
			fmt.Fprintf(os.Stderr, "touch: %v\n", err)
			return 1
		}
		ts = info.ModTime()
	} else if dStr := flags.Get("d"); dStr != "" {
		// GNU-style -d flag: parse free-form date string.
		// Try common formats first.
		layouts := []string{
			"2006-01-02 15:04:05",
			"2006-01-02T15:04:05",
			"2006-01-02",
			"01/02/2006 15:04:05",
			"Jan 2 15:04:05 2006",
			time.RFC3339,
			time.RFC1123,
			time.RFC1123Z,
		}
		var parsed bool
		for _, layout := range layouts {
			t, err := time.ParseInLocation(layout, dStr, time.Local)
			if err == nil {
				ts = t
				parsed = true
				break
			}
		}
		if !parsed {
			fmt.Fprintf(os.Stderr, "touch: invalid date format: %q\n", dStr)
			return 1
		}
	} else if tStr := flags.Get("t"); tStr != "" {
		layouts := []string{"200601021504.05", "200601021504", "0601021504"}
		var parsed bool
		for _, layout := range layouts {
			t, err := time.ParseInLocation(layout, tStr, time.Local)
			if err == nil {
				ts = t
				parsed = true
				break
			}
		}
		if !parsed {
			fmt.Fprintf(os.Stderr, "touch: invalid date format: %q\n", tStr)
			return 1
		}
	}

	if len(flags.Positional) == 0 {
		fmt.Fprintln(os.Stderr, "touch: missing file operand")
		return 1
	}
	noCreate := flags.Has("c")
	result, err := Run(flags.Positional, ts, noCreate)
	if err != nil {
		fmt.Fprintf(os.Stderr, "touch: %v\n", err)
		common.RenderError("touch", 1, "ETOUCH", err.Error(), jsonMode, stdout)
		return 1
	}
	common.Render("touch", result, jsonMode, stdout, func() {})
	return 0
}

func init() {
	dispatch.Register(dispatch.Command{Name: "touch", Usage: "Change file timestamps or create files", Run: run})
}
