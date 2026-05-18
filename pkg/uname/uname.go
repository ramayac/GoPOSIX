// Package uname implements the POSIX uname utility.
package uname

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
)

// UnameResult is the structured result for --json mode.
type UnameResult struct {
	Sysname  string `json:"sysname"`
	Nodename string `json:"nodename"`
	Release  string `json:"release"`
	Version  string `json:"version"`
	Machine  string `json:"machine"`
}

var spec = common.FlagSpec{
	Defs: []common.FlagDef{
		{Short: "a", Long: "all", Type: common.FlagBool},
		{Short: "s", Long: "kernel-name", Type: common.FlagBool},
		{Short: "n", Long: "nodename", Type: common.FlagBool},
		{Short: "r", Long: "kernel-release", Type: common.FlagBool},
		{Short: "v", Long: "kernel-version", Type: common.FlagBool},
		{Short: "m", Long: "machine", Type: common.FlagBool},
		{Long: "json", Type: common.FlagBool},
	},
}

// charsToString converts a null-terminated byte array to a Go string.
func charsToString(chars [65]int8) string {
	b := make([]byte, 0, 65)
	for _, c := range chars {
		if c == 0 {
			break
		}
		b = append(b, byte(c))
	}
	return string(b)
}

func run(args []string, out io.Writer) int {
	flags, err := common.ParseFlags(args, spec)
	if err != nil {
		fmt.Fprintf(os.Stderr, "uname: %v\n", err)
		return 2
	}
	jsonMode := flags.Has("json")

	result, err := Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "uname: %v\n", err)
		common.RenderError("uname", 1, "EUNAME", err.Error(), jsonMode, out)
		return 1
	}

	all := flags.Has("a")
	s := flags.Has("s") || all
	n := flags.Has("n") || all
	r := flags.Has("r") || all
	v := flags.Has("v") || all
	m := flags.Has("m") || all

	// Default: print kernel name only (like standard uname with no flags).
	if !s && !n && !r && !v && !m {
		s = true
	}

	common.Render("uname", result, jsonMode, out, func() {
		var parts []string
		if s {
			parts = append(parts, result.Sysname)
		}
		if n {
			parts = append(parts, result.Nodename)
		}
		if r {
			parts = append(parts, result.Release)
		}
		if v {
			parts = append(parts, result.Version)
		}
		if m {
			parts = append(parts, result.Machine)
		}
		fmt.Println(strings.Join(parts, " "))
	})
	return 0
}

func init() {
	dispatch.Register(dispatch.Command{
		Name:  "uname",
		Usage: "Print system information",
		Run:   run,
	})
}
