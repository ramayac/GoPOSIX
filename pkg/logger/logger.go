// Package logger implements the POSIX logger utility — submit messages to syslog.
package logger

import (
	"fmt"
	"io"
	"net"
	"os"
	"strings"

	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
)

// LoggerResult is the --json output.
type LoggerResult struct {
	Priority string `json:"priority"`
	Tag      string `json:"tag"`
	Message  string `json:"message"`
}

var spec = common.FlagSpec{
	Defs: []common.FlagDef{
		{Short: "p", Long: "priority", Type: common.FlagValue},
		{Short: "t", Long: "tag", Type: common.FlagValue},
		{Short: "s", Long: "stderr", Type: common.FlagBool},
		{Long: "json", Type: common.FlagBool},
	},
}

// facilityMapping maps facility names to their numeric codes (RFC 5424).
var facilityMapping = map[string]int{
	"kern":     0,
	"user":     1,
	"mail":     2,
	"daemon":   3,
	"auth":     4,
	"syslog":   5,
	"lpr":      6,
	"news":     7,
	"uucp":     8,
	"cron":     9,
	"authpriv": 10,
	"ftp":      11,
	"local0":   16,
	"local1":   17,
	"local2":   18,
	"local3":   19,
	"local4":   20,
	"local5":   21,
	"local6":   22,
	"local7":   23,
}

// severityMapping maps severity names to their numeric codes (RFC 5424).
var severityMapping = map[string]int{
	"emerg":   0,
	"alert":   1,
	"crit":    2,
	"err":     3,
	"error":   3,
	"warning": 4,
	"warn":    4,
	"notice":  5,
	"info":    6,
	"debug":   7,
}

// parsePriority parses a priority string like "user.notice" or "local0.info".
func parsePriority(s string) (int, error) {
	if s == "" {
		return 1*8 + 5, nil // user.notice
	}

	parts := strings.SplitN(s, ".", 2)
	facility := 1 // user
	severity := 5 // notice

	if len(parts) >= 1 && parts[0] != "" {
		if f, ok := facilityMapping[parts[0]]; ok {
			facility = f
		} else {
			return 0, fmt.Errorf("unknown facility: %s", parts[0])
		}
	}
	if len(parts) >= 2 {
		if sev, ok := severityMapping[parts[1]]; ok {
			severity = sev
		} else {
			return 0, fmt.Errorf("unknown severity: %s", parts[1])
		}
	}

	return facility*8 + severity, nil
}

// formatSyslogMessage formats a message according to RFC 3164 style.
func formatSyslogMessage(pri int, tag, msg string) string {
	// <PRI>TIMESTAMP HOSTNAME TAG: MESSAGE
	// POSIX logger uses a simpler format
	return fmt.Sprintf("<%d>%s: %s", pri, tag, msg)
}

// Run submits a message to syslog.
var (
	dialSyslogFn = func(network, address string) (net.Conn, error) {
		return net.Dial(network, address)
	}
	stderrWriter io.Writer = os.Stderr
)

// Run submits a message to syslog.
func Run(message, tag, priorityStr string, alsoStderr bool) (LoggerResult, error) {
	pri, err := parsePriority(priorityStr)
	if err != nil {
		return LoggerResult{}, err
	}

	if tag == "" {
		tag = "logger"
	}

	formatted := formatSyslogMessage(pri, tag, message)

	// Try to connect to /dev/log first
	conn, err := dialSyslogFn("unixgram", "/dev/log")
	if err != nil {
		// Fallback: try /var/run/syslog or UDP localhost
		conn, err = dialSyslogFn("unixgram", "/var/run/syslog")
		if err != nil {
			// Last resort: UDP to localhost:514
			conn, err = dialSyslogFn("udp", "127.0.0.1:514")
			if err != nil {
				// Silently ignore if no syslog available (matches GNU logger behavior)
				result := LoggerResult{
					Priority: priorityStr,
					Tag:      tag,
					Message:  message,
				}
				if alsoStderr {
					fmt.Fprintln(stderrWriter, message)
				}
				return result, nil
			}
		}
	}
	defer conn.Close()

	if _, err := conn.Write([]byte(formatted)); err != nil {
		return LoggerResult{}, err
	}

	if alsoStderr {
		fmt.Fprintln(stderrWriter, message)
	}

	return LoggerResult{
		Priority: priorityStr,
		Tag:      tag,
		Message:  message,
	}, nil
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer, cwd string) int {
	flags, err := common.ParseFlags(args, spec)
	if err != nil {
		fmt.Fprintf(stderr, "logger: %v\n", err)
		return 2
	}
	jsonMode := flags.Has("json")

	tag := "logger"
	if flags.Has("t") {
		tag = flags.Get("t")
	}

	priorityStr := "user.notice"
	if flags.Has("p") {
		priorityStr = flags.Get("p")
	}

	alsoStderr := flags.Has("s")

	message := strings.Join(flags.Positional, " ")

	// If no positional args, read from stdin
	if message == "" {
		data, err := io.ReadAll(stdin)
		if err != nil {
			fmt.Fprintf(stderr, "logger: %v\n", err)
			common.RenderError("logger", 1, "EREAD", err.Error(), jsonMode, stdout)
			return 1
		}
		message = strings.TrimSpace(string(data))
	}

	// Capture custom stderr mapping during run execution
	oldStderrWriter := stderrWriter
	stderrWriter = stderr
	defer func() { stderrWriter = oldStderrWriter }()

	result, err := Run(message, tag, priorityStr, alsoStderr)
	if err != nil {
		fmt.Fprintf(stderr, "logger: %v\n", err)
		common.RenderError("logger", 1, "ELOGGER", err.Error(), jsonMode, stdout)
		return 1
	}

	common.Render("logger", result, jsonMode, stdout, func() {})
	return 0
}

func init() {
	dispatch.Register(dispatch.Command{
		Name:  "logger",
		Usage: "Submit messages to the system logger",
		Run:   run,
	})
}
