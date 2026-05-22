// Package join implements the POSIX join utility — relational database join on sorted files.
package join

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

// JoinResult is the --json output.
type JoinResult struct {
	Records []map[string]string `json:"records"`
}

var spec = common.FlagSpec{
	Defs: []common.FlagDef{
		{Short: "1", Long: "", Type: common.FlagValue},
		{Short: "2", Long: "", Type: common.FlagValue},
		{Short: "t", Long: "", Type: common.FlagValue},
		{Short: "a", Long: "", Type: common.FlagValue},
		{Short: "v", Long: "", Type: common.FlagValue},
		{Short: "o", Long: "", Type: common.FlagValue},
		{Long: "json", Type: common.FlagBool},
	},
}

// line holds a parsed input line.
type line struct {
	raw    string
	key    string
	fields []string
}

// parseLine splits a line by the delimiter and extracts the key at fieldIdx.
func parseLine(text, delim string, fieldIdx int) line {
	fields := strings.SplitN(text, delim, -1)
	// strings.SplitN with -1 handles all cases including empty
	key := ""
	if fieldIdx >= 0 && fieldIdx < len(fields) {
		key = fields[fieldIdx]
	}
	return line{raw: text, key: key, fields: fields}
}

// readFile reads all lines from r, parses them, and returns them sorted by key.
func readFile(r io.Reader, delim string, fieldIdx int) ([]line, error) {
	var lines []line
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		text := scanner.Text()
		lines = append(lines, parseLine(text, delim, fieldIdx))
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return lines, nil
}

// parsePairSpec parses a pair specification like "1.2" or "0".
// Returns (fileNum, fieldNum) or an error.
func parsePairSpec(spec string) (int, int, error) {
	parts := strings.SplitN(spec, ".", 2)
	fileNum, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid file number in %q", spec)
	}
	fieldNum := 1
	if len(parts) == 2 {
		fieldNum, err = strconv.Atoi(parts[1])
		if err != nil {
			return 0, 0, fmt.Errorf("invalid field number in %q", spec)
		}
	}
	return fileNum, fieldNum, nil
}

// formatOutput formats a joined record according to the -o spec.
// keyIdx1 and keyIdx2 are the 0-based field indices used as join keys.
func formatOutput(l1, l2 line, delim, oSpec string, keyIdx1, keyIdx2 int) string {
	if oSpec == "" {
		// Default: all fields from file1 + fields from file2 (excluding key field)
		if l1.raw == "" {
			return l2.raw
		}
		if l2.raw == "" {
			return l1.raw
		}
		var parts []string
		parts = append(parts, l1.fields...)
		// file2: skip the key field
		for idx, f := range l2.fields {
			if idx == keyIdx2 {
				continue
			}
			parts = append(parts, f)
		}
		return strings.Join(parts, delim)
	}

	// Custom output format: space-separated list of FILE.FIELD
	var selected []string
	specs := strings.Fields(oSpec)
	for _, spec := range specs {
		fileNum, fieldNum, err := parsePairSpec(spec)
		if err != nil || (fileNum != 1 && fileNum != 2) || fieldNum < 1 {
			selected = append(selected, "")
			continue
		}

		var fields []string
		if fileNum == 1 {
			fields = l1.fields
		} else {
			fields = l2.fields
		}

		if fieldNum <= len(fields) {
			selected = append(selected, fields[fieldNum-1])
		} else {
			selected = append(selected, "")
		}
	}
	return strings.Join(selected, delim)
}

// Run performs a relational join on two sorted inputs.
func Run(r1, r2 io.Reader, field1, field2 int, delim string, a1, a2 bool, v1, v2 bool, oSpec string) (JoinResult, error) {
	lines1, err := readFile(r1, delim, field1-1)
	if err != nil {
		return JoinResult{}, err
	}
	lines2, err := readFile(r2, delim, field2-1)
	if err != nil {
		return JoinResult{}, err
	}

	var result JoinResult
	i, j := 0, 0
	keyIdx1 := field1 - 1
	keyIdx2 := field2 - 1

	for i < len(lines1) && j < len(lines2) {
		if lines1[i].key < lines2[j].key {
			if a1 || v1 {
				result.Records = append(result.Records, map[string]string{"line": lines1[i].raw})
			}
			i++
		} else if lines1[i].key > lines2[j].key {
			if a2 || v2 {
				result.Records = append(result.Records, map[string]string{"line": lines2[j].raw})
			}
			j++
		} else {
			// Keys match
			keyMatch := lines1[i].key
			startI := i
			for i < len(lines1) && lines1[i].key == keyMatch {
				i++
			}
			startJ := j
			for j < len(lines2) && lines2[j].key == keyMatch {
				j++
			}

			// -v suppresses matched lines entirely
			if v1 || v2 {
				continue
			}

			// Cartesian product for matching keys
			for mi := startI; mi < i; mi++ {
				for mj := startJ; mj < j; mj++ {
					output := formatOutput(lines1[mi], lines2[mj], delim, oSpec, keyIdx1, keyIdx2)
					result.Records = append(result.Records, map[string]string{"line": output})
				}
			}
		}
	}

	// Remaining lines from file1
	if a1 || v1 {
		for ; i < len(lines1); i++ {
			result.Records = append(result.Records, map[string]string{"line": lines1[i].raw})
		}
	}
	// Remaining lines from file2
	if a2 || v2 {
		for ; j < len(lines2); j++ {
			result.Records = append(result.Records, map[string]string{"line": lines2[j].raw})
		}
	}

	return result, nil
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer, cwd string) int {
	flags, err := common.ParseFlags(args, spec)
	if err != nil {
		fmt.Fprintf(os.Stderr, "join: %v\n", err)
		return 2
	}
	jsonMode := flags.Has("json")

	field1 := 1
	field2 := 1
	if flags.Has("1") {
		field1, _ = strconv.Atoi(flags.Get("1"))
	}
	if flags.Has("2") {
		field2, _ = strconv.Atoi(flags.Get("2"))
	}

	delim := " "
	if flags.Has("t") {
		delim = flags.Get("t")
	}
	if delim == "" {
		delim = " "
	}

	a1 := false
	a2 := false
	v1 := false
	v2 := false
	if a := flags.Get("a"); a != "" {
		if a == "1" {
			a1 = true
		} else if a == "2" {
			a2 = true
		}
	}
	if v := flags.Get("v"); v != "" {
		if v == "1" {
			v1 = true
		} else if v == "2" {
			v2 = true
		}
	}
	oSpec := flags.Get("o")

	if len(flags.Positional) < 2 {
		fmt.Fprintln(os.Stderr, "join: missing file operands")
		common.RenderError("join", 1, "EARGS", "missing file operands", jsonMode, stdout)
		return 1
	}

	file1 := flags.Positional[0]
	file2 := flags.Positional[1]

	var r1, r2 io.ReadCloser
	if file1 == "-" {
		r1 = os.Stdin
	} else {
		f, err := os.Open(file1)
		if err != nil {
			fmt.Fprintf(os.Stderr, "join: %v\n", err)
			common.RenderError("join", 1, "EOPEN", err.Error(), jsonMode, stdout)
			return 1
		}
		r1 = f
		defer f.Close()
	}

	if file2 == "-" {
		r2 = os.Stdin
	} else {
		f, err := os.Open(file2)
		if err != nil {
			fmt.Fprintf(os.Stderr, "join: %v\n", err)
			common.RenderError("join", 1, "EOPEN", err.Error(), jsonMode, stdout)
			return 1
		}
		r2 = f
		defer f.Close()
	}

	result, err := Run(r1, r2, field1, field2, delim, a1, a2, v1, v2, oSpec)
	if err != nil {
		fmt.Fprintf(os.Stderr, "join: %v\n", err)
		common.RenderError("join", 1, "EJOIN", err.Error(), jsonMode, stdout)
		return 1
	}

	common.Render("join", result, jsonMode, stdout, func() {
		for _, rec := range result.Records {
			fmt.Fprintln(stdout, rec["line"])
		}
	})
	return 0
}

func init() {
	dispatch.Register(dispatch.Command{
		Name:  "join",
		Usage: "Relational database join on sorted files",
		Run:   run,
	})
}
