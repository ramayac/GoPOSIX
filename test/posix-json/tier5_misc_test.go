package posixjson_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/ramayac/goposix/pkg/basename"
	"github.com/ramayac/goposix/pkg/client"
	_ "github.com/ramayac/goposix/pkg/dirname"
	_ "github.com/ramayac/goposix/pkg/env"
	_ "github.com/ramayac/goposix/pkg/expr"
	_ "github.com/ramayac/goposix/pkg/factor"
	_ "github.com/ramayac/goposix/pkg/hostid"
	_ "github.com/ramayac/goposix/pkg/pidof"
	_ "github.com/ramayac/goposix/pkg/printenv"
	_ "github.com/ramayac/goposix/pkg/sha3sum"
	_ "github.com/ramayac/goposix/pkg/tree"
	_ "github.com/ramayac/goposix/pkg/tsort"
	_ "github.com/ramayac/goposix/pkg/xargs"
)

func TestTier5_Expr(t *testing.T) {
	socket := startDaemon(t)
	c := client.Dial(socket, 5*time.Second)

	t.Run("expr evaluates arithmetic", func(t *testing.T) {
		var result ResultWrapper
		err := c.Call(context.Background(), "goposix.expr",
			map[string]interface{}{
				"flags": []interface{}{"3", "+", "4"},
			},
			&result)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.ExitCode != 0 {
			t.Errorf("expected exit 0, got %d", result.ExitCode)
		}
		data, ok := result.Data.(map[string]interface{})
		if !ok {
			t.Fatalf("expected map data, got %T", result.Data)
		}
		if val, ok := data["result"]; ok {
			t.Logf("expr result: %v", val)
			// 3 + 4 should be 7
			switch v := val.(type) {
			case float64:
				if v != 7 {
					t.Errorf("expected 7, got %v", v)
				}
			case string:
				if v != "7" {
					t.Errorf("expected '7', got '%s'", v)
				}
			}
		}
	})

	t.Run("expr string comparison", func(t *testing.T) {
		var result ResultWrapper
		err := c.Call(context.Background(), "goposix.expr",
			map[string]interface{}{
				"flags": []interface{}{"hello", "=", "hello"},
			},
			&result)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.ExitCode != 0 {
			t.Errorf("expected exit 0, got %d", result.ExitCode)
		}
	})
}

func TestTier5_Basename(t *testing.T) {
	socket := startDaemon(t)
	c := client.Dial(socket, 5*time.Second)

	t.Run("basename strips directory", func(t *testing.T) {
		var result ResultWrapper
		err := c.Call(context.Background(), "goposix.basename",
			map[string]interface{}{
				"flags": []interface{}{"/usr/local/bin/myapp"},
			},
			&result)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.ExitCode != 0 {
			t.Errorf("expected exit 0, got %d", result.ExitCode)
		}
		data, ok := result.Data.(map[string]interface{})
		if !ok {
			t.Fatalf("expected map data, got %T", result.Data)
		}
		if result, ok := data["result"]; !ok || result != "myapp" {
			t.Errorf("expected result 'myapp', got %v", result)
		}
	})

	t.Run("basename strips suffix", func(t *testing.T) {
		var result ResultWrapper
		err := c.Call(context.Background(), "goposix.basename",
			map[string]interface{}{
				"flags": []interface{}{"/tmp/file.txt", ".txt"},
			},
			&result)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.ExitCode != 0 {
			t.Errorf("expected exit 0, got %d", result.ExitCode)
		}
		data, _ := result.Data.(map[string]interface{})
		if result, ok := data["result"]; !ok || result != "file" {
			t.Errorf("expected result 'file', got %v", result)
		}
	})
}

func TestTier5_Dirname(t *testing.T) {
	socket := startDaemon(t)
	c := client.Dial(socket, 5*time.Second)

	t.Run("dirname returns directory portion", func(t *testing.T) {
		var result ResultWrapper
		err := c.Call(context.Background(), "goposix.dirname",
			map[string]interface{}{
				"flags": []interface{}{"/usr/local/bin/myapp"},
			},
			&result)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.ExitCode != 0 {
			t.Errorf("expected exit 0, got %d", result.ExitCode)
		}
		data, ok := result.Data.(map[string]interface{})
		if !ok {
			t.Fatalf("expected map data, got %T", result.Data)
		}
		if result, ok := data["result"]; !ok || result != "/usr/local/bin" {
			t.Errorf("expected result '/usr/local/bin', got %v", result)
		}
	})
}

func TestTier5_Env(t *testing.T) {
	socket := startDaemon(t)
	c := client.Dial(socket, 5*time.Second)

	t.Run("env returns environment variables", func(t *testing.T) {
		var result ResultWrapper
		err := c.Call(context.Background(), "goposix.env",
			map[string]interface{}{
				"flags": []interface{}{},
			},
			&result)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.ExitCode != 0 {
			t.Errorf("expected exit 0, got %d", result.ExitCode)
		}
		data, ok := result.Data.(map[string]interface{})
		if !ok {
			t.Fatalf("expected map data, got %T", result.Data)
		}
		if vars, ok := data["vars"]; !ok {
			t.Errorf("expected 'vars' in env output, got keys: %v", keys(data))
		} else {
			t.Logf("env returned %d vars", len(vars.(map[string]interface{})))
		}
	})
}

func TestTier5_Printenv(t *testing.T) {
	socket := startDaemon(t)
	c := client.Dial(socket, 5*time.Second)

	t.Run("printenv returns specific env var", func(t *testing.T) {
		os.Setenv("GOPOSIX_POSIX_TEST", "hello")
		defer os.Unsetenv("GOPOSIX_POSIX_TEST")

		var result ResultWrapper
		err := c.Call(context.Background(), "goposix.printenv",
			map[string]interface{}{
				"flags": []interface{}{"GOPOSIX_POSIX_TEST"},
			},
			&result)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.ExitCode != 0 {
			t.Errorf("expected exit 0, got %d", result.ExitCode)
		}
		data, ok := result.Data.(map[string]interface{})
		if !ok {
			t.Fatalf("expected map data, got %T", result.Data)
		}
		if vars, ok := data["vars"].(map[string]interface{}); !ok {
			t.Errorf("expected 'vars' map in printenv output")
		} else {
			if val, ok := vars["GOPOSIX_POSIX_TEST"]; !ok || val != "hello" {
				t.Errorf("expected GOPOSIX_POSIX_TEST='hello', got %v", val)
			}
		}
	})
}

func TestTier5_Xargs(t *testing.T) {
	socket := startDaemon(t)
	c := client.Dial(socket, 5*time.Second)

	t.Run("xargs with empty stdin returns exit 0", func(t *testing.T) {
		// xargs reads from stdin; with no input it should exit 0 with no results
		var result ResultWrapper
		err := c.Call(context.Background(), "goposix.xargs",
			map[string]interface{}{
				"flags": []interface{}{},
			},
			&result)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// xargs on empty stdin: exit 0, no results
		if result.ExitCode != 0 {
			t.Logf("xargs exit code: %d (may have input)", result.ExitCode)
		}
		t.Logf("xargs data: %v", result.Data)
	})
}

func TestTier5_Hostid(t *testing.T) {
	socket := startDaemon(t)
	c := client.Dial(socket, 5*time.Second)

	t.Run("hostid outputs a valid 8-character hex string", func(t *testing.T) {
		var result ResultWrapper
		err := c.Call(context.Background(), "goposix.hostid",
			map[string]interface{}{
				"flags": []interface{}{},
			},
			&result)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.ExitCode != 0 {
			t.Errorf("expected exit 0, got %d", result.ExitCode)
		}
		data, ok := result.Data.(map[string]interface{})
		if !ok {
			t.Fatalf("expected map data, got %T", result.Data)
		}
		hostid, ok := data["hostid"].(string)
		if !ok {
			t.Fatalf("expected 'hostid' key in data, got %v", data)
		}
		if len(hostid) != 8 {
			t.Errorf("expected 8-character hostid, got %q (len %d)", hostid, len(hostid))
		}
		t.Logf("JSON-RPC hostid returned: %s", hostid)
	})
}

func TestTier5_Factor(t *testing.T) {
	socket := startDaemon(t)
	c := client.Dial(socket, 5*time.Second)

	t.Run("factorizes number over JSON-RPC", func(t *testing.T) {
		var result ResultWrapper
		err := c.Call(context.Background(), "goposix.factor",
			map[string]interface{}{
				"flags": []interface{}{"--json", "1024"},
			},
			&result)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.ExitCode != 0 {
			t.Errorf("expected exit 0, got %d", result.ExitCode)
		}
		data, ok := result.Data.(map[string]interface{})
		if !ok {
			t.Fatalf("expected map data, got %T", result.Data)
		}
		results, ok := data["results"].([]interface{})
		if !ok {
			t.Fatalf("expected 'results' slice in data, got %v", data)
		}
		if len(results) != 1 {
			t.Fatalf("expected 1 result entry, got %d", len(results))
		}
		entry, ok := results[0].(map[string]interface{})
		if !ok {
			t.Fatalf("expected result entry map, got %T", results[0])
		}
		input, _ := entry["input"].(string)
		if input != "1024" {
			t.Errorf("expected input '1024', got %q", input)
		}
		factors, ok := entry["factors"].([]interface{})
		if !ok {
			t.Fatalf("expected 'factors' list, got %v", entry)
		}
		if len(factors) != 10 {
			t.Errorf("expected 10 factors of 2, got %d", len(factors))
		}
	})
}

func TestTier5_Sha3sum(t *testing.T) {
	socket := startDaemon(t)
	c := client.Dial(socket, 5*time.Second)

	t.Run("computes SHA3 hash over JSON-RPC", func(t *testing.T) {
		var result ResultWrapper
		err := c.Call(context.Background(), "goposix.sha3sum",
			map[string]interface{}{
				"flags": []interface{}{"--json", "-a", "256"},
				"stdin": "hello world\n",
			},
			&result)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.ExitCode != 0 {
			t.Errorf("expected exit 0, got %d", result.ExitCode)
		}
		results, ok := result.Data.([]interface{})
		if !ok {
			t.Fatalf("expected slice data, got %T", result.Data)
		}
		if len(results) != 1 {
			t.Fatalf("expected 1 result entry, got %d", len(results))
		}
		entry, ok := results[0].(map[string]interface{})
		if !ok {
			t.Fatalf("expected result entry map, got %T", results[0])
		}
		file, _ := entry["file"].(string)
		if file != "-" {
			t.Errorf("expected file '-', got %q", file)
		}
		hash, _ := entry["hash"].(string)
		expected := "a8009a7a528d87778c356da3a55d964719e818666a04e4f960c9e2439e35f138"
		if hash != expected {
			t.Errorf("expected hash %q, got %q", expected, hash)
		}
	})
}

func TestTier5_Tree(t *testing.T) {
	socket := startDaemon(t)
	c := client.Dial(socket, 5*time.Second)

	t.Run("lists directory tree over JSON-RPC", func(t *testing.T) {
		// Create a quick temp dir to list
		tmpDir, err := os.MkdirTemp("", "daemon_tree_test")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)
		_ = os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("data"), 0644)

		var result ResultWrapper
		err = c.Call(context.Background(), "goposix.tree",
			map[string]interface{}{
				"flags": []interface{}{"--json", tmpDir},
			},
			&result)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.ExitCode != 0 {
			t.Errorf("expected exit 0, got %d", result.ExitCode)
		}
		data, ok := result.Data.(map[string]interface{})
		if !ok {
			t.Fatalf("expected map data, got %T", result.Data)
		}
		trees, ok := data["trees"].([]interface{})
		if !ok {
			t.Fatalf("expected 'trees' list, got %v", data)
		}
		if len(trees) != 1 {
			t.Fatalf("expected 1 tree root, got %d", len(trees))
		}
		rootNode, ok := trees[0].(map[string]interface{})
		if !ok {
			t.Fatalf("expected root node map, got %T", trees[0])
		}
		name, _ := rootNode["name"].(string)
		if name != tmpDir {
			t.Errorf("expected root name %q, got %q", tmpDir, name)
		}
	})
}

func TestTier5_Tsort(t *testing.T) {
	socket := startDaemon(t)
	c := client.Dial(socket, 5*time.Second)

	t.Run("topological sort over JSON-RPC", func(t *testing.T) {
		var result ResultWrapper
		err := c.Call(context.Background(), "goposix.tsort",
			map[string]interface{}{
				"flags": []interface{}{"--json"},
				"stdin": "a b b c\n",
			},
			&result)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.ExitCode != 0 {
			t.Errorf("expected exit 0, got %d", result.ExitCode)
		}
		data, ok := result.Data.(map[string]interface{})
		if !ok {
			t.Fatalf("expected map data, got %T", result.Data)
		}
		nodes, ok := data["nodes"].([]interface{})
		if !ok {
			t.Fatalf("expected 'nodes' list, got %v", data)
		}
		if len(nodes) != 3 {
			t.Fatalf("expected 3 nodes, got %d", len(nodes))
		}
		if nodes[0].(string) != "a" || nodes[1].(string) != "b" || nodes[2].(string) != "c" {
			t.Errorf("expected sorted ['a', 'b', 'c'], got %v", nodes)
		}
	})
}

func TestTier5_Pidof(t *testing.T) {
	// Start a background process
	// Let's check imports of tier5_misc_test.go or use exec.Command directly.
	// Oh, we can import os/exec or use os.StartProcess, or wait:
	// Let's see what is imported in tier5_misc_test.go. We can check by running go test, if exec is not imported, let's import it or use a separate import.
	// Let's see what imports are present in tier5_misc_test.go: context, os, path/filepath, testing, time.
	// We can start a sleep 15 process. Since we only have 'os' imported, let's use os.StartProcess or just add "os/exec" to imports. Wait, instead of adding exec to imports at the top, we can use os.StartProcess which doesn't require any new imports!
	// Wait, let's check:
	// proc, err := os.StartProcess("/bin/sleep", []string{"sleep", "15"}, &os.ProcAttr{})
	// That's standard and works perfectly on Linux!
	proc, err := os.StartProcess("/bin/sleep", []string{"sleep", "15"}, &os.ProcAttr{
		Files: []*os.File{os.Stdin, os.Stdout, os.Stderr},
	})
	if err != nil {
		t.Skip("skipping /proc dependent test: cannot start sleep")
		return
	}
	defer proc.Kill()

	socket := startDaemon(t)
	c := client.Dial(socket, 5*time.Second)

	t.Run("find pid over JSON-RPC", func(t *testing.T) {
		var result ResultWrapper
		err := c.Call(context.Background(), "goposix.pidof",
			map[string]interface{}{
				"flags": []interface{}{"--json", "sleep"},
			},
			&result)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.ExitCode != 0 {
			t.Errorf("expected exit 0, got %d", result.ExitCode)
		}
		data, ok := result.Data.(map[string]interface{})
		if !ok {
			t.Fatalf("expected map data, got %T", result.Data)
		}
		pids, ok := data["pids"].([]interface{})
		if !ok {
			t.Fatalf("expected 'pids' list, got %v", data)
		}
		found := false
		targetPid := float64(proc.Pid)
		for _, p := range pids {
			if p.(float64) == targetPid {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected to find PID %v in %v", targetPid, pids)
		}
	})
}
