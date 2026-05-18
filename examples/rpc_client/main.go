// RPC client example: demonstrates using the GoPOSIX daemon via JSON-RPC.
//
// This program starts a goposix daemon, creates a session, executes a multi-step
// file-inspection task using the JSON-RPC API, then cleans up.
//
// Build: go build -o rpc_client ./examples/rpc_client
// Run:   ./rpc_client
package main

import (
	"encoding/json"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// rpcRequest is a JSON-RPC 2.0 request.
type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
	ID      int             `json:"id"`
}

// rpcResponse is a JSON-RPC 2.0 response.
type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
	ID      int             `json:"id"`
}

type rpcError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// daemonResult wraps the goposix daemon's embedded result envelope.
type daemonResult struct {
	ExitCode int             `json:"exitCode"`
	Data     json.RawMessage `json:"data"`
}

// sessionInfo is returned by goposix.session.create.
type sessionInfo struct {
	SessionID  string            `json:"sessionId"`
	CWD        string            `json:"cwd"`
	Env        map[string]string `json:"env"`
	LastActive string            `json:"lastActive"`
}

func main() {
	log.SetFlags(log.Ltime)
	log.Println("=== GoPOSIX RPC Client Example ===")
	log.Println()

	tmpDir, err := os.MkdirTemp("", "goposix-rpc-*")
	if err != nil {
		log.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	socketPath := filepath.Join(tmpDir, "goposix.sock")
	daemonPath := "./goposix"

	// Build daemon if we have the source available.
	if _, err := os.Stat(daemonPath); os.IsNotExist(err) {
		log.Println("Building goposix...")
		out, err := exec.Command("go", "build", "-o", daemonPath, "./cmd/goposix").CombinedOutput()
		if err != nil {
			log.Fatalf("build failed: %v\n%s", err, out)
		}
	}

	// Step 1 — start daemon.
	log.Printf("Starting daemon on %s...", socketPath)
	daemon := exec.Command(daemonPath, "daemon", "-s", socketPath, "-w", "4")
	daemon.Stdout = os.Stdout
	daemon.Stderr = os.Stderr
	if err := daemon.Start(); err != nil {
		log.Fatalf("failed to start daemon: %v", err)
	}
	defer func() {
		daemon.Process.Signal(os.Interrupt)
		daemon.Wait()
	}()

	// Wait for socket to appear.
	if !waitForSocket(socketPath, 5*time.Second) {
		log.Fatal("daemon socket did not appear in time")
	}
	log.Println("Daemon is ready.")

	// Step 2 — ping.
	conn, id := dial(socketPath)
	log.Println("Connected to daemon.")

	result := call(conn, id, "goposix.ping", nil)
	ping := struct {
		Pong    bool   `json:"pong"`
		Uptime  string `json:"uptime"`
		Version string `json:"version"`
	}{}
	mustUnmarshal(result, &ping)
	log.Printf("Ping: pong=%v version=%s uptime=%s", ping.Pong, ping.Version, ping.Uptime)

	// Step 3 — create a session.
	id++
	result = call(conn, id, "goposix.session.create", nil)
	var session sessionInfo
	mustUnmarshal(result, &session)
	log.Printf("Session created: %s (cwd=%s)", session.SessionID, session.CWD)

	// Step 4 — set session CWD for local-path commands.
	id++
	callVoid(conn, id, "goposix.session.setCwd", map[string]string{
		"sessionId": session.SessionID,
		"path":      "/etc",
	})

	// Step 5 — run a file-inspection task using relative paths within CWD.
	// 5a. List files in /etc (via session CWD).
	id++
	result = call(conn, id, "goposix.ls", map[string]interface{}{
		"sessionId": session.SessionID,
	})
	var lsResult struct {
		ExitCode int `json:"exitCode"`
		Data     struct {
			Files []struct {
				Name  string `json:"name"`
				Size  int64  `json:"size"`
				IsDir bool   `json:"isDir"`
			} `json:"files"`
			Total int `json:"total"`
		} `json:"data"`
	}
	mustUnmarshal(result, &lsResult)
	log.Printf("ls /etc: %d entries found", lsResult.Data.Total)

	// 5b. Count lines in hosts (relative to session CWD /etc).
	id++
	result = call(conn, id, "goposix.wc", map[string]interface{}{
		"sessionId": session.SessionID,
		"path":      "hosts",
	})
	var wcResult daemonResult
	mustUnmarshal(result, &wcResult)
	log.Printf("wc hosts: exitCode=%d", wcResult.ExitCode)
	if wcResult.Data != nil {
		var wcData struct {
			Lines int `json:"lines"`
			Words int `json:"words"`
		}
		json.Unmarshal(wcResult.Data, &wcData)
		log.Printf("  lines=%d words=%d", wcData.Lines, wcData.Words)
	}

	// 5c. Use shell.exec to run a shell command in the session.
	id++
	result = call(conn, id, "goposix.shell.exec", map[string]interface{}{
		"sessionId": session.SessionID,
		"script":    "echo hello from goposix",
	})
	var execResult struct {
		Stdout   string `json:"stdout"`
		Stderr   string `json:"stderr"`
		ExitCode uint8  `json:"exitCode"`
	}
	mustUnmarshal(result, &execResult)
	log.Printf("shell.exec: stdout=%q exitCode=%d", execResult.Stdout, execResult.ExitCode)

	// Step 6 — inspect host.conf (relative to session CWD /etc).
	log.Println()
	log.Println("--- Inspecting host.conf ---")
	id++
	result = call(conn, id, "goposix.cat", map[string]interface{}{
		"sessionId": session.SessionID,
		"path":      "host.conf",
	})
	var catResult daemonResult
	mustUnmarshal(result, &catResult)
	log.Printf("cat host.conf: exitCode=%d", catResult.ExitCode)
	if catResult.Data != nil {
		var catData struct {
			Lines     []string `json:"lines"`
			LineCount int      `json:"lineCount"`
		}
		json.Unmarshal(catResult.Data, &catData)
		log.Printf("  lineCount=%d first_line=%q", catData.LineCount, firstLine(catData.Lines))
	}

	// Step 7 — destroy the session.
	id++
	callVoid(conn, id, "goposix.session.destroy", map[string]string{
		"sessionId": session.SessionID,
	})
	log.Println("Session destroyed.")

	conn.Close()

	// Step 8 — stop daemon.
	daemon.Process.Signal(os.Interrupt)
	daemon.Wait()
	log.Println()
	log.Println("=== RPC client example complete ===")
}

func firstLine(lines []string) string {
	if len(lines) > 0 {
		return lines[0]
	}
	return ""
}

// --- helpers ---

func dial(socketPath string) (net.Conn, int) {
	conn, err := net.DialTimeout("unix", socketPath, 5*time.Second)
	if err != nil {
		log.Fatalf("dial failed: %v", err)
	}
	return conn, 1
}

func call(conn net.Conn, id int, method string, params interface{}) json.RawMessage {
	conn.SetDeadline(time.Now().Add(5 * time.Second))

	req := rpcRequest{JSONRPC: "2.0", Method: method, ID: id}
	if params != nil {
		b, _ := json.Marshal(params)
		req.Params = b
	}

	enc := json.NewEncoder(conn)
	if err := enc.Encode(req); err != nil {
		log.Fatalf("write %s failed: %v", method, err)
	}

	var resp rpcResponse
	dec := json.NewDecoder(conn)
	if err := dec.Decode(&resp); err != nil {
		if err == io.EOF {
			log.Fatalf("%s: connection closed (daemon may have crashed)", method)
		}
		log.Fatalf("decode %s response failed: %v", method, err)
	}

	if resp.Error != nil {
		log.Fatalf("%s RPC error: code=%d msg=%s", method, resp.Error.Code, resp.Error.Message)
	}

	return resp.Result
}

func callVoid(conn net.Conn, id int, method string, params interface{}) {
	_ = call(conn, id, method, params)
}

func waitForSocket(path string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); err == nil {
			return true
		}
		time.Sleep(50 * time.Millisecond)
	}
	return false
}

func mustUnmarshal(raw json.RawMessage, v interface{}) {
	if err := json.Unmarshal(raw, v); err != nil {
		log.Fatalf("unmarshal failed: %v\nraw: %s", err, string(raw))
	}
}
