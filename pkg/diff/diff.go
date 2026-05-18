// Package diff implements the POSIX diff utility.
package diff

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
)

// Hunk represents a section of difference between files.
type Hunk struct {
	OldStart int      `json:"oldStart"`
	OldLines int      `json:"oldLines"`
	NewStart int      `json:"newStart"`
	NewLines int      `json:"newLines"`
	Lines    []string `json:"lines"`
}

// DiffResult is the structured result for --json mode.
type DiffResult struct {
	Files  []string `json:"files"`
	Differ bool     `json:"differ"`
	Hunks  []Hunk   `json:"hunks"`
}

var spec = common.FlagSpec{
	Defs: []common.FlagDef{
		{Short: "u", Long: "unified", Type: common.FlagBool},
		{Short: "U", Long: "unified-context", Type: common.FlagValue},
		{Short: "b", Long: "ignore-space-change", Type: common.FlagBool},
		{Short: "w", Long: "ignore-all-space", Type: common.FlagBool},
		{Short: "B", Long: "ignore-blank-lines", Type: common.FlagBool},
		{Short: "q", Long: "brief", Type: common.FlagBool},
		{Short: "r", Long: "recursive", Type: common.FlagBool},
		{Short: "N", Long: "new-file", Type: common.FlagBool},
		{Long: "json", Type: common.FlagBool},
	},
}

type diffOp int

const (
	opEq diffOp = iota
	opDel
	opIns
	opIgnoredDel
	opIgnoredIns
)

type diffItem struct {
	op   diffOp
	text string
}

func normalizeSpace(lines []string) []string {
	out := make([]string, len(lines))
	for i, l := range lines {
		l = strings.TrimRight(l, " \t\r\n")
		var b strings.Builder
		inSpace := false
		for j := 0; j < len(l); j++ {
			c := l[j]
			if c == ' ' || c == '\t' || c == '\r' || c == '\n' {
				if !inSpace {
					b.WriteByte(' ')
					inSpace = true
				}
			} else {
				b.WriteByte(c)
				inSpace = false
			}
		}
		out[i] = b.String()
	}
	return out
}

func stripAllSpace(lines []string) []string {
	out := make([]string, len(lines))
	for i, l := range lines {
		var b strings.Builder
		for j := 0; j < len(l); j++ {
			c := l[j]
			if c != ' ' && c != '\t' && c != '\r' && c != '\n' {
				b.WriteByte(c)
			}
		}
		out[i] = b.String()
	}
	return out
}

func filterBlankLineChanges(script []diffItem) []diffItem {
	for i := 0; i < len(script); {
		if script[i].op == opDel || script[i].op == opIns {
			// Find end of block
			j := i
			allBlank := true
			for j < len(script) && (script[j].op == opDel || script[j].op == opIns) {
				if strings.TrimSpace(script[j].text) != "" {
					allBlank = false
				}
				j++
			}
			if allBlank {
				for k := i; k < j; k++ {
					if script[k].op == opDel {
						script[k].op = opIgnoredDel
					} else {
						script[k].op = opIgnoredIns
					}
				}
			}
			i = j
		} else {
			i++
		}
	}
	return script
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Diff computes the differences between two sequences of strings using an O(ND) algorithm.
func Diff(a, b []string) []diffItem {
	n := len(a)
	m := len(b)
	maxD := n + m

	if maxD == 0 {
		return nil
	}

	v := make([]int, 2*maxD+1)
	trace := make([][]int, 0, maxD+1)

	var d, k, x, y int

Outer:
	for d = 0; d <= maxD; d++ {
		for k = -d; k <= d; k += 2 {
			if k == -d || (k != d && v[maxD+k-1] < v[maxD+k+1]) {
				x = v[maxD+k+1]
			} else {
				x = v[maxD+k-1] + 1
			}
			y = x - k

			for x < n && y < m && a[x] == b[y] {
				x++
				y++
			}
			v[maxD+k] = x

			if x >= n && y >= m {
				tr := make([]int, 2*maxD+1)
				copy(tr, v)
				trace = append(trace, tr)
				break Outer
			}
		}
		tr := make([]int, 2*maxD+1)
		copy(tr, v)
		trace = append(trace, tr)
	}

	// Backtrack
	var script []diffItem
	x = n
	y = m

	for d > 0 {
		k = x - y
		var prevK int
		if k == -d || (k != d && trace[d-1][maxD+k-1] < trace[d-1][maxD+k+1]) {
			prevK = k + 1
		} else {
			prevK = k - 1
		}
		prevX := trace[d-1][maxD+prevK]
		var startX int
		var isDel bool
		if prevK == k - 1 { 
			startX = prevX + 1
			isDel = true
		} else { 
			startX = prevX
			isDel = false
		}

		for x > startX {
			x--
			y--
			script = append(script, diffItem{op: opEq, text: a[x]})
		}
		
		if isDel {
			x--
			script = append(script, diffItem{op: opDel, text: a[x]})
		} else {
			y--
			script = append(script, diffItem{op: opIns, text: b[y]})
		}
		
		d--
	}

	for x > 0 && y > 0 {
		x--
		y--
		script = append(script, diffItem{op: opEq, text: a[x]})
	}

	// Reverse script
	for i, j := 0, len(script)-1; i < j; i, j = i+1, j-1 {
		script[i], script[j] = script[j], script[i]
	}

	return script
}

// BuildHunks builds unified diff hunks from a diff script with a given number of context lines.
func BuildHunks(script []diffItem, context int, newline1, newline2 bool) []Hunk {
	var hunks []Hunk
	
	i := 0
	oldLine := 1
	newLine := 1

	for i < len(script) {
		// Skip initial equal lines and ignored lines
		for i < len(script) && (script[i].op == opEq || script[i].op == opIgnoredDel || script[i].op == opIgnoredIns) {
			if script[i].op == opEq || script[i].op == opIgnoredDel {
				oldLine++
			}
			if script[i].op == opEq || script[i].op == opIgnoredIns {
				newLine++
			}
			i++
		}
		if i >= len(script) {
			break
		}

		// Found a difference, backtrack for context
		startIdx := i - context
		if startIdx < 0 {
			startIdx = 0
		}
		
		// Find end of current difference block
		endIdx := i
		for endIdx < len(script) {
			nextDiff := endIdx
			// Skip diff ops
			for nextDiff < len(script) && (script[nextDiff].op == opDel || script[nextDiff].op == opIns) {
				nextDiff++
			}

			// Find next equal/ignored block
			eqCount := 0
			for nextDiff < len(script) && (script[nextDiff].op == opEq || script[nextDiff].op == opIgnoredDel || script[nextDiff].op == opIgnoredIns) {
				eqCount++
				nextDiff++
			}
			
			if nextDiff >= len(script) {
				if eqCount > context {
					endIdx = nextDiff - eqCount + context
				} else {
					endIdx = len(script)
				}
				break
			}
			
			if eqCount > 2*context {
				eqStart := nextDiff - eqCount
				endIdx = eqStart + context
				if endIdx > len(script) {
					endIdx = len(script)
				}
				break
			}
			endIdx = nextDiff
		}

		if endIdx > len(script) {
			endIdx = len(script)
		}

		hunk := Hunk{
			OldStart: 1, 
			NewStart: 1, 
		}

		hunkOld := 1
		hunkNew := 1
		for j := 0; j < startIdx; j++ {
			if script[j].op == opEq || script[j].op == opDel || script[j].op == opIgnoredDel {
				hunkOld++
			}
			if script[j].op == opEq || script[j].op == opIns || script[j].op == opIgnoredIns {
				hunkNew++
			}
		}

		hunk.OldStart = hunkOld
		hunk.NewStart = hunkNew

		for j := startIdx; j < endIdx; j++ {
			switch script[j].op {
			case opEq:
				hunk.Lines = append(hunk.Lines, " "+script[j].text)
				hunk.OldLines++
				hunk.NewLines++
			case opDel:
				hunk.Lines = append(hunk.Lines, "-"+script[j].text)
				hunk.OldLines++
			case opIns:
				hunk.Lines = append(hunk.Lines, "+"+script[j].text)
				hunk.NewLines++
			case opIgnoredDel:
				hunk.Lines = append(hunk.Lines, "-"+script[j].text)
				hunk.OldLines++
			case opIgnoredIns:
				hunk.Lines = append(hunk.Lines, "+"+script[j].text)
				hunk.NewLines++
			}
		}

		hunks = append(hunks, hunk)
		i = endIdx
		
		// Update oldLine and newLine to the end of the hunk
		oldLine = hunkOld + hunk.OldLines
		newLine = hunkNew + hunk.NewLines
	}

	return hunks
}

// GenerateDiff compares two file contents and produces hunks and a differ boolean.
func GenerateDiff(content1, content2 string, contextLines int, ignoreSpace, ignoreAllSpace, ignoreBlankLines bool) (bool, []Hunk) {
	lines1 := strings.Split(content1, "\n")
	if len(lines1) > 0 && lines1[len(lines1)-1] == "" {
		lines1 = lines1[:len(lines1)-1]
	}
	if content1 == "" {
		lines1 = nil
	}

	lines2 := strings.Split(content2, "\n")
	if len(lines2) > 0 && lines2[len(lines2)-1] == "" {
		lines2 = lines2[:len(lines2)-1]
	}
	if content2 == "" {
		lines2 = nil
	}
	
	// Normalize for -b (space-change) or -w (all-space)
	compLines1 := lines1
	compLines2 := lines2
	if ignoreAllSpace {
		compLines1 = stripAllSpace(lines1)
		compLines2 = stripAllSpace(lines2)
	} else if ignoreSpace {
		compLines1 = normalizeSpace(lines1)
		compLines2 = normalizeSpace(lines2)
	}

	// For -B, we want to ignore changes that consist entirely of blank lines.
	// Easiest approach is to run the diff on compLines1 and compLines2,
	// then post-process the script.
	script := Diff(compLines1, compLines2)
	
	// Post-process to fix up the text fields to be the original lines,
	// since we compared the normalized ones.
	x := 0
	y := 0
	for i := range script {
		if script[i].op == opEq {
			script[i].text = lines1[x]
			x++
			y++
		} else if script[i].op == opDel {
			script[i].text = lines1[x]
			x++
		} else if script[i].op == opIns {
			script[i].text = lines2[y]
			y++
		}
	}
	
	if ignoreBlankLines {
		script = filterBlankLineChanges(script)
	}
	
	// If the file does not end in a newline, strings.Split produces a trailing empty string,
	// but we might want to preserve exact lines. We'll leave it as is for simplicity.
	// If the string is empty, split returns [""] which is one line, except if it's really empty.
	if content1 == "" {
		lines1 = nil
	}
	if content2 == "" {
		lines2 = nil
	}

	// Now script contains the true differences using original lines
	differ := false
	for _, item := range script {
		if item.op == opDel || item.op == opIns {
			differ = true
			break
		}
	}

	// Track whether files end in a newline (for \ No newline marker)
	newline1 := content1 != "" && content1[len(content1)-1] == '\n'
	newline2 := content2 != "" && content2[len(content2)-1] == '\n'

	hunks := BuildHunks(script, contextLines, newline1, newline2)
	return differ, hunks
}

func run(args []string, out io.Writer) int {
	flags, err := common.ParseFlags(args, spec)
	if err != nil {
		fmt.Fprintf(os.Stderr, "diff: %v\n", err)
		return 2
	}

	jsonMode := flags.Has("json")
	files := flags.Positional
	
	contextLines := 3
	if flags.Has("U") {
		c, err := strconv.Atoi(flags.Get("U"))
		if err == nil && c >= 0 {
			contextLines = c
		}
	}

	if len(files) != 2 {
		common.RenderError("diff", 2, "USAGE", "missing operand", jsonMode, out)
		if !jsonMode {
			fmt.Fprintf(os.Stderr, "diff: missing operand\n")
		}
		return 2
	}

	// Directory diff: if -r is set and both paths exist as dirs.
	recursive := flags.Has("r")
	treatNew := flags.Has("N")
	ignoreSpace := flags.Has("b") || flags.Has("w")
	ignoreAllSpace := flags.Has("w")
	ignoreBlankLines := flags.Has("B")
	if recursive {
		s1, e1 := os.Stat(files[0])
		s2, e2 := os.Stat(files[1])
		if e1 == nil && e2 == nil && s1.IsDir() && s2.IsDir() {
			return diffDirs(files[0], files[1], contextLines, ignoreSpace, ignoreAllSpace, ignoreBlankLines, treatNew, jsonMode, out)
		}
		// One is dir, other is file: look for file's basename inside the dir.
		// E.g., "diff -ur dir1 dir2/subdir/-" → diff "dir1/-" vs "dir2/subdir/-"
		if e1 == nil && s1.IsDir() && e2 == nil && !s2.IsDir() {
			files[0] = files[0] + "/" + filepath.Base(files[1])
		} else if e1 == nil && !s1.IsDir() && e2 == nil && s2.IsDir() {
			files[1] = files[1] + "/" + filepath.Base(files[0])
		}
	}

	var b1, b2 []byte
	if files[0] == files[1] {
		// Optimization: if both files are the same (e.g. diff - -), they are identical.
		// Don't read anything.
		common.Render("diff", DiffResult{
			Files:  []string{files[0], files[1]},
			Differ: false,
			Hunks:  nil,
		}, jsonMode, out, func() {})
		return 0
	}

	if files[0] == "-" {
		b1, err = io.ReadAll(os.Stdin)
	} else {
		b1, err = os.ReadFile(files[0])
	}
	if err != nil {
		common.RenderError("diff", 2, "IO", err.Error(), jsonMode, out)
		if !jsonMode {
			fmt.Fprintf(os.Stderr, "diff: %v\n", err)
		}
		return 2
	}
	if files[1] == "-" {
		b2, err = io.ReadAll(os.Stdin)
	} else {
		b2, err = os.ReadFile(files[1])
	}
	if err != nil {
		common.RenderError("diff", 2, "IO", err.Error(), jsonMode, out)
		if !jsonMode {
			fmt.Fprintf(os.Stderr, "diff: %v\n", err)
		}
		return 2
	}

	differ, hunks := GenerateDiff(string(b1), string(b2), contextLines, ignoreSpace, ignoreAllSpace, ignoreBlankLines)

	res := DiffResult{
		Files:  []string{files[0], files[1]},
		Differ: differ,
		Hunks:  hunks,
	}

	newline1 := len(b1) == 0 || b1[len(b1)-1] == '\n'
	newline2 := len(b2) == 0 || b2[len(b2)-1] == '\n'

	if differ {
		common.Render("diff", res, jsonMode, out, func() {
			if flags.Has("q") {
				fmt.Fprintf(out, "Files %s and %s differ\n", files[0], files[1])
			} else if flags.Has("u") || flags.Has("U") {
				fmt.Fprintf(out, "--- %s\n+++ %s\n", files[0], files[1])
				for _, h := range hunks {
					// POSIX unified diff hunk range: omit count if 1, use "start,0" if 0
					oldStr := fmt.Sprintf("%d,%d", h.OldStart, h.OldLines)
					if h.OldLines == 1 {
						oldStr = fmt.Sprintf("%d", h.OldStart)
					} else if h.OldLines == 0 {
						oldStr = fmt.Sprintf("%d,0", h.OldStart-1)
					}
					newStr := fmt.Sprintf("%d,%d", h.NewStart, h.NewLines)
					if h.NewLines == 1 {
						newStr = fmt.Sprintf("%d", h.NewStart)
					} else if h.NewLines == 0 {
						newStr = fmt.Sprintf("%d,0", h.NewStart-1)
					}
					fmt.Fprintf(out, "@@ -%s +%s @@\n", oldStr, newStr)
					for i, l := range h.Lines {
						fmt.Fprintln(out, l)
						// Emit no-newline marker after last line of a file if needed
						isLastLine := i == len(h.Lines)-1
						if isLastLine {
							isDelLine := len(l) > 0 && l[0] == '-'
							isInsLine := len(l) > 0 && l[0] == '+'
							if isDelLine && !newline1 {
								fmt.Fprintln(out, `\ No newline at end of file`)
							} else if isInsLine && !newline2 {
								fmt.Fprintln(out, `\ No newline at end of file`)
							}
						}
					}
				}
			} else {
				fmt.Fprintf(out, "Files %s and %s differ\n", files[0], files[1])
			}
		})
		return 1
	}

	common.Render("diff", res, jsonMode, out, func() {})
	return 0
}

// joinPreserving preserves the raw dir string, only adding a separator if needed.
func joinPreserving(dir, rel string) string {
	if strings.HasSuffix(dir, "/") {
		return dir + rel
	}
	return dir + "/" + rel
}

// diffDirs recursively compares two directories and outputs unified diffs.
func diffDirs(dir1, dir2 string, contextLines int, ignoreSpace, ignoreAllSpace, ignoreBlankLines, treatNew, jsonMode bool, out io.Writer) int {
	// Collect all relative paths from both directories.
	paths1 := make(map[string]string) // rel → abs
	paths2 := make(map[string]string)

	filepath.WalkDir(dir1, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(dir1, p)
		paths1[rel] = p
		return nil
	})
	filepath.WalkDir(dir2, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(dir2, p)
		paths2[rel] = p
		return nil
	})

	// Collect all unique relative paths, sorted.
	allPaths := make(map[string]bool)
	for p := range paths1 {
		allPaths[p] = true
	}
	for p := range paths2 {
		allPaths[p] = true
	}
	sorted := make([]string, 0, len(allPaths))
	for p := range allPaths {
		sorted = append(sorted, p)
	}
	sort.Strings(sorted)

	exitCode := 0
	for _, rel := range sorted {
		p1, in1 := paths1[rel]
		p2, in2 := paths2[rel]

		// With -N, treat missing files as present (empty).
		// Check for non-regular files BEFORE checking existence.
		if in1 && !in2 {
			fi1, _ := os.Lstat(p1)
			if fi1 != nil && !fi1.Mode().IsRegular() {
				if treatNew {
					fmt.Fprintf(out, "File %s is not a regular file or directory and was skipped\n", joinPreserving(dir1, rel))
					if exitCode == 0 {
						exitCode = 1
					}
					continue
				} else {
					fmt.Fprintf(out, "Only in %s: %s\n", dir1, rel)
					exitCode = 1
					continue
				}
			}
		}
		if !in1 && in2 {
			fi2, _ := os.Lstat(p2)
			if fi2 != nil && !fi2.Mode().IsRegular() {
				if treatNew {
					fmt.Fprintf(out, "File %s is not a regular file or directory and was skipped\n", joinPreserving(dir2, rel))
					if exitCode == 0 {
						exitCode = 1
					}
					continue
				} else {
					fmt.Fprintf(out, "Only in %s: %s\n", dir2, rel)
					exitCode = 1
					continue
				}
			}
		}

		if in1 && in2 {
			// Both exist: diff them.
			// Skip non-regular files.
			fi1, _ := os.Lstat(p1)
			fi2, _ := os.Lstat(p2)
			if fi1 != nil && !fi1.Mode().IsRegular() {
				fmt.Fprintf(out, "File %s is not a regular file or directory and was skipped\n", joinPreserving(dir1, rel))
				if exitCode == 0 {
					exitCode = 1
				}
				continue
			}
			if fi2 != nil && !fi2.Mode().IsRegular() {
				fmt.Fprintf(out, "File %s is not a regular file or directory and was skipped\n", joinPreserving(dir2, rel))
				if exitCode == 0 {
					exitCode = 1
				}
				continue
			}
			b1, _ := os.ReadFile(p1)
			b2, _ := os.ReadFile(p2)
			differ, hunks := GenerateDiff(string(b1), string(b2), contextLines, ignoreSpace, ignoreAllSpace, ignoreBlankLines)
			if differ || !treatNew {
				if differ {
					exitCode = 1
					fmt.Fprintf(out, "--- %s\n+++ %s\n", joinPreserving(dir1, rel), joinPreserving(dir2, rel))
					newline1 := len(b1) == 0 || b1[len(b1)-1] == '\n'
					newline2 := len(b2) == 0 || b2[len(b2)-1] == '\n'
					for _, h := range hunks {
						oldStr := fmt.Sprintf("%d,%d", h.OldStart, h.OldLines)
						if h.OldLines == 1 {
							oldStr = fmt.Sprintf("%d", h.OldStart)
						} else if h.OldLines == 0 {
							oldStr = fmt.Sprintf("%d,0", h.OldStart-1)
						}
						newStr := fmt.Sprintf("%d,%d", h.NewStart, h.NewLines)
						if h.NewLines == 1 {
							newStr = fmt.Sprintf("%d", h.NewStart)
						} else if h.NewLines == 0 {
							newStr = fmt.Sprintf("%d,0", h.NewStart-1)
						}
						fmt.Fprintf(out, "@@ -%s +%s @@\n", oldStr, newStr)
						for i, l := range h.Lines {
							fmt.Fprintln(out, l)
							isLastLine := i == len(h.Lines)-1
							if isLastLine {
								isDelLine := len(l) > 0 && l[0] == '-'
								isInsLine := len(l) > 0 && l[0] == '+'
								if isDelLine && !newline1 {
									fmt.Fprintln(out, `\ No newline at end of file`)
								} else if isInsLine && !newline2 {
									fmt.Fprintln(out, `\ No newline at end of file`)
								}
							}
						}
					}
				}
			}
		} else if in1 && !in2 {
			if treatNew {
				// -N: treat missing file as empty, diff against empty.
				b1, _ := os.ReadFile(p1)
				differ, hunks := GenerateDiff(string(b1), "", contextLines, ignoreSpace, ignoreAllSpace, ignoreBlankLines)
				if differ || !treatNew {
					if differ {
						exitCode = 1
					}
					fmt.Fprintf(out, "--- %s\n+++ %s\n", joinPreserving(dir1, rel), joinPreserving(dir2, rel))
					newline1 := len(b1) == 0 || b1[len(b1)-1] == '\n'
					newline2 := true
					for _, h := range hunks {
						oldStr := fmt.Sprintf("%d,%d", h.OldStart, h.OldLines)
						if h.OldLines == 1 {
							oldStr = fmt.Sprintf("%d", h.OldStart)
						} else if h.OldLines == 0 {
							oldStr = fmt.Sprintf("%d,0", h.OldStart-1)
						}
						newStr := fmt.Sprintf("%d,%d", h.NewStart, h.NewLines)
						if h.NewLines == 1 {
							newStr = fmt.Sprintf("%d", h.NewStart)
						} else if h.NewLines == 0 {
							newStr = fmt.Sprintf("%d,0", h.NewStart-1)
						}
						fmt.Fprintf(out, "@@ -%s +%s @@\n", oldStr, newStr)
						for i, l := range h.Lines {
							fmt.Fprintln(out, l)
							isLastLine := i == len(h.Lines)-1
							if isLastLine {
								isDelLine := len(l) > 0 && l[0] == '-'
								isInsLine := len(l) > 0 && l[0] == '+'
								if isDelLine && !newline1 {
									fmt.Fprintln(out, `\ No newline at end of file`)
								} else if isInsLine && !newline2 {
									fmt.Fprintln(out, `\ No newline at end of file`)
								}
							}
						}
					}
				}
			} else {
				fmt.Fprintf(out, "Only in %s: %s\n", dir1, rel)
				exitCode = 1
			}
		} else if !in1 && in2 {
			if treatNew {
				// -N: treat missing file as empty, diff against empty.
				b2, _ := os.ReadFile(p2)
				differ, hunks := GenerateDiff("", string(b2), contextLines, ignoreSpace, ignoreAllSpace, ignoreBlankLines)
				if differ || !treatNew {
					if differ {
						exitCode = 1
					}
					fmt.Fprintf(out, "--- %s\n+++ %s\n", joinPreserving(dir1, rel), joinPreserving(dir2, rel))
					newline1 := true
					newline2 := len(b2) == 0 || b2[len(b2)-1] == '\n'
					for _, h := range hunks {
						oldStr := fmt.Sprintf("%d,%d", h.OldStart, h.OldLines)
						if h.OldLines == 1 {
							oldStr = fmt.Sprintf("%d", h.OldStart)
						} else if h.OldLines == 0 {
							oldStr = fmt.Sprintf("%d,0", h.OldStart-1)
						}
						newStr := fmt.Sprintf("%d,%d", h.NewStart, h.NewLines)
						if h.NewLines == 1 {
							newStr = fmt.Sprintf("%d", h.NewStart)
						} else if h.NewLines == 0 {
							newStr = fmt.Sprintf("%d,0", h.NewStart-1)
						}
						fmt.Fprintf(out, "@@ -%s +%s @@\n", oldStr, newStr)
						for i, l := range h.Lines {
							fmt.Fprintln(out, l)
							isLastLine := i == len(h.Lines)-1
							if isLastLine {
								isDelLine := len(l) > 0 && l[0] == '-'
								isInsLine := len(l) > 0 && l[0] == '+'
								if isDelLine && !newline1 {
									fmt.Fprintln(out, `\ No newline at end of file`)
								} else if isInsLine && !newline2 {
									fmt.Fprintln(out, `\ No newline at end of file`)
								}
							}
						}
					}
				}
			} else {
				fmt.Fprintf(out, "Only in %s: %s\n", dir2, rel)
				exitCode = 1
			}
		}
	}
	return exitCode
}

func init() {
	dispatch.Register(dispatch.Command{
		Name:  "diff",
		Usage: "Compare files line by line",
		Run:   run,
	})
}
