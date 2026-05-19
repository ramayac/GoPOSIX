// Package common provides shared utilities for goposix utilities.
//
// Flag parsing types and functions are re-exported from internal/getopt.
package common

import "github.com/ramayac/goposix/internal/getopt"

// Types re-exported for backward compatibility.
type (
	FlagDef     = getopt.FlagDef
	FlagSpec    = getopt.FlagSpec
	FlagType    = getopt.FlagType
	ParseResult = getopt.ParseResult
	FlagError   = getopt.FlagError
)

// Constants re-exported.
const (
	FlagBool          = getopt.FlagBool
	FlagValue         = getopt.FlagValue
	FlagOptionalValue = getopt.FlagOptionalValue
)

// ParseFlags is re-exported from internal/getopt.
var ParseFlags = getopt.ParseFlags
