// Package seq implements the POSIX/GNU-compliant seq utility.
package seq

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
)

// SeqResult contains the printed sequence.
type SeqResult struct {
	Sequence []string `json:"sequence"`
}

// preProcessSeqArgs pre-processes arguments to prevent negative numbers
// from being interpreted as short/long flags.
func preProcessSeqArgs(args []string) ([]string, error) {
	var result []string
	inserted := false

	i := 0
	for i < len(args) {
		arg := args[i]

		if inserted {
			result = append(result, arg)
			i++
			continue
		}

		if arg == "--" {
			inserted = true
			result = append(result, arg)
			i++
			continue
		}

		// -s or --separator expects a value
		if arg == "-s" || arg == "--separator" {
			result = append(result, arg)
			i++
			if i < len(args) {
				result = append(result, args[i])
				i++
			}
			continue
		}
		if strings.HasPrefix(arg, "-s=") || strings.HasPrefix(arg, "--separator=") {
			result = append(result, arg)
			i++
			continue
		}

		if strings.HasPrefix(arg, "-") {
			// Check if it's a negative number
			isNegNum := false
			if len(arg) > 1 {
				firstChar := arg[1]
				if (firstChar >= '0' && firstChar <= '9') || firstChar == '.' {
					isNegNum = true
				}
			}
			if isNegNum {
				result = append(result, "--")
				inserted = true
				result = append(result, arg)
				i++
				continue
			}

			result = append(result, arg)
			i++
			continue
		}

		// First non-flag positional argument
		result = append(result, "--")
		inserted = true
		result = append(result, arg)
		i++
	}
	return result, nil
}

var spec = common.FlagSpec{
	Defs: []common.FlagDef{
		{Short: "s", Long: "separator", Type: common.FlagValue},
		{Short: "w", Long: "equal-width", Type: common.FlagBool},
		{Long: "json", Type: common.FlagBool},
	},
	PreProcess: preProcessSeqArgs,
}

// parseNum parses a string float and determines its decimal precision and integer width.
func parseNum(s string) (float64, int, int, error) {
	val, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, 0, 0, err
	}

	clean := s
	if len(clean) > 0 && (clean[0] == '+' || clean[0] == '-') {
		clean = clean[1:]
	}

	dotIdx := strings.Index(clean, ".")
	if dotIdx == -1 {
		return val, 0, len(clean), nil
	}
	return val, len(clean) - dotIdx - 1, dotIdx, nil
}

// formatNum formats a float to target precision and equal width padding.
func formatNum(val float64, precision int, intWidth int, equalWidth bool) string {
	fs := fmt.Sprintf("%.*f", precision, val)
	if !equalWidth {
		return fs
	}
	sign := ""
	if strings.HasPrefix(fs, "-") {
		sign = "-"
		fs = fs[1:]
	}

	var intPart, decPart string
	dotIdx := strings.Index(fs, ".")
	if dotIdx == -1 {
		intPart = fs
		decPart = ""
	} else {
		intPart = fs[:dotIdx]
		decPart = fs[dotIdx:]
	}

	if len(intPart) < intWidth {
		intPart = strings.Repeat("0", intWidth-len(intPart)) + intPart
	}
	return sign + intPart + decPart
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer, cwd string) int {
	flags, err := common.ParseFlags(args, spec)
	if err != nil {
		fmt.Fprintf(stderr, "seq: %v\n", err)
		return 2
	}

	separator := "\n"
	if flags.Has("separator") {
		separator = flags.Get("separator")
	}
	equalWidth := flags.Has("equal-width")
	jsonMode := flags.Has("json")

	posArgs := flags.Positional
	if len(posArgs) < 1 || len(posArgs) > 3 {
		fmt.Fprintf(stderr, "seq: invalid number of arguments\n")
		return 1
	}

	var firstStr, stepStr, lastStr string
	if len(posArgs) == 1 {
		firstStr = "1"
		stepStr = "1"
		lastStr = posArgs[0]
	} else if len(posArgs) == 2 {
		firstStr = posArgs[0]
		stepStr = "1"
		lastStr = posArgs[1]
	} else {
		firstStr = posArgs[0]
		stepStr = posArgs[1]
		lastStr = posArgs[2]
	}

	first, firstPrec, firstWidth, err := parseNum(firstStr)
	if err != nil {
		fmt.Fprintf(stderr, "seq: invalid argument %s\n", firstStr)
		return 1
	}
	step, stepPrec, _, err := parseNum(stepStr)
	if err != nil {
		fmt.Fprintf(stderr, "seq: invalid argument %s\n", stepStr)
		return 1
	}
	last, _, lastWidth, err := parseNum(lastStr)
	if err != nil {
		fmt.Fprintf(stderr, "seq: invalid argument %s\n", lastStr)
		return 1
	}

	// Precision rule
	targetPrecision := stepPrec
	if len(posArgs) == 1 {
		_, lastPrec, _, _ := parseNum(lastStr)
		targetPrecision = lastPrec
	} else if len(posArgs) == 2 {
		if firstPrec > targetPrecision {
			targetPrecision = firstPrec
		}
		_, lastPrec, _, _ := parseNum(lastStr)
		if lastPrec > targetPrecision {
			targetPrecision = lastPrec
		}
	} else {
		if firstPrec > targetPrecision {
			targetPrecision = firstPrec
		}
	}

	// Width rule for equal width
	targetIntWidth := firstWidth
	if lastWidth > targetIntWidth {
		targetIntWidth = lastWidth
	}

	if step == 0 {
		if jsonMode {
			fmt.Fprintf(stderr, "seq: infinite sequence is not supported in JSON mode\n")
			return 1
		}
		// Infinite print for step=0
		for {
			fs := formatNum(first, targetPrecision, targetIntWidth, equalWidth)
			if _, err := fmt.Fprint(stdout, fs); err != nil {
				break
			}
			if _, err := fmt.Fprint(stdout, separator); err != nil {
				break
			}
		}
		return 0
	}

	var sequence []string
	const eps = 1e-9

	if step > 0 {
		for curr := first; curr <= last+eps; curr += step {
			// Precision boundary safety
			if curr > last && (curr-last) > eps {
				break
			}
			sequence = append(sequence, formatNum(curr, targetPrecision, targetIntWidth, equalWidth))
		}
	} else {
		for curr := first; curr >= last-eps; curr += step {
			// Precision boundary safety
			if curr < last && (last-curr) > eps {
				break
			}
			sequence = append(sequence, formatNum(curr, targetPrecision, targetIntWidth, equalWidth))
		}
	}

	result := SeqResult{
		Sequence: sequence,
	}

	common.Render("seq", result, jsonMode, stdout, func() {
		if len(sequence) > 0 {
			outStr := strings.Join(sequence, separator)
			fmt.Fprint(stdout, outStr)
			// seq always outputs a trailing newline after the last element or separator
			if !strings.HasSuffix(outStr, "\n") {
				fmt.Fprint(stdout, "\n")
			}
		}
	})

	return 0
}



func init() {
	dispatch.Register(dispatch.Command{
		Name:  "seq",
		Usage: "Print sequences of numbers",
		Run:   run,
	})
}
