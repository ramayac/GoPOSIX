// Package common provides shared helpers for goposix utilities.
package common

import (
	"fmt"
	"io"
)

// HasJSONFlag reports whether args contains --json.
func HasJSONFlag(args []string) bool {
	for _, a := range args {
		if a == "--json" {
			return true
		}
	}
	return false
}

// RenderFlagError renders a flag-parse error and returns the given POSIX exit
// code.
//
// When args contains --json it writes a JSON error envelope to errOut;
// otherwise it writes a plain "<name>: <message>" line to errOut. The caller
// supplies the exit code so each command keeps its historical flag-error
// convention (most use 2, some use 1).
func RenderFlagError(name string, args []string, err error, errOut io.Writer, exitCode int) int {
	if HasJSONFlag(args) {
		RenderError(name, exitCode, "FLAG_ERROR", err.Error(), true, errOut)
		return exitCode
	}
	fmt.Fprintf(errOut, "%s: %v\n", name, err)
	return exitCode
}
