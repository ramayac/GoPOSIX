// Package factor implements the POSIX-compliant factor utility.
package factor

import (
	"bufio"
	"fmt"
	"io"
	"math/bits"
	"sort"
	"strconv"
	"strings"

	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
)

// FactorEntry is the JSON structure for a single number's factorization.
type FactorEntry struct {
	Input   string   `json:"input"`
	Factors []uint64 `json:"factors,omitempty"`
	Error   string   `json:"error,omitempty"`
}

// FactorResult is the overall JSON result structure for the factor utility.
type FactorResult struct {
	Results []FactorEntry `json:"results"`
}

// preProcessFactorArgs pre-processes command-line arguments to prevent
// negative numbers or other numeric strings starting with '-' from being
// interpreted as short or long flags.
func preProcessFactorArgs(args []string) ([]string, error) {
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

		if strings.HasPrefix(arg, "-") {
			// If it matches exactly one of our supported flags, keep it
			if arg == "-h" || arg == "--help" || arg == "-V" || arg == "--version" || arg == "--json" {
				result = append(result, arg)
				i++
				continue
			}

			// Otherwise, it is a positional argument (like -10, or an invalid input)
			// Insert "--" before it to prevent flag parser error
			result = append(result, "--")
			inserted = true
			result = append(result, arg)
			i++
			continue
		}

		// First non-flag argument: terminate flag parsing
		result = append(result, "--")
		inserted = true
		result = append(result, arg)
		i++
	}
	return result, nil
}

var spec = common.FlagSpec{
	Defs: []common.FlagDef{
		{Short: "h", Long: "help", Type: common.FlagBool},
		{Short: "V", Long: "version", Type: common.FlagBool},
		{Long: "json", Type: common.FlagBool},
	},
	PreProcess: preProcessFactorArgs,
}

func init() {
	dispatch.Register(dispatch.Command{
		Name:  "factor",
		Usage: "Factorize numbers into prime components",
		Run:   run,
	})
}

// mulMod calculates (a * b) % m using 128-bit multiplication and division.
// It will not overflow as long as a, b < m.
func mulMod(a, b, m uint64) uint64 {
	hi, lo := bits.Mul64(a, b)
	_, rem := bits.Div64(hi, lo, m)
	return rem
}

// powMod calculates (base ^ exp) % mod.
func powMod(base, exp, mod uint64) uint64 {
	var result uint64 = 1
	base = base % mod
	for exp > 0 {
		if exp&1 == 1 {
			result = mulMod(result, base, mod)
		}
		base = mulMod(base, base, mod)
		exp >>= 1
	}
	return result
}

// millerRabin performs a deterministic primality test for uint64.
func millerRabin(n uint64) bool {
	if n < 2 {
		return false
	}
	if n == 2 || n == 3 {
		return true
	}
	if n%2 == 0 {
		return false
	}

	// Write n-1 as d * 2^s
	d := n - 1
	s := 0
	for d%2 == 0 {
		d /= 2
		s++
	}

	// Deterministic bases for 64-bit integers
	bases := []uint64{2, 3, 5, 7, 11, 13, 17, 19, 23, 29, 31, 37}

	for _, a := range bases {
		if n <= a {
			break
		}
		x := powMod(a, d, n)
		if x == 1 || x == n-1 {
			continue
		}
		composite := true
		for r := 0; r < s-1; r++ {
			x = mulMod(x, x, n)
			if x == n-1 {
				composite = false
				break
			}
		}
		if composite {
			return false
		}
	}
	return true
}

func gcd(a, b uint64) uint64 {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}

// pollardRho finds a non-trivial factor of composite n.
func pollardRho(n uint64) uint64 {
	if n%2 == 0 {
		return 2
	}
	if n%3 == 0 {
		return 3
	}

	for c := uint64(1); ; c++ {
		var x uint64 = 2
		var y uint64 = 2
		var d uint64 = 1

		f := func(x uint64) uint64 {
			return (mulMod(x, x, n) + c) % n
		}

		for d == 1 {
			x = f(x)
			y = f(f(y))
			var diff uint64
			if x > y {
				diff = x - y
			} else {
				diff = y - x
			}
			d = gcd(diff, n)
		}

		if d != n {
			return d
		}
	}
}

// factorize recursively finds all prime factors of n.
func factorize(n uint64, factors *[]uint64) {
	if n < 2 {
		return
	}
	if millerRabin(n) {
		*factors = append(*factors, n)
		return
	}
	d := pollardRho(n)
	factorize(d, factors)
	factorize(n/d, factors)
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer, cwd string) int {
	// Pre-detect jsonMode in case ParseFlags fails
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
			common.RenderError("factor", 1, "FLAG_ERROR", err.Error(), true, stderr)
		} else {
			fmt.Fprintf(stderr, "factor: %v\n", err)
		}
		return 1
	}

	if flags.Has("h") || flags.Has("help") {
		helpText := "Usage: factor [NUMBER]...\n\n" +
			"Print the prime factors of each NUMBER.\n\n" +
			"Options:\n" +
			"  -h, --help     Print help\n" +
			"  -V, --version  Print version"
		common.Render("factor", struct {
			Help string `json:"help"`
		}{Help: helpText}, jsonMode, stdout, func() {
			fmt.Fprintln(stdout, helpText)
		})
		return 0
	}

	if flags.Has("V") || flags.Has("version") {
		versionText := "factor version v1.0.0-goposix"
		common.Render("factor", struct {
			Version string `json:"version"`
		}{Version: versionText}, jsonMode, stdout, func() {
			fmt.Fprintln(stdout, versionText)
		})
		return 0
	}

	var inputs []string
	if len(flags.Positional) > 0 {
		inputs = flags.Positional
	} else {
		// Read whitespace-separated inputs from stdin
		scanner := bufio.NewScanner(stdin)
		scanner.Split(bufio.ScanWords)
		for scanner.Scan() {
			inputs = append(inputs, scanner.Text())
		}
	}

	var results []FactorEntry
	hasError := false

	for _, inputStr := range inputs {
		clean := strings.TrimSpace(inputStr)
		origClean := clean
		if len(clean) > 0 && clean[0] == '+' {
			clean = clean[1:]
		}

		// Ensure we don't crash or parse negative values as uint
		if strings.HasPrefix(clean, "-") || clean == "" {
			hasError = true
			if jsonMode {
				results = append(results, FactorEntry{
					Input: origClean,
					Error: "invalid number",
				})
			} else {
				fmt.Fprintf(stderr, "factor: %s: invalid number\n", origClean)
			}
			continue
		}

		val, err := strconv.ParseUint(clean, 10, 64)
		if err != nil {
			hasError = true
			if jsonMode {
				results = append(results, FactorEntry{
					Input: origClean,
					Error: "invalid number",
				})
			} else {
				fmt.Fprintf(stderr, "factor: %s: invalid number\n", origClean)
			}
			continue
		}

		var factors []uint64
		if val > 1 {
			factorize(val, &factors)
			sort.Slice(factors, func(i, j int) bool {
				return factors[i] < factors[j]
			})
		}

		results = append(results, FactorEntry{
			Input:   origClean,
			Factors: factors,
		})

		if !jsonMode {
			// Print standard factor output format: "num: f1 f2 f3"
			fmt.Fprintf(stdout, "%d:", val)
			for _, f := range factors {
				fmt.Fprintf(stdout, " %d", f)
			}
			fmt.Fprintln(stdout)
		}
	}

	if jsonMode {
		common.Render("factor", FactorResult{Results: results}, true, stdout, nil)
	}

	if hasError {
		return 1
	}
	return 0
}
