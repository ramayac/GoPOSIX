// Package common provides shared utilities for goposix utilities.
package common

import "fmt"

// FlagType enumerates the kinds of values a flag can hold.
type FlagType int

const (
	FlagBool          FlagType = iota // -l, --all
	FlagValue                         // --key=value
	FlagOptionalValue                 // --key[=value], -e[eof-str]
)

// FlagDef describes a single accepted flag.
type FlagDef struct {
	Short string   // single character, e.g. "l"
	Long  string   // long name without --, e.g. "all"
	Type  FlagType // Bool, Value, or OptionalValue
}

// FlagSpec is the set of accepted flags for a command.
type FlagSpec struct {
	Defs               []FlagDef
	StopAtFirstNonFlag bool // if true, stop parsing at first non-flag argument (echo, printf)

	compiled *compiledSpec // lazily built on first call
}

// FlagError is returned when flag parsing fails.
type FlagError struct {
	ExitCode int
	Msg      string
}

func (e *FlagError) Error() string { return e.Msg }

// ParseResult holds the parsed flags and positional arguments.
type ParseResult struct {
	Bools      map[string]bool
	Values     map[string]string
	ValuesList map[string][]string
	Count      map[string]int
	Positional []string
	Stdin      bool
}

func newParseResult() *ParseResult {
	return &ParseResult{
		Bools:      make(map[string]bool),
		Values:     make(map[string]string),
		ValuesList: make(map[string][]string),
		Count:      make(map[string]int),
	}
}

// Has returns true if the short or long flag name was set.
func (r *ParseResult) Has(name string) bool {
	return r.Bools[name]
}

// Get returns the value for a value-type flag (last value seen).
func (r *ParseResult) Get(name string) string {
	return r.Values[name]
}

// GetAll returns all values for a value-type flag.
func (r *ParseResult) GetAll(name string) []string {
	return r.ValuesList[name]
}

func errf(code int, format string, args ...interface{}) *FlagError {
	return &FlagError{ExitCode: code, Msg: fmt.Sprintf(format, args...)}
}

// ParseFlags parses args according to spec.
// Unknown flags return *FlagError with ExitCode 2.
func ParseFlags(args []string, spec FlagSpec) (*ParseResult, error) {
	cs := spec.getOrCompile()
	res := newParseResult()
	argsLen := len(args)
	res.Positional = make([]string, 0, argsLen)

	i := 0
	for i < argsLen {
		arg := args[i]

		// End of flags marker.
		if arg == "--" {
			i++
			for i < argsLen {
				res.Positional = append(res.Positional, args[i])
				i++
			}
			break
		}

		// Stdin marker.
		if arg == "-" {
			res.Stdin = true
			res.Positional = append(res.Positional, "-")
			i++
			continue
		}

		// Not a flag → positional.
		if len(arg) < 2 || arg[0] != '-' {
			if cs.stopAtFirst {
				for i < argsLen {
					res.Positional = append(res.Positional, args[i])
					i++
				}
				break
			}
			res.Positional = append(res.Positional, arg)
			i++
			continue
		}

		// Long flag: --name or --name=value.
		if arg[1] == '-' {
			ni, err := cs.parseLong(args, i, res)
			if err != nil {
				return nil, err
			}
			i = ni
			continue
		}

		// Short flag(s): -laR or -ofile.
		ni, err := cs.parseShort(args, i, res)
		if err != nil {
			return nil, err
		}
		i = ni
	}

	return res, nil
}

// parseLong handles --name or --name=value at position i.
func (cs *compiledSpec) parseLong(args []string, i int, res *ParseResult) (int, error) {
	arg := args[i]
	name := arg[2:]
	var value string
	hasEq := false

	// Scan for '=' inline.
	for j := 0; j < len(name); j++ {
		if name[j] == '=' {
			value = name[j+1:]
			name = name[:j]
			hasEq = true
			break
		}
	}

	cf := cs.lookupLong(name)
	if cf == nil {
		return 0, errf(2, "unknown flag: --%s", name)
	}

	if cf.Type == FlagBool {
		res.Bools[cf.Short] = true
		res.Count[cf.Short]++
		if cf.Long != "" {
			res.Bools[cf.Long] = true
			res.Count[cf.Long]++
		}
		return i + 1, nil
	}

	// Value or OptionalValue flag.
	if !hasEq {
		if cf.Type == FlagOptionalValue {
			value = ""
		} else {
			if i+1 >= len(args) {
				return 0, errf(2, "flag --%s requires a value", name)
			}
			i++
			value = args[i]
		}
	}

	if cf.Short != "" {
		res.Values[cf.Short] = value
		res.ValuesList[cf.Short] = append(res.ValuesList[cf.Short], value)
		res.Bools[cf.Short] = true
	}
	res.Values[cf.Long] = value
	res.ValuesList[cf.Long] = append(res.ValuesList[cf.Long], value)
	res.Bools[cf.Long] = true

	return i + 1, nil
}

// parseShort handles -abc or -ovalue at position i.
func (cs *compiledSpec) parseShort(args []string, i int, res *ParseResult) (int, error) {
	arg := args[i]
	chars := arg[1:]

	for ci := 0; ci < len(chars); ci++ {
		b := chars[ci]
		cf := cs.lookupShort(b)
		if cf == nil {
			return 0, errf(2, "unknown flag: -%c", b)
		}

		if cf.Type == FlagBool {
			res.Bools[cf.Short] = true
			res.Count[cf.Short]++
			if cf.Long != "" {
				res.Bools[cf.Long] = true
				res.Count[cf.Long]++
			}
			continue
		}

		remainder := chars[ci+1:]
		if remainder != "" {
			if cf.Short != "" {
				res.Values[cf.Short] = remainder
				res.ValuesList[cf.Short] = append(res.ValuesList[cf.Short], remainder)
				res.Bools[cf.Short] = true
			}
			if cf.Long != "" {
				res.Values[cf.Long] = remainder
				res.ValuesList[cf.Long] = append(res.ValuesList[cf.Long], remainder)
				res.Bools[cf.Long] = true
			}
			return i + 1, nil
		}

		if cf.Type == FlagOptionalValue {
			if cf.Short != "" {
				res.Values[cf.Short] = ""
				res.ValuesList[cf.Short] = append(res.ValuesList[cf.Short], "")
				res.Bools[cf.Short] = true
			}
			if cf.Long != "" {
				res.Values[cf.Long] = ""
				res.ValuesList[cf.Long] = append(res.ValuesList[cf.Long], "")
				res.Bools[cf.Long] = true
			}
			return i + 1, nil
		}

		if i+1 >= len(args) {
			return 0, errf(2, "flag -%s requires a value", cf.Short)
		}
		i++
		value := args[i]
		if cf.Short != "" {
			res.Values[cf.Short] = value
			res.ValuesList[cf.Short] = append(res.ValuesList[cf.Short], value)
			res.Bools[cf.Short] = true
		}
		if cf.Long != "" {
			res.Values[cf.Long] = value
			res.ValuesList[cf.Long] = append(res.ValuesList[cf.Long], value)
			res.Bools[cf.Long] = true
		}
		return i + 1, nil
	}

	return i + 1, nil
}
