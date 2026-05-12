// Package common provides the standard JSON output envelope for korego utilities.
package common

import (
	"encoding/json"
	"io"
)

// Version is the korego release version, injected at build time with -ldflags.
var Version = "0.1.0"

// JSONEnvelope is the standard top-level response format for every utility when
// run with --json.  All six keys are always present.
type JSONEnvelope struct {
	Command       string      `json:"command"`
	Version       string      `json:"version"`
	SchemaVersion string      `json:"schemaVersion"`
	ExitCode      int         `json:"exitCode"`
	Data          interface{} `json:"data"`
	Error         *ErrorInfo  `json:"error"`
}

// ErrorInfo carries a POSIX-style error code and a human-readable message.
type ErrorInfo struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Render writes output to out.  If jsonMode is true it marshals a
// JSONEnvelope; otherwise it calls textFn for the traditional text output.
func Render(cmdName string, data interface{}, jsonMode bool, out io.Writer, textFn func()) {
	if jsonMode {
		env := JSONEnvelope{
			Command:       cmdName,
			Version:       Version,
			SchemaVersion: "1.0",
			ExitCode:      0,
			Data:          data,
			Error:         nil,
		}
		enc := json.NewEncoder(out)
		enc.SetEscapeHTML(false)
		_ = enc.Encode(env)
	} else {
		textFn()
	}
}

// RenderError writes a JSON error envelope to out and returns the exit code.
// If jsonMode is false it is a no-op (caller should print its own error to stderr).
func RenderError(cmdName string, exitCode int, errCode, message string, jsonMode bool, out io.Writer) {
	if !jsonMode {
		return
	}
	env := JSONEnvelope{
		Command:       cmdName,
		Version:       Version,
		SchemaVersion: "1.0",
		ExitCode:      exitCode,
		Data:          nil,
		Error: &ErrorInfo{
			Code:    errCode,
			Message: message,
		},
	}
	enc := json.NewEncoder(out)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(env)
}
