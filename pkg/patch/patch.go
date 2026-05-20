// Package patch implements the POSIX patch utility — apply unified diffs to files.
package patch

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
)

// Hunk represents a single unified diff hunk.
type Hunk struct {
	OldStart int
	OldCount int
	NewStart int
	NewCount int
	Lines    []string // with +, -, or space prefix (no prefix on trailing "\ No newline")
}

// Patch represents a parsed unified diff.
type Patch struct {
	OldFile string
	NewFile string
	Hunks   []Hunk
}

var spec = common.FlagSpec{
	Defs: []common.FlagDef{
		{Short: "p", Long: "", Type: common.FlagValue},
		{Short: "R", Long: "", Type: common.FlagBool},
		{Short: "N", Long: "", Type: common.FlagBool},
		{Long: "json", Type: common.FlagBool},
	},
}

// Run applies patch to the target file. patchData contains the unified diff,
// and targetPath is the optional explicit file to patch. If empty, the target
// is derived from the patch header.
func Run(patchData []byte, targetPath string, stripLevel int, reverse, ignoreApplied bool) (*PatchResult, error) {
	patches, err := ParsePatch(strings.NewReader(string(patchData)))
	if err != nil {
		return nil, fmt.Errorf("parse patch: %w", err)
	}
	if len(patches) == 0 {
		return nil, fmt.Errorf("no patches found in input")
	}

	p := patches[0]

	// Apply strip level to filenames
	oldFile := stripPath(p.OldFile, stripLevel)
	newFile := stripPath(p.NewFile, stripLevel)

	// Determine target file
	target := targetPath
	if target == "" {
		// Use the new file name from the patch
		target = newFile
	}

	// Check if creating a new file
	isNewFile := oldFile == "/dev/null" || p.OldFile == "/dev/null"

	var original []byte
	if !isNewFile {
		var err error
		original, err = os.ReadFile(target)
		if err != nil {
			msg := "No such file or directory"
			if !os.IsNotExist(err) {
				msg = err.Error()
			}
			return &PatchResult{
				File:     target,
				Applied:  0,
				Rejected: len(p.Hunks),
				Msg:      fmt.Sprintf("patch: can't open '%s': %s", target, msg),
			}, fmt.Errorf("can't open '%s': %s", target, msg)
		}
	}

	var lines []string
	if isNewFile {
		lines = nil
	} else {
		lines = strings.Split(string(original), "\n")
		// Handle trailing newline: if file ends with \n, lines has a trailing ""
		if len(original) > 0 && original[len(original)-1] == '\n' {
			lines = lines[:len(lines)-1]
		}
	}

	result := &PatchResult{File: target, IsNew: isNewFile}
	var appliedHunks []Hunk

	for hi, hunk := range p.Hunks {
		h := hunk
		if reverse {
			h = reverseHunk(h)
		}

		newLines, ok := applyHunk(lines, h, ignoreApplied)
		if ok {
			lines = newLines
			appliedHunks = append(appliedHunks, hunk)
			result.Applied++
		} else {
			result.Rejected++
			result.FailedHunk = &hunk
			result.FailedHunkNum = hi + 1

			// Build failure message
			var sb strings.Builder
			// Check if this looks like a reversed/already-applied hunk:
			// the forward hunk's expected result already exists in the file.
			if alreadyApplied(lines, hunk) {
				sb.WriteString(fmt.Sprintf("Possibly reversed hunk %d at %d\n", hi+1, hunk.NewStart+hunk.NewCount))
			}
			sb.WriteString(fmt.Sprintf("Hunk %d FAILED %d/%d.\n", hi+1, 1, 1))
			for _, l := range hunk.Lines {
				sb.WriteString(l + "\n")
			}
			result.Msg = sb.String()

			// Keep original content on failure
			break
		}
	}

	// Write result
	if result.Applied > 0 {
		output := strings.Join(lines, "\n")
		// Add trailing newline if the original had one (or for new files)
		if isNewFile || (len(original) > 0 && original[len(original)-1] == '\n') {
			output += "\n"
		}
		if err := os.WriteFile(target, []byte(output), 0666); err != nil {
			return result, fmt.Errorf("write %s: %w", target, err)
		}
	}

	if result.Rejected > 0 {
		return result, fmt.Errorf("%d hunk(s) failed", result.Rejected)
	}

	return result, nil
}

// PatchResult holds the outcome of a patch operation.
type PatchResult struct {
	File          string `json:"file"`
	Applied       int    `json:"applied"`
	Rejected      int    `json:"rejected"`
	IsNew         bool   `json:"is_new,omitempty"`
	Msg           string `json:"message,omitempty"`
	FailedHunk    *Hunk  `json:"-"`
	FailedHunkNum int    `json:"-"`
}

// stripPath removes n leading path components.
func stripPath(path string, n int) string {
	// Always remove trailing tab and timestamp (common in diff headers)
	if idx := strings.IndexByte(path, '\t'); idx >= 0 {
		path = path[:idx]
	}
	if n <= 0 {
		return path
	}

	// Walk through path counting path components, treating consecutive
	// slashes as a single separator for counting purposes.
	// Find the byte position where n components have been skipped.
	pos := 0
	components := 0
	inComponent := false
	for i := 0; i < len(path); i++ {
		if path[i] == '/' {
			if inComponent {
				components++
				if components == n {
					// Skip past this slash and any immediately following slashes
					pos = i + 1
					for pos < len(path) && path[pos] == '/' {
						pos++
					}
					break
				}
			}
			inComponent = false
		} else {
			if !inComponent && components == 0 {
				// First component starts here
			}
			inComponent = true
		}
	}
	if components < n {
		// Not enough components — return just the last one
		// Find last non-slash sequence
		last := len(path)
		for last > 0 && path[last-1] == '/' {
			last--
		}
		start := last
		for start > 0 && path[start-1] != '/' {
			start--
		}
		return path[start:last]
	}
	return path[pos:]
}

// reverseHunk swaps additions and deletions.
func reverseHunk(h Hunk) Hunk {
	r := Hunk{
		OldStart: h.NewStart,
		OldCount: h.NewCount,
		NewStart: h.OldStart,
		NewCount: h.OldCount,
		Lines:    make([]string, len(h.Lines)),
	}
	for i, l := range h.Lines {
		switch {
		case strings.HasPrefix(l, "+"):
			r.Lines[i] = "-" + l[1:]
		case strings.HasPrefix(l, "-"):
			r.Lines[i] = "+" + l[1:]
		default:
			r.Lines[i] = l
		}
	}
	return r
}

// applyHunk tries to apply a single hunk to lines. Returns new lines and true on success.
func applyHunk(lines []string, h Hunk, ignoreApplied bool) ([]string, bool) {
	// Special case: new file (OldCount == 0)
	if h.OldCount == 0 {
		return applyAt(lines, h, 0), true
	}

	// Check if hunk is already applied
	if alreadyApplied(lines, h) {
		if ignoreApplied {
			return lines, true
		}
		return lines, false
	}

	// Try the expected position first (OldStart-1). Verify context lines match there.
	pos := h.OldStart - 1
	if pos < 0 {
		pos = 0
	}

	if pos+int(h.OldCount) <= len(lines) || h.OldCount == 0 {
		// Verify: check that context lines (space prefix) match the target
		// and deletion lines (- prefix) exist in the target
		if verifyHunk(lines, h, pos) {
			return applyAt(lines, h, pos), true
		}
	}

	// Try fuzzy matching: slide the hunk position up and down
	for offset := 1; offset <= len(lines); offset++ {
		if pos-offset >= 0 {
			if verifyHunk(lines, h, pos-offset) {
				return applyAt(lines, h, pos-offset), true
			}
		}
		if pos+offset+int(h.OldCount) <= len(lines) {
			if verifyHunk(lines, h, pos+offset) {
				return applyAt(lines, h, pos+offset), true
			}
		}
	}

	// Try reverse hunk
	rh := reverseHunk(h)
	if alreadyApplied(lines, rh) {
		return lines, false
	}
	return lines, false
}

// verifyHunk checks that the context and deletion lines of hunk match target at pos.
func verifyHunk(lines []string, h Hunk, pos int) bool {
	if pos < 0 {
		return false
	}
	// Be lenient about OldCount — allow hunk to reference beyond EOF
	// (BusyBox does this for internal buffering edge cases)
	off := pos
	for _, l := range h.Lines {
		if strings.HasPrefix(l, "-") {
			// Deletion: line should exist in target (or be past EOF — lenient)
			if off < len(lines) && lines[off] != l[1:] {
				return false
			}
			off++
		} else if l == "" || strings.HasPrefix(l, " ") {
			// Context line: must match target (or be past EOF for empty lines)
			expected := ""
			if len(l) > 0 {
				expected = l[1:]
			}
			if off < len(lines) && lines[off] != expected {
				return false
			}
			// If off >= len(lines) and expected is empty, that's OK (implicit empty lines)
			if off >= len(lines) && expected != "" {
				return false
			}
			off++
		}
		// + lines don't consume target lines
	}
	return true
}

// applyAt applies a hunk at the given position in lines.
func applyAt(lines []string, h Hunk, pos int) []string {
	var result []string
	if pos < len(lines) {
		result = append(result, lines[:pos]...)
	} else {
		result = append(result, lines...)
	}

	for _, l := range h.Lines {
		if strings.HasPrefix(l, "+") {
			result = append(result, l[1:])
		} else if l == "" || strings.HasPrefix(l, " ") {
			// Context line: remove leading space, keep empty lines as-is
			if len(l) > 0 {
				result = append(result, l[1:])
			} else {
				result = append(result, "")
			}
		}
		// - lines are skipped
	}

	afterPos := pos + int(h.OldCount)
	if afterPos < len(lines) {
		result = append(result, lines[afterPos:]...)
	}

	return result
}

// findContext finds the position in lines where context lines match.
// startHint is the 0-based line where the hunk should ideally start.
func findContext(lines []string, ctx []string, startHint int) int {
	if len(ctx) == 0 {
		return startHint
	}

	// Try from startHint, with expanding offset
	for offset := 0; offset <= len(lines); offset++ {
		// Try before
		if startHint-offset >= 0 {
			if matchContext(lines[startHint-offset:], ctx) {
				return startHint - offset
			}
		}
		// Try after
		if offset > 0 && startHint+offset < len(lines) {
			if matchContext(lines[startHint+offset:], ctx) {
				return startHint + offset
			}
		}
	}
	return -1
}

// matchContext checks if ctx matches the beginning of lines.
func matchContext(lines []string, ctx []string) bool {
	if len(lines) < len(ctx) {
		return false
	}
	for i, c := range ctx {
		if lines[i] != c {
			return false
		}
	}
	return true
}

// alreadyApplied checks if the hunk's result is already present in the file.
func alreadyApplied(lines []string, h Hunk) bool {
	// Build expected result of applying this hunk
	var expected []string
	for _, l := range h.Lines {
		if strings.HasPrefix(l, "+") {
			expected = append(expected, l[1:])
		} else if !strings.HasPrefix(l, "-") && l != "" {
			expected = append(expected, l[1:])
		}
	}

	if len(expected) == 0 {
		return false
	}

	// Search for expected sequence anywhere in file, starting near OldStart
	start := h.OldStart - 1
	if start < 0 {
		start = 0
	}
	end := start + len(expected) + 10
	if end > len(lines) {
		end = len(lines)
	}

	for pos := start; pos+len(expected) <= end && pos+len(expected) <= len(lines); pos++ {
		match := true
		for i, e := range expected {
			if lines[pos+i] != e {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}

	return false
}

// ---------------------------------------------------------------------------
// Unified diff parser
// ---------------------------------------------------------------------------

// ParsePatch parses a unified diff from r.
func ParsePatch(r io.Reader) ([]Patch, error) {
	scanner := bufio.NewScanner(r)
	var patches []Patch
	var current *Patch
	var currentHunk *Hunk
	var headerLines []string
	var inHunk bool

	for scanner.Scan() {
		line := scanner.Text()

		switch {
		case strings.HasPrefix(line, "--- "):
			if current != nil {
				if currentHunk != nil {
					current.Hunks = append(current.Hunks, *currentHunk)
					currentHunk = nil
				}
				patches = append(patches, *current)
			}
			current = &Patch{OldFile: line[4:]}
			inHunk = false
			headerLines = nil

		case strings.HasPrefix(line, "+++ "):
			if current != nil {
				current.NewFile = line[4:]
			}

		case strings.HasPrefix(line, "@@ "):
			if current != nil {
				if currentHunk != nil {
					current.Hunks = append(current.Hunks, *currentHunk)
				}
				h, err := parseHunkHeader(line)
				if err == nil {
					currentHunk = &h
					inHunk = true
				}
			}

		case inHunk && currentHunk != nil:
			// Collect hunk lines
			if line == "" || strings.HasPrefix(line, " ") ||
				strings.HasPrefix(line, "+") || strings.HasPrefix(line, "-") {
				currentHunk.Lines = append(currentHunk.Lines, line)
			} else if line == `\ No newline at end of file` {
				currentHunk.Lines = append(currentHunk.Lines, line)
			} else if strings.HasPrefix(line, "-- ") {
				// Git version footer — stop collecting but don't close hunk
				// The hunk is complete; next line will trigger a new state
			} else {
				// Other non-hunk line ends the hunk
				inHunk = false
				headerLines = append(headerLines, line)
			}

		default:
			if current == nil {
				headerLines = append(headerLines, line)
			}
		}
	}

	if current != nil {
		if currentHunk != nil {
			current.Hunks = append(current.Hunks, *currentHunk)
		}
		patches = append(patches, *current)
	}

	return patches, scanner.Err()
}

func parseHunkHeader(line string) (Hunk, error) {
	// @@ -oldStart,oldCount +newStart,newCount @@
	line = strings.TrimPrefix(line, "@@ ")
	if idx := strings.Index(line, " @@"); idx >= 0 {
		line = line[:idx]
	}
	parts := strings.Fields(line)
	if len(parts) < 2 {
		return Hunk{}, fmt.Errorf("invalid hunk header: %s", line)
	}

	oldPart := strings.TrimPrefix(parts[0], "-")
	newPart := strings.TrimPrefix(parts[1], "+")

	h := Hunk{OldCount: 1, NewCount: 1}

	if idx := strings.IndexByte(oldPart, ','); idx >= 0 {
		h.OldStart, _ = strconv.Atoi(oldPart[:idx])
		h.OldCount, _ = strconv.Atoi(oldPart[idx+1:])
	} else {
		h.OldStart, _ = strconv.Atoi(oldPart)
	}

	if idx := strings.IndexByte(newPart, ','); idx >= 0 {
		h.NewStart, _ = strconv.Atoi(newPart[:idx])
		h.NewCount, _ = strconv.Atoi(newPart[idx+1:])
	} else {
		h.NewStart, _ = strconv.Atoi(newPart)
	}

	return h, nil
}

// ---------------------------------------------------------------------------
// CLI glue
// ---------------------------------------------------------------------------

func run(args []string, stdin io.Reader, stdout io.Writer) int {
	return patchRun(args, stdout, os.Stderr, os.Stdin)
}

func patchRun(args []string, stdout, stderr io.Writer, stdin io.Reader) int {
	flags, err := common.ParseFlags(args, spec)
	if err != nil {
		fmt.Fprintf(stderr, "patch: %v\n", err)
		return 2
	}

	jsonMode := flags.Has("json")

	stripLevel := 0
	if ps := flags.Get("p"); ps != "" {
		stripLevel, _ = strconv.Atoi(ps)
	}
	reverse := flags.Has("R")
	ignoreApplied := flags.Has("N")

	// Read patch data
	patchData, err := io.ReadAll(stdin)
	if err != nil {
		common.RenderError("patch", 2, "IO", err.Error(), jsonMode, stdout)
		if !jsonMode {
			fmt.Fprintf(stderr, "patch: read stdin: %v\n", err)
		}
		return 2
	}

	// Determine target file: explicit positional arg, or from patch header
	var targetPath string
	if len(flags.Positional) > 0 {
		targetPath = flags.Positional[0]
		// If there's a second positional arg, it's the patch file
		if len(flags.Positional) > 1 {
			patchFile := flags.Positional[1]
			data, ferr := os.ReadFile(patchFile)
			if ferr != nil {
				common.RenderError("patch", 2, "ENOENT", ferr.Error(), jsonMode, stdout)
				if !jsonMode {
					fmt.Fprintf(stderr, "patch: %s: %v\n", patchFile, ferr)
				}
				return 2
			}
			patchData = data
		}
	}

	result, err := Run(patchData, targetPath, stripLevel, reverse, ignoreApplied)

	// Determine exit code: 0 on success, 1 on hunk failure or internal error.
	// The envelope exitCode is always 0 (Render convention); consumers check
	// result.Rejected > 0 or the process exit code for failure detection.
	exitCode := 0
	if err != nil {
		exitCode = 1
	}

	// If Run returned no result at all (parse error), render an error envelope.
	if result == nil {
		common.RenderError("patch", exitCode, "EPATCH", err.Error(), jsonMode, stdout)
		if !jsonMode {
			fmt.Fprintf(stderr, "patch: %v\n", err)
		}
		return exitCode
	}

	// Success or hunk failure — both have structured PatchResult data.
	common.Render("patch", result, jsonMode, stdout, func() {
		msg := "patching file"
		if result.IsNew {
			msg = "creating"
		}
		if result.Applied > 0 || result.Rejected > 0 {
			fmt.Fprintf(stderr, "%s %s\n", msg, result.File)
		}
		if err != nil && result.Msg != "" {
			fmt.Fprint(stderr, result.Msg)
			if !strings.HasSuffix(result.Msg, "\n") {
				fmt.Fprint(stderr, "\n")
			}
		}
	})
	return exitCode
}

func init() {
	dispatch.Register(dispatch.Command{
		Name:  "patch",
		Usage: "patch [-pN] [-R] [-N] [--json] [file [patchfile]] — apply a unified diff to files",
		Run:   run,
	})
}
