package posixjson_test

// Tier 9 — JSON contract tests for the Phase 28 audit (5d. JSON Changes).
//
// These are the commands that had no JSON-RPC daemon test before the audit:
// bc, mount, hexdump, makedevs, mdev, mkfs.minix, wget, xxd, rx. The shell
// daemon assertion is added here too (F15): before the fix, goposix.shell
// treated the daemon-prepended --json as a file name.

import (
	"context"
	"os"
	"testing"
	"time"

	_ "github.com/ramayac/goposix/pkg/bc"
	_ "github.com/ramayac/goposix/pkg/hexdump"
	_ "github.com/ramayac/goposix/pkg/mkfs_minix"
	_ "github.com/ramayac/goposix/pkg/xxd"
	client "github.com/ramayac/goposix/test/testutil"
)

func TestTier9_Bc(t *testing.T) {
	socket := startDaemon(t)
	c := client.Dial(socket, 5*time.Second)

	t.Run("bc evaluates stdin over RPC", func(t *testing.T) {
		var result ResultWrapper
		err := c.Call(context.Background(), "goposix.bc",
			map[string]interface{}{"stdin": "1+2\n"}, &result)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.ExitCode != 0 {
			t.Fatalf("expected exit 0, got %d", result.ExitCode)
		}
		data, ok := result.Data.(map[string]interface{})
		if !ok {
			t.Fatalf("expected map data, got %T", result.Data)
		}
		lines, ok := data["lines"].([]interface{})
		if !ok || len(lines) != 1 || lines[0].(string) != "3" {
			t.Errorf("expected lines [3], got %v", data["lines"])
		}
	})
}

func TestTier9_Mount(t *testing.T) {
	socket := startDaemon(t)
	c := client.Dial(socket, 5*time.Second)

	t.Run("mount lists mounts over RPC", func(t *testing.T) {
		var result ResultWrapper
		err := c.Call(context.Background(), "goposix.mount",
			map[string]interface{}{"flags": []interface{}{}}, &result)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.ExitCode != 0 {
			t.Fatalf("expected exit 0, got %d", result.ExitCode)
		}
		data, ok := result.Data.(map[string]interface{})
		if !ok {
			t.Fatalf("expected map data, got %T", result.Data)
		}
		mounts, ok := data["mounts"].([]interface{})
		if !ok || len(mounts) == 0 {
			t.Fatalf("expected non-empty mounts array, got %v", data["mounts"])
		}
		first, ok := mounts[0].(map[string]interface{})
		if !ok {
			t.Fatalf("expected map entry, got %T", mounts[0])
		}
		for _, key := range []string{"device", "mountpoint", "fstype", "options"} {
			if _, ok := first[key]; !ok {
				t.Errorf("expected %q key in mount entry, got %v", key, first)
			}
		}
	})
}

func TestTier9_Hexdump(t *testing.T) {
	socket := startDaemon(t)
	c := client.Dial(socket, 5*time.Second)

	t.Run("hexdump dumps stdin over RPC", func(t *testing.T) {
		var result ResultWrapper
		err := c.Call(context.Background(), "goposix.hexdump",
			map[string]interface{}{"stdin": "hello"}, &result)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.ExitCode != 0 {
			t.Fatalf("expected exit 0, got %d", result.ExitCode)
		}
		data, ok := result.Data.(map[string]interface{})
		if !ok {
			t.Fatalf("expected map data, got %T", result.Data)
		}
		lines, ok := data["lines"].([]interface{})
		if !ok || len(lines) == 0 {
			t.Errorf("expected non-empty lines, got %v", data["lines"])
		}
	})
}

func TestTier9_Makedevs(t *testing.T) {
	socket := startDaemon(t)
	c := client.Dial(socket, 5*time.Second)

	t.Run("makedevs reports the device table over RPC", func(t *testing.T) {
		dir := t.TempDir()
		table := dir + "/devtable.txt"
		if err := os.WriteFile(table, []byte("devname c 0600 0 0 1 3\n"), 0644); err != nil {
			t.Fatal(err)
		}
		var result ResultWrapper
		// Without root the node creation fails, so exit 1 is expected;
		// the JSON envelope must still be present and describe the table.
		err := c.Call(context.Background(), "goposix.makedevs",
			map[string]interface{}{"flags": []interface{}{"-d", table, dir}}, &result)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.ExitCode != 1 {
			t.Errorf("expected exit 1 (no root), got %d", result.ExitCode)
		}
		data, ok := result.Data.(map[string]interface{})
		if !ok {
			t.Fatalf("expected map data, got %T", result.Data)
		}
		for _, key := range []string{"table", "rootdir", "created", "failedCount"} {
			if _, ok := data[key]; !ok {
				t.Errorf("expected %q key in data, got %v", key, data)
			}
		}
		if data["failedCount"].(float64) < 1 {
			t.Errorf("expected failedCount >= 1, got %v", data["failedCount"])
		}
	})
}

func TestTier9_Mdev(t *testing.T) {
	socket := startDaemon(t)
	c := client.Dial(socket, 5*time.Second)

	t.Run("mdev dry-run discovers devices over RPC", func(t *testing.T) {
		var result ResultWrapper
		err := c.Call(context.Background(), "goposix.mdev",
			map[string]interface{}{"flags": []interface{}{"-d"}}, &result)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.ExitCode != 0 {
			t.Fatalf("expected exit 0, got %d", result.ExitCode)
		}
		data, ok := result.Data.(map[string]interface{})
		if !ok {
			t.Fatalf("expected map data, got %T", result.Data)
		}
		devices, ok := data["devices"].([]interface{})
		if !ok || len(devices) == 0 {
			t.Fatalf("expected non-empty devices array, got %v", data["devices"])
		}
		first, ok := devices[0].(map[string]interface{})
		if !ok {
			t.Fatalf("expected map entry, got %T", devices[0])
		}
		for _, key := range []string{"name", "type", "major", "minor", "path"} {
			if _, ok := first[key]; !ok {
				t.Errorf("expected %q key in device entry, got %v", key, first)
			}
		}
	})
}

func TestTier9_MkfsMinix(t *testing.T) {
	socket := startDaemon(t)
	c := client.Dial(socket, 5*time.Second)

	t.Run("mkfs.minix formats an image over RPC", func(t *testing.T) {
		img := t.TempDir() + "/minix.img"
		var result ResultWrapper
		err := c.Call(context.Background(), "goposix.mkfs.minix",
			map[string]interface{}{"flags": []interface{}{img, "1000"}}, &result)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.ExitCode != 0 {
			t.Fatalf("expected exit 0, got %d", result.ExitCode)
		}
		data, ok := result.Data.(map[string]interface{})
		if !ok {
			t.Fatalf("expected map data, got %T", result.Data)
		}
		for _, key := range []string{"inodes", "zones", "first_data_zone", "imap_blocks", "zmap_blocks"} {
			if _, ok := data[key]; !ok {
				t.Errorf("expected %q key in data, got %v", key, data)
			}
		}
	})
}

func TestTier9_Wget(t *testing.T) {
	socket := startDaemon(t)
	c := client.Dial(socket, 5*time.Second)

	t.Run("wget downloads over RPC", func(t *testing.T) {
		out := t.TempDir() + "/wget.html"
		var result ResultWrapper
		err := c.Call(context.Background(), "goposix.wget",
			map[string]interface{}{"flags": []interface{}{"-O", out, "https://example.com"}}, &result)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.ExitCode != 0 {
			t.Fatalf("expected exit 0, got %d", result.ExitCode)
		}
		data, ok := result.Data.(map[string]interface{})
		if !ok {
			t.Fatalf("expected map data, got %T", result.Data)
		}
		for _, key := range []string{"url", "output_file", "bytes_downloaded", "status_code"} {
			if _, ok := data[key]; !ok {
				t.Errorf("expected %q key in data, got %v", key, data)
			}
		}
		if data["status_code"].(float64) != 200 {
			t.Errorf("expected status_code 200, got %v", data["status_code"])
		}
	})
}

func TestTier9_Xxd(t *testing.T) {
	socket := startDaemon(t)
	c := client.Dial(socket, 5*time.Second)

	t.Run("xxd hex-dumps stdin over RPC", func(t *testing.T) {
		var result ResultWrapper
		err := c.Call(context.Background(), "goposix.xxd",
			map[string]interface{}{"stdin": "hi"}, &result)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.ExitCode != 0 {
			t.Fatalf("expected exit 0, got %d", result.ExitCode)
		}
		data, ok := result.Data.(map[string]interface{})
		if !ok {
			t.Fatalf("expected map data, got %T", result.Data)
		}
		lines, ok := data["lines"].([]interface{})
		if !ok || len(lines) == 0 {
			t.Errorf("expected non-empty lines, got %v", data["lines"])
		}
	})
}

func TestTier9_Rx(t *testing.T) {
	socket := startDaemon(t)
	c := client.Dial(socket, 5*time.Second)

	t.Run("rx without a sender reports a clean exit code over RPC", func(t *testing.T) {
		var result ResultWrapper
		err := c.Call(context.Background(), "goposix.rx",
			map[string]interface{}{"flags": []interface{}{t.TempDir() + "/rx.out"}, "stdin": "garbage"}, &result)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// No XMODEM sender on stdin: the handshake times out and rx
		// reports exit 1. The handshake bytes must not leak into stdout.
		if result.ExitCode != 1 {
			t.Errorf("expected exit 1, got %d", result.ExitCode)
		}
	})
}

func TestTier9_Shell(t *testing.T) {
	socket := startDaemon(t)
	c := client.Dial(socket, 5*time.Second)

	t.Run("shell returns the JSON envelope over RPC", func(t *testing.T) {
		var result ResultWrapper
		err := c.Call(context.Background(), "goposix.shell",
			map[string]interface{}{"flags": []interface{}{"-c", "echo hi"}}, &result)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.ExitCode != 0 {
			t.Fatalf("expected exit 0, got %d", result.ExitCode)
		}
		data, ok := result.Data.(map[string]interface{})
		if !ok {
			t.Fatalf("expected map data, got %T", result.Data)
		}
		if data["exitCode"].(float64) != 0 {
			t.Errorf("expected script exitCode 0, got %v", data["exitCode"])
		}
		if data["stdout"].(string) != "hi\n" {
			t.Errorf("expected captured stdout %q, got %q", "hi\n", data["stdout"])
		}
		if data["stderr"].(string) != "" {
			t.Errorf("expected empty captured stderr, got %q", data["stderr"])
		}
	})

	t.Run("shell script exit code survives the envelope", func(t *testing.T) {
		var result ResultWrapper
		err := c.Call(context.Background(), "goposix.shell",
			map[string]interface{}{"flags": []interface{}{"-c", "exit 7"}}, &result)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// The utility returns the script's exit code as its process exit
		// code; the script's own exit code also travels in data.exitCode.
		if result.ExitCode != 7 {
			t.Errorf("expected exit 7, got %d", result.ExitCode)
		}
		data, ok := result.Data.(map[string]interface{})
		if !ok {
			t.Fatalf("expected map data, got %T", result.Data)
		}
		if data["exitCode"].(float64) != 7 {
			t.Errorf("expected script exitCode 7, got %v", data["exitCode"])
		}
	})
}
