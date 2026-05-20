// Package sort implements the POSIX sort utility.
package sort

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
)

// SortResult is the --json output for sort.
type SortResult struct {
	Lines []string `json:"lines"`
	Count int      `json:"count"`
}

var spec = common.FlagSpec{
	Defs: []common.FlagDef{
		{Short: "r", Long: "reverse", Type: common.FlagBool},
		{Short: "n", Long: "numeric-sort", Type: common.FlagBool},
		{Short: "u", Long: "unique", Type: common.FlagBool},
		{Short: "k", Long: "key", Type: common.FlagValue},
		{Short: "t", Long: "field-separator", Type: common.FlagValue},
		{Short: "o", Long: "output", Type: common.FlagValue},
		{Short: "s", Long: "stable", Type: common.FlagBool},
		{Short: "z", Long: "zero-terminated", Type: common.FlagBool},
		{Short: "h", Long: "human-numeric-sort", Type: common.FlagBool},
		{Short: "M", Long: "month-sort", Type: common.FlagBool},
		{Long: "json", Type: common.FlagBool},
	},
}

type keySpec struct {
	startField int
	startChar  int
	endField   int
	endChar    int
	numeric    bool
	reverse    bool
	month      bool
	human      bool
}

type lineItem struct {
	original  string
	keys      []string
	numVals   []float64
	validNum  []bool
	humanVals []humanVal
}

var monthOrder = map[string]int{
	"jan": 1, "january": 1, "feb": 2, "february": 2,
	"mar": 3, "march": 3, "apr": 4, "april": 4,
	"may": 5, "jun": 6, "june": 6,
	"jul": 7, "july": 7, "aug": 8, "august": 8,
	"sep": 9, "september": 9, "oct": 10, "october": 10,
	"nov": 11, "november": 11, "dec": 12, "december": 12,
}

var suffixRank = map[byte]int{
	'K': 1, 'k': 2,
	'M': 3,
	'G': 4,
	'T': 5,
	'P': 6,
	'E': 7,
	'Z': 8,
	'Y': 9,
}

type humanVal struct {
	num       float64
	hasSuffix bool
	suffix    int // rank
	orig      string
}

func parseHumanVal(s string) humanVal {
	s = strings.TrimSpace(s)
	if s == "" {
		return humanVal{}
	}
	// Find where the numeric prefix ends.
	numEnd := 0
	dotSeen := false
	for numEnd < len(s) {
		c := s[numEnd]
		if c >= '0' && c <= '9' {
			numEnd++
		} else if c == '.' && !dotSeen {
			dotSeen = true
			numEnd++
		} else {
			break
		}
	}
	numStr := s[:numEnd]
	suffixChar := byte(0)
	if numEnd < len(s) {
		suffixChar = s[numEnd]
	}
	v, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return humanVal{orig: s}
	}
	if rank, ok := suffixRank[suffixChar]; ok {
		return humanVal{num: v, hasSuffix: true, suffix: rank, orig: s}
	}
	return humanVal{num: v, hasSuffix: false, orig: s}
}

// parseNumericPrefix parses a leading numeric prefix from s (like C's strtod).
// Returns the parsed value and true if at least one digit was found.
func parseNumericPrefix(s string) (float64, bool) {
	s = strings.TrimLeft(s, " \t")
	if s == "" {
		return 0, false
	}
	// Find end of numeric prefix: digits, optional decimal point, optional sign.
	i := 0
	if i < len(s) && (s[i] == '+' || s[i] == '-') {
		i++
	}
	hasDigit := false
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		hasDigit = true
		i++
	}
	if i < len(s) && s[i] == '.' {
		i++
		for i < len(s) && s[i] >= '0' && s[i] <= '9' {
			hasDigit = true
			i++
		}
	}
	if !hasDigit {
		return 0, false
	}
	v, err := strconv.ParseFloat(s[:i], 64)
	if err != nil {
		return 0, false
	}
	return v, true
}

func parseHuman(s string) (float64, bool) {
	hv := parseHumanVal(s)
	if hv.hasSuffix {
		// Encode as: num*10 + suffixRank. This keeps ordering correct.
		// Actually we can't combine them like this for sorting because the comparison
		// needs to know suffix vs no-suffix. Return a marker.
		return hv.num, true
	}
	return hv.num, hv.hasSuffix
}

func parseMonth(s string) (int, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}
	lower := strings.ToLower(s)
	if m, ok := monthOrder[lower]; ok {
		return m, true
	}
	if len(lower) >= 3 {
		if m, ok := monthOrder[lower[:3]]; ok {
			return m, true
		}
	}
	return 0, false
}

func extractKey(line string, ks keySpec, delimiter string) string {
	var parts []string
	if delimiter != "" {
		parts = strings.Split(line, delimiter)
	} else {
		parts = strings.Fields(line)
	}
	if ks.startField <= 0 {
		return line
	}
	var keyParts []string
	end := ks.endField
	if end <= 0 || end < ks.startField {
		end = ks.startField
	}
	for f := ks.startField; f <= end && f <= len(parts); f++ {
		keyParts = append(keyParts, parts[f-1])
	}
	key := strings.Join(keyParts, delimiter)
	if delimiter == "" {
		key = strings.Join(keyParts, " ")
	}
	startChar := ks.startChar
	if startChar <= 0 && ks.endChar > 0 {
		startChar = 1
	}
	if startChar > 0 && len(key) > 0 {
		runes := []rune(key)
		start := startChar - 1
		if start >= len(runes) {
			return ""
		}
		endChar := len(runes)
		if ks.endChar > 0 && ks.endChar <= len(runes) {
			endChar = ks.endChar
		}
		key = string(runes[start:endChar])
	}
	return key
}

func parseKeySpec(kStr string, globalNumeric, globalReverse, globalMonth, globalHuman bool) []keySpec {
	var specs []keySpec
	ks := keySpec{}
	rest := kStr

	// Find the comma (if any) separating start and end field specs.
	commaIdx := strings.IndexByte(rest, ',')

	// Find where flags start in the start field spec (or the whole thing if no comma).
	searchEnd := len(rest)
	if commaIdx >= 0 {
		searchEnd = commaIdx
	}
	flagIdx := strings.IndexAny(rest[:searchEnd], "nrMhb")
	fieldEnd := searchEnd
	if flagIdx >= 0 {
		fieldEnd = flagIdx
	}
	ks.startField, ks.startChar = parseFieldChar(rest[:fieldEnd])
	rest = rest[fieldEnd:]

	if commaIdx >= 0 {
		// Skip the comma, then parse end field + flags.
		rest = rest[1:] // skip the ',' we already passed
		// Find flags in the end field spec.
		flagIdx2 := strings.IndexAny(rest, "nrMhb")
		if flagIdx2 >= 0 {
			ks.endField, ks.endChar = parseFieldChar(rest[:flagIdx2])
			rest = rest[flagIdx2:]
		} else {
			ks.endField, ks.endChar = parseFieldChar(rest)
			rest = ""
		}
	}

	for _, ch := range rest {
		switch ch {
		case 'n':
			ks.numeric = true
		case 'r':
			ks.reverse = true
		case 'M':
			ks.month = true
		case 'h':
			ks.human = true
		}
	}
	if globalNumeric && !ks.numeric && !ks.month && !ks.human {
		ks.numeric = true
	}
	// Note: globalReverse is NOT applied to ks.reverse here.
	// Global -r only reverses the tiebreaker (last-resort comparison),
	// while per-key r (in -k M,Nnr) reverses the entire key comparison.
	if globalMonth {
		ks.month = true
	}
	if globalHuman {
		ks.human = true
	}
	specs = append(specs, ks)
	return specs
}

func parseFieldChar(s string) (field, char int) {
	field = 1
	if s == "" {
		return
	}
	if dot := strings.IndexByte(s, '.'); dot >= 0 {
		if f, err := strconv.Atoi(s[:dot]); err == nil {
			field = f
		}
		if c, err := strconv.Atoi(s[dot+1:]); err == nil {
			char = c
		}
	} else {
		if f, err := strconv.Atoi(s); err == nil {
			field = f
		}
	}
	return
}

func parseLines(r io.Reader, keySpecs []keySpec, delimiter string, zeroTerminated bool) ([]lineItem, error) {
	// 256MB total input cap to prevent OOM (sort reads entire input into memory)
	r = io.LimitReader(r, 256*1024*1024)
	var items []lineItem
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024)
	if zeroTerminated {
		scanner.Split(scanNUL)
	}
	for scanner.Scan() {
		text := scanner.Text()
		item := lineItem{original: text}
		if len(keySpecs) == 0 {
			item.keys = []string{text}
			item.numVals = make([]float64, 1)
			item.validNum = make([]bool, 1)
		} else {
			item.keys = make([]string, len(keySpecs))
			item.numVals = make([]float64, len(keySpecs))
			item.validNum = make([]bool, len(keySpecs))
			item.humanVals = make([]humanVal, len(keySpecs))
			for i, ks := range keySpecs {
				key := extractKey(text, ks, delimiter)
				item.keys[i] = key
				if ks.human {
					item.humanVals[i] = parseHumanVal(key)
				} else if ks.numeric {
					if v, ok := parseNumericPrefix(key); ok {
						item.numVals[i] = v
						item.validNum[i] = true
					}
				} else if ks.month {
					if v, ok := parseMonth(key); ok {
						item.numVals[i] = float64(v)
						item.validNum[i] = true
					}
				}
			}
		}
		items = append(items, item)
	}
	return items, scanner.Err()
}

func scanNUL(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if i := bytes.IndexByte(data, 0); i >= 0 {
		return i + 1, data[:i], nil
	}
	if atEOF {
		return len(data), data, nil
	}
	return 0, nil, nil
}

func Run(items []lineItem, keySpecs []keySpec, reverse, numeric, unique, month, human, stable, tieReverse bool) []string {
	// If keySpecs are empty but global options are set, fill computed values.
	if len(keySpecs) == 0 && (numeric || month || human) {
		keySpecs = []keySpec{{numeric: numeric, reverse: reverse, month: month, human: human}}
		for i := range items {
			if len(items[i].numVals) == 0 {
				items[i].numVals = make([]float64, 1)
				items[i].validNum = make([]bool, 1)
				items[i].humanVals = make([]humanVal, 1)
			}
			key := items[i].original
			if len(items[i].keys) > 0 {
				key = items[i].keys[0]
			}
			if human {
				items[i].humanVals[0] = parseHumanVal(key)
			}
			if numeric {
				if v, ok := parseNumericPrefix(key); ok {
					items[i].numVals[0] = v
					items[i].validNum[0] = true
				}
			}
			if month {
				if v, ok := parseMonth(key); ok {
					items[i].numVals[0] = float64(v)
					items[i].validNum[0] = true
				}
			}
		}
	}

	slices.SortStableFunc(items, func(a, b lineItem) int {
		ksList := keySpecs
		if len(ksList) == 0 {
			ksList = []keySpec{{numeric: numeric, reverse: reverse, month: month, human: human}}
		}
		for ki := range ksList {
			if ki >= len(a.keys) || ki >= len(b.keys) {
				break
			}
			ks := ksList[ki]

			// Human sort: special comparison.
			if ks.human && ki < len(a.humanVals) && ki < len(b.humanVals) {
				ha := a.humanVals[ki]
				hb := b.humanVals[ki]
				cmp := compareHuman(ha, hb)
				if cmp != 0 {
					if ks.reverse {
						return -cmp
					}
					return cmp
				}
				continue
			}

			ak := a.keys[ki]
			bk := b.keys[ki]
			var less bool
			tie := false
			if (ks.month || ks.numeric) && ki < len(a.validNum) && ki < len(b.validNum) && a.validNum[ki] && b.validNum[ki] {
				less = a.numVals[ki] < b.numVals[ki]
				tie = a.numVals[ki] == b.numVals[ki]
			} else if ks.numeric {
				// Non-numeric values sort before numeric values.
				if a.validNum[ki] && !b.validNum[ki] {
					less = false
				} else if !a.validNum[ki] && b.validNum[ki] {
					less = true
				} else {
					less = ak < bk
					tie = ak == bk
				}
			} else {
				less = ak < bk
				tie = ak == bk
			}
			if tie {
				continue
			}
			if ks.reverse {
				if less {
					return 1
				}
				return -1
			}
			if less {
				return -1
			}
			return 1
		}
		// Last-resort comparison: full line (disabled by -s stable).
		// tieReverse (from global -r) only affects this fallback, not key comparisons.
		if !stable {
			cmp := 0
			if a.original < b.original {
				cmp = -1
			} else if a.original > b.original {
				cmp = 1
			}
			if tieReverse && cmp != 0 {
				return -cmp
			}
			return cmp
		}
		return 0
	})

	var result []string
	if unique {
		var lastKeys []string
		for i, item := range items {
			keys := item.keys
			if len(keys) == 0 {
				keys = []string{item.original}
			}
			if i == 0 || !slices.Equal(keys, lastKeys) {
				result = append(result, item.original)
				lastKeys = keys
			}
		}
	} else {
		for _, item := range items {
			result = append(result, item.original)
		}
	}
	return result
}

func run(args []string, out io.Writer) int {
	return sortRun(args, out, os.Stderr, os.Stdin)
}

func sortRun(args []string, out io.Writer, errOut io.Writer, stdin io.Reader) int {
	flags, err := common.ParseFlags(args, spec)
	if err != nil {
		fmt.Fprintf(errOut, "sort: %v\n", err)
		return 2
	}
	jsonMode := flags.Has("json")
	reverse := flags.Has("r")
	numeric := flags.Has("n")
	unique := flags.Has("u")
	delimiter := flags.Get("t")
	outputFile := flags.Get("o")
	zeroTerm := flags.Has("z")
	human := flags.Has("h")
	stable := flags.Has("s")
	month := flags.Has("M")

	var keySpecs []keySpec
	for _, kStr := range flags.GetAll("k") {
		specs := parseKeySpec(kStr, numeric, reverse, month, human)
		keySpecs = append(keySpecs, specs...)
	}
	// tieReverse: global -r only reverses the last-resort fallback when -k is used
	// AND the key spec uses numeric/month/human comparison.
	// For string-based -k specs, global -r reverses the key comparison normally.
	hasNumericKey := false
	for _, ks := range keySpecs {
		if ks.numeric || ks.month || ks.human {
			hasNumericKey = true
			break
		}
	}
	if reverse && len(keySpecs) > 0 && !hasNumericKey {
		// Apply global -r to all key specs (string comparison).
		for i := range keySpecs {
			keySpecs[i].reverse = !keySpecs[i].reverse
		}
	}
	tieReverse := reverse && len(keySpecs) > 0 && hasNumericKey

	if len(keySpecs) == 0 {
		if reverse || numeric || month || human {
			keySpecs = []keySpec{{reverse: reverse, numeric: numeric, month: month, human: human}}
			tieReverse = false
		}
	}

	var readers []io.Reader
	if len(flags.Positional) == 0 {
		readers = append(readers, stdin)
	} else {
		for _, path := range flags.Positional {
			if path == "-" {
				readers = append(readers, stdin)
			} else {
				f, err := os.Open(path)
				if err != nil {
					fmt.Fprintf(errOut, "sort: %s: %v\n", path, err)
					return 1
				}
				defer f.Close()
				readers = append(readers, f)
			}
		}
	}

	var allItems []lineItem
	for _, r := range readers {
		items, err := parseLines(r, keySpecs, delimiter, zeroTerm)
		if err != nil {
			fmt.Fprintf(errOut, "sort: %v\n", err)
			return 1
		}
		allItems = append(allItems, items...)
	}

	sortedLines := Run(allItems, keySpecs, reverse, numeric, unique, month, human, stable, tieReverse)

	var w io.Writer = out
	if outputFile != "" {
		f, err := os.Create(outputFile)
		if err != nil {
			fmt.Fprintf(errOut, "sort: %s: %v\n", outputFile, err)
			return 1
		}
		defer f.Close()
		w = f
	}

	if jsonMode {
		common.Render("sort", SortResult{Lines: sortedLines, Count: len(sortedLines)}, true, out, func() {})
	} else {
		for i, line := range sortedLines {
			if i > 0 {
				if zeroTerm {
					w.Write([]byte{0})
				} else {
					fmt.Fprintln(w)
				}
			}
			fmt.Fprint(w, line)
		}
		if zeroTerm && len(sortedLines) > 0 {
			w.Write([]byte{0})
		} else if !zeroTerm && len(sortedLines) > 0 {
			fmt.Fprintln(w)
		}
	}
	return 0
}

func compareHuman(a, b humanVal) int {
	// No suffix always comes BEFORE suffix.
	if !a.hasSuffix && b.hasSuffix {
		return -1
	}
	if a.hasSuffix && !b.hasSuffix {
		return 1
	}
	// Both no suffix: compare numerically, tie by original string.
	if !a.hasSuffix && !b.hasSuffix {
		if a.num != b.num {
			return cmpFloat(a.num, b.num)
		}
		if a.orig < b.orig {
			return -1
		}
		if a.orig > b.orig {
			return 1
		}
		return 0
	}
	// Both have suffix: compare suffix rank first, then numeric prefix, then original.
	if a.suffix != b.suffix {
		if a.suffix < b.suffix {
			return -1
		}
		return 1
	}
	if a.num != b.num {
		return cmpFloat(a.num, b.num)
	}
	if a.orig < b.orig {
		return -1
	}
	if a.orig > b.orig {
		return 1
	}
	return 0
}

func cmpFloat(a, b float64) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

func init() {
	dispatch.Register(dispatch.Command{Name: "sort", Usage: "Sort lines of text files", Run: run})
}
