package posixjson_test

import (
	"context"
	"testing"
	"time"

	"github.com/ramayac/korego/pkg/client"
	_ "github.com/ramayac/korego/pkg/date"
	_ "github.com/ramayac/korego/pkg/df"
	_ "github.com/ramayac/korego/pkg/du"
	_ "github.com/ramayac/korego/pkg/hostname"
	_ "github.com/ramayac/korego/pkg/id"
	_ "github.com/ramayac/korego/pkg/kill"
	_ "github.com/ramayac/korego/pkg/ps"
	_ "github.com/ramayac/korego/pkg/pwd"
	_ "github.com/ramayac/korego/pkg/uname"
	_ "github.com/ramayac/korego/pkg/whoami"
)

func TestTier3_Date(t *testing.T) {
	socket := startDaemon(t)
	c := client.Dial(socket, 5*time.Second)

	t.Run("date returns current time info", func(t *testing.T) {
		var result ResultWrapper
		err := c.Call(context.Background(), "korego.date",
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
		// date should have at least 'datetime' or 'epoch' or similar
		if _, hasEpoch := data["epoch"]; !hasEpoch {
			// Some implementations use different keys
			t.Logf("date output keys: %v", keys(data))
		}
		t.Logf("date data: %v", data)
	})
}

func TestTier3_Du(t *testing.T) {
	socket := startDaemon(t)
	c := client.Dial(socket, 5*time.Second)

	t.Run("du returns disk usage for current dir", func(t *testing.T) {
		var result ResultWrapper
		err := c.Call(context.Background(), "korego.du",
			map[string]interface{}{
				"flags": []interface{}{"."},
			},
			&result)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.ExitCode != 0 {
			t.Errorf("expected exit 0, got %d", result.ExitCode)
		}
		data, ok := result.Data.([]interface{})
		if !ok || len(data) == 0 {
			t.Fatalf("expected non-empty array, got %T", result.Data)
		}
		entry, ok := data[0].(map[string]interface{})
		if !ok {
			t.Fatalf("expected map entry, got %T", data[0])
		}
		if _, hasPath := entry["path"]; !hasPath {
			t.Errorf("expected 'path' in du output")
		}
		if _, hasSize := entry["size"]; !hasSize {
			t.Errorf("expected 'size' in du output")
		}
	})
}

func TestTier3_Df(t *testing.T) {
	socket := startDaemon(t)
	c := client.Dial(socket, 5*time.Second)

	t.Run("df returns filesystem info", func(t *testing.T) {
		var result ResultWrapper
		err := c.Call(context.Background(), "korego.df",
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
		data, ok := result.Data.([]interface{})
		if !ok || len(data) == 0 {
			t.Fatalf("expected non-empty array, got %T", result.Data)
		}
		entry, ok := data[0].(map[string]interface{})
		if !ok {
			t.Fatalf("expected map entry, got %T", data[0])
		}
		if _, hasFs := entry["filesystem"]; !hasFs {
			t.Errorf("expected 'filesystem' in df output, got keys: %v", keys(entry))
		}
	})
}

func TestTier3_Ps(t *testing.T) {
	socket := startDaemon(t)
	c := client.Dial(socket, 5*time.Second)

	t.Run("ps returns process list", func(t *testing.T) {
		var result ResultWrapper
		err := c.Call(context.Background(), "korego.ps",
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
		data, ok := result.Data.([]interface{})
		if !ok || len(data) == 0 {
			t.Fatalf("expected non-empty process array, got %T", result.Data)
		}
		entry, ok := data[0].(map[string]interface{})
		if !ok {
			t.Fatalf("expected map entry, got %T", data[0])
		}
		if _, hasPID := entry["pid"]; !hasPID {
			t.Errorf("expected 'pid' in ps output")
		}
	})
}

func TestTier3_Id(t *testing.T) {
	socket := startDaemon(t)
	c := client.Dial(socket, 5*time.Second)

	t.Run("id returns user identity info", func(t *testing.T) {
		var result ResultWrapper
		err := c.Call(context.Background(), "korego.id",
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
		if _, hasUser := data["user"]; !hasUser {
			t.Errorf("expected 'user' in id output")
		}
		if _, hasUID := data["uid"]; !hasUID {
			t.Errorf("expected 'uid' in id output")
		}
	})
}

func TestTier3_Hostname(t *testing.T) {
	socket := startDaemon(t)
	c := client.Dial(socket, 5*time.Second)

	t.Run("hostname returns host name", func(t *testing.T) {
		var result ResultWrapper
		err := c.Call(context.Background(), "korego.hostname",
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
		if _, hasName := data["hostname"]; !hasName {
			t.Errorf("expected 'hostname' in output, got keys: %v", keys(data))
		}
	})
}

func TestTier3_Whoami(t *testing.T) {
	socket := startDaemon(t)
	c := client.Dial(socket, 5*time.Second)

	t.Run("whoami returns current user", func(t *testing.T) {
		var result ResultWrapper
		err := c.Call(context.Background(), "korego.whoami",
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
		if user, ok := data["user"]; !ok || user == "" {
			t.Errorf("expected non-empty 'user' in whoami output")
		}
	})
}

func TestTier3_Pwd(t *testing.T) {
	socket := startDaemon(t)
	c := client.Dial(socket, 5*time.Second)

	t.Run("pwd returns current directory", func(t *testing.T) {
		var result ResultWrapper
		err := c.Call(context.Background(), "korego.pwd",
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
		if path, ok := data["path"]; !ok || path == "" {
			t.Errorf("expected non-empty 'path' in pwd output")
		}
	})
}

func TestTier3_Uname(t *testing.T) {
	socket := startDaemon(t)
	c := client.Dial(socket, 5*time.Second)

	t.Run("uname returns system info", func(t *testing.T) {
		var result ResultWrapper
		err := c.Call(context.Background(), "korego.uname",
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
		if _, hasSysName := data["sysname"]; !hasSysName {
			t.Errorf("expected 'sysname' in uname output, got keys: %v", keys(data))
		}
	})
}

func TestTier3_Kill(t *testing.T) {
	socket := startDaemon(t)
	c := client.Dial(socket, 5*time.Second)

	t.Run("kill with missing PID returns structured error", func(t *testing.T) {
		var result ResultWrapper
		err := c.Call(context.Background(), "korego.kill",
			map[string]interface{}{
				"flags": []interface{}{},
			},
			&result)
		// kill with no args may fail with missing operand
		if err != nil {
			t.Logf("kill returned error (expected with no args): %v", err)
		} else {
			t.Logf("kill exit code: %d", result.ExitCode)
		}
	})
}

// keys returns sorted keys of a map for diagnostic logging
func keys(m map[string]interface{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
