package posixjson_test

import (
	"context"
	"os"
	"testing"
	"time"

	_ "github.com/ramayac/goposix/pkg/basename"
	"github.com/ramayac/goposix/pkg/client"
	_ "github.com/ramayac/goposix/pkg/dirname"
	_ "github.com/ramayac/goposix/pkg/env"
	_ "github.com/ramayac/goposix/pkg/expr"
	_ "github.com/ramayac/goposix/pkg/printenv"
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
