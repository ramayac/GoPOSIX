// Package goposix — daemon forwarding (M5: CLI smart routing).
//
// When a GoPOSIX binary is invoked via symlink (e.g., /bin/ls -> /bin/goposix)
// and a daemon socket is available, commands are forwarded to the persistent
// daemon instead of cold-starting a new Go process. This reduces per-call
// latency from ~7ms (cold Go runtime init) to ~60µs (persistent connection).
package goposix

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// socketExists reports whether the daemon socket path exists.
func socketExists(path string) bool {
	fi, err := os.Stat(path)
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeSocket != 0
}

// isStdinPiped reports whether stdin is a pipe or redirect (not a terminal).
// When stdin is piped, we fall back to cold start because the daemon doesn't
// support streaming stdin through JSON-RPC.
func isStdinPiped() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return true // can't check — assume cold start is safer
	}
	return fi.Mode()&os.ModeCharDevice == 0
}

// forwardToDaemon sends a command to the persistent daemon via JSON-RPC and
// returns its exit code. The daemon runs the command with rawOutput=true so
// the response contains human-readable stdout text.
//
// This is the core of M5 smart routing: symlink invocations skip the cold
// Go runtime start and reuse the already-running daemon process.
func forwardToDaemon(socketPath string, argv []string) int {
	// Parse the command name and args — same logic as RunWithWriter.
	binName := filepath.Base(argv[0])
	cmdName := binName

	if isWellKnown(cmdName) {
		if len(argv) < 2 {
			// No subcommand — can't forward. Fall back to cold start.
			return -1
		}
		// Skip meta-flags like --help, --version, --list-commands.
		switch argv[1] {
		case "--help", "-h", "--version", "--list-commands":
			return -1
		}
		cmdName = strings.TrimSpace(argv[1])
		argv = argv[1:] // shift so args start at the command
	}

	// Build the JSON-RPC request.
	id := 1
	method := "goposix." + cmdName
	args := argv[1:] // skip command name

	params := map[string]interface{}{
		"rawOutput": true,
		"flags":     args,
	}

	request := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  method,
		"params":  params,
		"id":      id,
	}

	body, err := json.Marshal(request)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: forward marshal error: %v\n", binName, err)
		return 126
	}

	// Connect to the daemon with a short timeout.
	conn, err := net.DialTimeout("unix", socketPath, 2*time.Second)
	if err != nil {
		// Daemon not reachable — fall back to cold start.
		return -1
	}
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(5 * time.Second))

	// Send request.
	if _, err := conn.Write(append(body, '\n')); err != nil {
		return -1
	}

	// Read response (newline-delimited JSON — daemon keeps the connection open).
	var resp struct {
		JSONRPC string `json:"jsonrpc"`
		ID      int    `json:"id"`
		Result  struct {
			ExitCode int    `json:"exitCode"`
			Stdout   string `json:"stdout"`
		} `json:"result"`
		Error *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}

	dec := json.NewDecoder(conn)
	if err := dec.Decode(&resp); err != nil {
		fmt.Fprintf(os.Stderr, "%s: forward parse error: %v\n", binName, err)
		return 126
	}

	if resp.Error != nil {
		fmt.Fprintf(os.Stderr, "%s: %s\n", binName, resp.Error.Message)
		return 126
	}

	// Print the command's stdout.
	os.Stdout.WriteString(resp.Result.Stdout)

	return resp.Result.ExitCode
}

// TryForward checks for a daemon socket and forwards the command if possible.
// Returns the exit code from the forwarded command, or -1 if forwarding was
// not attempted (caller should use normal cold-start dispatch).
func TryForward() int {
	socketPath := os.Getenv("GOPOSIX_SOCKET")
	if socketPath == "" {
		socketPath = "/var/run/goposix.sock"
	}

	if !socketExists(socketPath) {
		return -1
	}

	// Don't forward when stdin is piped — the daemon can't handle stdin
	// streaming through JSON-RPC. Cold start handles this correctly.
	if isStdinPiped() {
		return -1
	}

	return forwardToDaemon(socketPath, os.Args)
}
