// Package tr implements the POSIX tr utility.
package tr

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
)

// TrResult is the structured result for --json mode.
type TrResult struct {
	Lines     []string `json:"lines"`
	LineCount int      `json:"lineCount"`
	BytesIn   int64    `json:"bytesIn"`
	BytesOut  int64    `json:"bytesOut"`
}

var spec = common.FlagSpec{
	Defs: []common.FlagDef{
		{Short: "d", Long: "delete", Type: common.FlagBool},
		{Short: "s", Long: "squeeze-repeats", Type: common.FlagBool},
		{Short: "c", Long: "complement", Type: common.FlagBool},
		{Long: "json", Type: common.FlagBool},
	},
}

// charClasses maps POSIX character class names to their rune sets.
var charClasses = map[string][]rune{
	"[:digit:]":  []rune("0123456789"),
	"[:xdigit:]": []rune("0123456789ABCDEFabcdef"),
	"[:alpha:]":  []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"),
	"[:alnum:]":  []rune("0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"),
	"[:lower:]":  []rune("abcdefghijklmnopqrstuvwxyz"),
	"[:upper:]":  []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ"),
	"[:space:]":  []rune(" \t\n\r\f\v"),
	"[:blank:]":  []rune(" \t"),
	"[:punct:]":  []rune("!\"#$%&'()*+,-./:;<=>?@[\\]^_`{|}~"),
	"[:print:]":  []rune(" !\"#$%&'()*+,-./0123456789:;<=>?@ABCDEFGHIJKLMNOPQRSTUVWXYZ[\\]^_`abcdefghijklmnopqrstuvwxyz{|}~"),
	"[:graph:]":  []rune("!\"#$%&'()*+,-./0123456789:;<=>?@ABCDEFGHIJKLMNOPQRSTUVWXYZ[\\]^_`abcdefghijklmnopqrstuvwxyz{|}~"),
	"[:cntrl:]":  buildCntrl(),
}

func buildCntrl() []rune {
	var r []rune
	for i := 0; i < 32; i++ {
		r = append(r, rune(i))
	}
	r = append(r, 127)
	return r
}

// expandSet expands a set string like a-z, [:digit:] into a map of runes.
func expandSet(s string) map[rune]bool {
	set := make(map[rune]bool)
	// First, expand character classes like [:digit:].
	expanded := s
	for name, runes := range charClasses {
		expanded = strings.ReplaceAll(expanded, name, string(runes))
	}
	runes := []rune(expanded)
	for i := 0; i < len(runes); i++ {
		if i+2 < len(runes) && runes[i+1] == '-' {
			for r := runes[i]; r <= runes[i+2]; r++ {
				set[r] = true
			}
			i += 2
		} else {
			set[runes[i]] = true
		}
	}
	return set
}

// expandSetList expands a set into a slice of runes.
func expandSetList(s string) []rune {
	// First, expand character classes like [:digit:].
	expanded := s
	for name, runes := range charClasses {
		expanded = strings.ReplaceAll(expanded, name, string(runes))
	}
	var list []rune
	runes := []rune(expanded)
	for i := 0; i < len(runes); i++ {
		if i+2 < len(runes) && runes[i+1] == '-' {
			for r := runes[i]; r <= runes[i+2]; r++ {
				list = append(list, r)
			}
			i += 2
		} else {
			list = append(list, runes[i])
		}
	}
	return list
}

func Run(r io.Reader, w io.Writer, set1, set2 string, deleteFlag, squeezeFlag, complementFlag bool) error {
	reader := bufio.NewReader(r)

	s1List := expandSetList(set1)
	s1Map := expandSet(set1)

	var s2List []rune
	if set2 != "" {
		s2List = expandSetList(set2)
	}

	// Translation map
	trans := make(map[rune]rune)
	if !deleteFlag && !squeezeFlag && len(s2List) > 0 {
		for i, r1 := range s1List {
			r2 := s2List[len(s2List)-1]
			if i < len(s2List) {
				r2 = s2List[i]
			}
			trans[r1] = r2
		}
	}

	var lastWrite rune = utf8.RuneError

	for {
		rn, _, err := reader.ReadRune()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		inSet1 := s1Map[rn]
		if complementFlag {
			inSet1 = !inSet1
		}

		if deleteFlag && inSet1 {
			continue // Delete
		}

		outRune := rn
		if inSet1 && !deleteFlag && len(s2List) > 0 {
			if mapped, ok := trans[rn]; ok {
				outRune = mapped
			} else if complementFlag {
				outRune = s2List[len(s2List)-1]
			}
		}

		if squeezeFlag {
			// POSIX says: if -s is given without -d, and outRune is in set2 (or set1 if no set2), squeeze.
			// Simplified: if squeezing and outRune == lastWrite and outRune is in the squeeze set...
			var inSqueezeSet bool
			if len(s2List) > 0 {
				inSqueezeSet = expandSet(set2)[outRune]
			} else {
				inSqueezeSet = inSet1
			}

			if inSqueezeSet && outRune == lastWrite {
				continue
			}
		}

		fmt.Fprint(w, string(outRune))
		lastWrite = outRune
	}

	return nil
}

func run(args []string, out io.Writer) int {
	flags, err := common.ParseFlags(args, spec)
	if err != nil {
		fmt.Fprintf(os.Stderr, "tr: %v\n", err)
		return 2
	}
	deleteFlag := flags.Has("d")
	squeezeFlag := flags.Has("s")
	complementFlag := flags.Has("c")
	jsonMode := flags.Has("json")

	if len(flags.Positional) < 1 {
		if jsonMode {
			common.RenderError("tr", 1, "MISSING", "missing operand", true, out)
		}
		fmt.Fprintln(os.Stderr, "tr: missing operand")
		return 1
	}

	set1 := flags.Positional[0]
	set2 := ""
	if len(flags.Positional) > 1 {
		set2 = flags.Positional[1]
	}

	// Unescape sets (naive)
	set1 = strings.ReplaceAll(set1, "\\n", "\n")
	set1 = strings.ReplaceAll(set1, "\\t", "\t")
	set2 = strings.ReplaceAll(set2, "\\n", "\n")
	set2 = strings.ReplaceAll(set2, "\\t", "\t")

	if jsonMode {
		// Read all stdin, process, capture output
		input, err := io.ReadAll(os.Stdin)
		if err != nil {
			common.RenderError("tr", 1, "READ", err.Error(), true, out)
			return 1
		}
		bytesIn := int64(len(input))

		var buf bytes.Buffer
		err = Run(strings.NewReader(string(input)), &buf, set1, set2, deleteFlag, squeezeFlag, complementFlag)
		if err != nil {
			common.RenderError("tr", 1, "PROCESS", err.Error(), true, out)
			return 1
		}
		bytesOut := int64(buf.Len())

		lines := strings.Split(buf.String(), "\n")
		if len(lines) > 0 && lines[len(lines)-1] == "" {
			lines = lines[:len(lines)-1]
		}

		common.Render("tr", TrResult{
			Lines:     lines,
			LineCount: len(lines),
			BytesIn:   bytesIn,
			BytesOut:  bytesOut,
		}, true, out, func() {})
		return 0
	}

	err = Run(os.Stdin, os.Stdout, set1, set2, deleteFlag, squeezeFlag, complementFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "tr: %v\n", err)
		return 1
	}

	return 0
}

func init() {
	dispatch.Register(dispatch.Command{Name: "tr", Usage: "Translate, squeeze, and/or delete characters", Run: run})
}
