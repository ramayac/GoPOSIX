package integration

import (
	"encoding/json"
	"net"
	"os"
	"testing"
)

// TestRPCWorkflow simulates a multi-step RPC workflow.
func TestRPCWorkflow(t *testing.T) {
	conn, err := net.Dial("unix", "/tmp/goposix.sock")
	if err != nil {
		t.Skipf("daemon not running at /tmp/goposix.sock, skipping RPC test: %v", err)
		return
	}
	defer conn.Close()

	dec := json.NewDecoder(conn)

	// Helper to send request
	rpcCall := func(method string, params interface{}) map[string]interface{} {
		req := map[string]interface{}{
			"jsonrpc": "2.0",
			"method":  method,
			"params":  params,
			"id":      1,
		}
		b, _ := json.Marshal(req)
		conn.Write(b)

		var res map[string]interface{}
		err := dec.Decode(&res)
		if err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if res["error"] != nil {
			t.Fatalf("RPC error on %s: %v", method, res["error"])
		}
		return res
	}

	// 1. Create session
	res := rpcCall("goposix.session.create", nil)
	sessionData := res["result"].(map[string]interface{})
	sessionId := sessionData["sessionId"].(string)

	// 2. Create temp directory on host to test with
	tmpDir, err := os.MkdirTemp("", "goposix-rpc-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// 3. Set CWD
	rpcCall("goposix.session.setCwd", map[string]interface{}{
		"sessionId": sessionId,
		"path":      tmpDir,
	})

	// 4. Write a file (via shell.exec)
	rpcCall("goposix.shell.exec", map[string]interface{}{
		"sessionId": sessionId,
		"script":    "echo 'hello world' > test.txt",
	})

	// 5. Read the file back (cat)
	res = rpcCall("goposix.cat", map[string]interface{}{
		"sessionId": sessionId,
		"path":      "test.txt",
	})
	if res["result"] == nil {
		t.Fatalf("Expected cat result, got nil")
	}

	// 6. Grep for a pattern
	res = rpcCall("goposix.grep", map[string]interface{}{
		"sessionId": sessionId,
		"flags":     []string{"hello"},
		"path":      "test.txt",
	})
	if res["result"] == nil {
		t.Fatalf("Expected grep result, got nil")
	}

	// 7. Get checksum (sha256sum)
	res = rpcCall("goposix.sha256sum", map[string]interface{}{
		"sessionId": sessionId,
		"path":      "test.txt",
	})
	if res["result"] == nil {
		t.Fatalf("Expected sha256sum result, got nil")
	}

	// 8. Destroy session
	rpcCall("goposix.session.destroy", map[string]interface{}{
		"sessionId": sessionId,
	})
}
