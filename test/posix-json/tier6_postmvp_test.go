package posixjson_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/ramayac/goposix/pkg/cksum"
	"github.com/ramayac/goposix/pkg/client"
	_ "github.com/ramayac/goposix/pkg/join"
	_ "github.com/ramayac/goposix/pkg/link"
	_ "github.com/ramayac/goposix/pkg/logger"
	_ "github.com/ramayac/goposix/pkg/logname"
	_ "github.com/ramayac/goposix/pkg/mkfifo"
	_ "github.com/ramayac/goposix/pkg/nice"
	_ "github.com/ramayac/goposix/pkg/nohup"
	_ "github.com/ramayac/goposix/pkg/split"
	_ "github.com/ramayac/goposix/pkg/tty"
	_ "github.com/ramayac/goposix/pkg/unlink"
	_ "github.com/ramayac/goposix/pkg/who"
)

func TestTier6_Link(t *testing.T) {
	socket := startDaemon(t)
	c := client.Dial(socket, 5*time.Second)

	tmp := t.TempDir()
	src := filepath.Join(tmp, "link_src")
	dst := filepath.Join(tmp, "link_dst")
	if err := os.WriteFile(src, []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}

	t.Run("link creates hard link", func(t *testing.T) {
		var result ResultWrapper
		err := c.Call(context.Background(), "goposix.link",
			map[string]interface{}{
				"flags": []interface{}{src, dst},
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
		if s, ok := data["source"]; !ok || s != src {
			t.Errorf("expected source=%q, got %v", src, s)
		}
		if d, ok := data["target"]; !ok || d != dst {
			t.Errorf("expected target=%q, got %v", dst, d)
		}
	})

	t.Run("link nonexistent source fails", func(t *testing.T) {
		var result ResultWrapper
		err := c.Call(context.Background(), "goposix.link",
			map[string]interface{}{
				"flags": []interface{}{"/nonexistent/link_src", "/tmp/link_dst"},
			},
			&result)
		if err == nil && result.ExitCode == 0 {
			t.Error("expected non-zero exit for nonexistent source")
		}
	})
}

func TestTier6_Unlink(t *testing.T) {
	socket := startDaemon(t)
	c := client.Dial(socket, 5*time.Second)

	tmp := t.TempDir()
	fpath := filepath.Join(tmp, "unlink_test")
	if err := os.WriteFile(fpath, []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}

	t.Run("unlink removes file", func(t *testing.T) {
		var result ResultWrapper
		err := c.Call(context.Background(), "goposix.unlink",
			map[string]interface{}{
				"flags": []interface{}{fpath},
			},
			&result)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.ExitCode != 0 {
			t.Errorf("expected exit 0, got %d", result.ExitCode)
		}
		if _, statErr := os.Stat(fpath); statErr == nil {
			t.Error("file still exists after unlink")
		}
		data, ok := result.Data.(map[string]interface{})
		if !ok {
			t.Fatalf("expected map data, got %T", result.Data)
		}
		if r, ok := data["removed"]; !ok || r != fpath {
			t.Errorf("expected removed=%q, got %v", fpath, r)
		}
	})

	t.Run("unlink nonexistent file fails", func(t *testing.T) {
		var result ResultWrapper
		err := c.Call(context.Background(), "goposix.unlink",
			map[string]interface{}{
				"flags": []interface{}{"/nonexistent/unlink_file"},
			},
			&result)
		if err == nil && result.ExitCode == 0 {
			t.Error("expected non-zero exit for nonexistent file")
		}
	})
}

func TestTier6_Logname(t *testing.T) {
	socket := startDaemon(t)
	c := client.Dial(socket, 5*time.Second)

	t.Run("logname returns login name", func(t *testing.T) {
		var result ResultWrapper
		err := c.Call(context.Background(), "goposix.logname",
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
		if name, ok := data["logname"]; !ok || name == "" {
			t.Errorf("expected non-empty logname, got %v", name)
		}
		t.Logf("logname: %v", data["logname"])
	})
}

func TestTier6_Tty(t *testing.T) {
	socket := startDaemon(t)
	c := client.Dial(socket, 5*time.Second)

	t.Run("tty returns is_tty status", func(t *testing.T) {
		var result ResultWrapper
		err := c.Call(context.Background(), "goposix.tty",
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
		if _, ok := data["is_tty"]; !ok {
			t.Errorf("expected 'is_tty' in tty output, got keys: %v", keys(data))
		}
		t.Logf("tty is_tty: %v", data["is_tty"])
	})

	t.Run("tty -s returns exit code", func(t *testing.T) {
		var result ResultWrapper
		err := c.Call(context.Background(), "goposix.tty",
			map[string]interface{}{
				"flags": []interface{}{"-s"},
			},
			&result)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// -s returns 1 when not a tty (daemon is never a tty)
		if result.ExitCode != 1 {
			t.Logf("tty -s exit: %d (1 = not a tty)", result.ExitCode)
		}
	})
}

func TestTier6_Mkfifo(t *testing.T) {
	socket := startDaemon(t)
	c := client.Dial(socket, 5*time.Second)

	tmp := t.TempDir()
	fifoPath := filepath.Join(tmp, "test_fifo")

	t.Run("mkfifo creates named pipe", func(t *testing.T) {
		var result ResultWrapper
		err := c.Call(context.Background(), "goposix.mkfifo",
			map[string]interface{}{
				"flags": []interface{}{fifoPath},
			},
			&result)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.ExitCode != 0 {
			t.Errorf("expected exit 0, got %d", result.ExitCode)
		}
		info, statErr := os.Stat(fifoPath)
		if statErr != nil {
			t.Fatalf("fifo not created: %v", statErr)
		}
		if info.Mode()&os.ModeNamedPipe == 0 {
			t.Error("expected named pipe")
		}
		data, ok := result.Data.(map[string]interface{})
		if !ok {
			t.Fatalf("expected map data, got %T", result.Data)
		}
		if p, ok := data["path"]; !ok || p != fifoPath {
			t.Errorf("expected path=%q in output, got %v", fifoPath, p)
		}
	})

	t.Run("mkfifo with -m custom mode", func(t *testing.T) {
		customFifo := filepath.Join(tmp, "custom_fifo")
		var result ResultWrapper
		err := c.Call(context.Background(), "goposix.mkfifo",
			map[string]interface{}{
				"flags": []interface{}{"-m", "0600", customFifo},
			},
			&result)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.ExitCode != 0 {
			t.Errorf("expected exit 0, got %d", result.ExitCode)
		}
		info, _ := os.Stat(customFifo)
		if info.Mode().Perm() != 0600 {
			t.Errorf("expected mode 0600, got 0%o", info.Mode().Perm())
		}
	})
}

func TestTier6_Split(t *testing.T) {
	socket := startDaemon(t)
	c := client.Dial(socket, 5*time.Second)

	tmp := t.TempDir()
	inputFile := filepath.Join(tmp, "split_input")
	prefix := filepath.Join(tmp, "split_")
	if err := os.WriteFile(inputFile, []byte("1\n2\n3\n4\n5\n6\n7\n8\n"), 0644); err != nil {
		t.Fatal(err)
	}

	t.Run("split by lines with file input", func(t *testing.T) {
		var result ResultWrapper
		err := c.Call(context.Background(), "goposix.split",
			map[string]interface{}{
				"flags": []interface{}{"-l", "3", inputFile, prefix},
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
		if chunks, ok := data["chunks"]; !ok || chunks.(float64) < 2 {
			t.Errorf("expected chunks >= 2, got %v", chunks)
		}
		if files, ok := data["files"]; ok {
			t.Logf("split produced: %v", files)
		}
	})
}

func TestTier6_Join(t *testing.T) {
	socket := startDaemon(t)
	c := client.Dial(socket, 5*time.Second)

	tmp := t.TempDir()
	f1 := filepath.Join(tmp, "join_1")
	f2 := filepath.Join(tmp, "join_2")
	if err := os.WriteFile(f1, []byte("1:one\n2:two\n3:three\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(f2, []byte("1:alpha\n2:beta\n4:gamma\n"), 0644); err != nil {
		t.Fatal(err)
	}

	t.Run("join merges sorted files by key", func(t *testing.T) {
		var result ResultWrapper
		err := c.Call(context.Background(), "goposix.join",
			map[string]interface{}{
				"flags": []interface{}{"-t", ":", f1, f2},
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
		records, ok := data["records"].([]interface{})
		if !ok || len(records) < 1 {
			t.Errorf("expected non-empty records, got %v", records)
		}
		t.Logf("join records: %v", records)
	})

	t.Run("join -v1 returns unpaired lines", func(t *testing.T) {
		var result ResultWrapper
		err := c.Call(context.Background(), "goposix.join",
			map[string]interface{}{
				"flags": []interface{}{"-t", ":", "-v1", f1, f2},
			},
			&result)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.ExitCode != 0 {
			t.Errorf("expected exit 0, got %d", result.ExitCode)
		}
		data, _ := result.Data.(map[string]interface{})
		records, _ := data["records"].([]interface{})
		t.Logf("join -v1 records: %v", records)
	})
}

func TestTier6_Cksum(t *testing.T) {
	socket := startDaemon(t)
	c := client.Dial(socket, 5*time.Second)

	tmp := t.TempDir()
	fpath := filepath.Join(tmp, "cksum_test")
	if err := os.WriteFile(fpath, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	t.Run("cksum computes CRC-32", func(t *testing.T) {
		var result ResultWrapper
		err := c.Call(context.Background(), "goposix.cksum",
			map[string]interface{}{
				"flags": []interface{}{fpath},
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
		files, ok := data["files"].([]interface{})
		if !ok || len(files) == 0 {
			t.Fatalf("expected non-empty files array, got %v", files)
		}
		fileData := files[0].(map[string]interface{})
		if cksum, ok := fileData["checksum"]; !ok {
			t.Errorf("expected 'checksum' in file data, got keys: %v", keys(fileData))
		} else {
			// "hello" = 3287646509
			if cksum.(float64) != 3287646509 {
				t.Errorf("expected checksum 3287646509, got %v", cksum)
			}
		}
		if bytes, ok := fileData["bytes"]; ok {
			if bytes.(float64) != 5 {
				t.Errorf("expected 5 bytes, got %v", bytes)
			}
		}
	})

	t.Run("cksum empty file", func(t *testing.T) {
		emptyPath := filepath.Join(tmp, "cksum_empty")
		if err := os.WriteFile(emptyPath, []byte{}, 0644); err != nil {
			t.Fatal(err)
		}
		var result ResultWrapper
		err := c.Call(context.Background(), "goposix.cksum",
			map[string]interface{}{
				"flags": []interface{}{emptyPath},
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

func TestTier6_Logger(t *testing.T) {
	socket := startDaemon(t)
	c := client.Dial(socket, 5*time.Second)

	t.Run("logger submits message (may fail silently)", func(t *testing.T) {
		var result ResultWrapper
		err := c.Call(context.Background(), "goposix.logger",
			map[string]interface{}{
				"flags": []interface{}{"-t", "posixjson", "test message from posix-json"},
			},
			&result)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// logger may exit 0 even if syslog is unreachable
		if result.ExitCode != 0 {
			t.Logf("logger exit %d (syslog may be unavailable)", result.ExitCode)
		}
		data, ok := result.Data.(map[string]interface{})
		if ok {
			t.Logf("logger data: tag=%v, message=%v", data["tag"], data["message"])
		}
	})

	t.Run("logger with priority flag", func(t *testing.T) {
		var result ResultWrapper
		err := c.Call(context.Background(), "goposix.logger",
			map[string]interface{}{
				"flags": []interface{}{"-p", "user.info", "-t", "testpri", "priority test"},
			},
			&result)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		data, ok := result.Data.(map[string]interface{})
		if ok {
			if pri, ok := data["priority"]; ok {
				t.Logf("logger priority: %v", pri)
			}
		}
	})
}

func TestTier6_Who(t *testing.T) {
	socket := startDaemon(t)
	c := client.Dial(socket, 5*time.Second)

	t.Run("who returns user list", func(t *testing.T) {
		var result ResultWrapper
		err := c.Call(context.Background(), "goposix.who",
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
		if _, ok := data["users"]; !ok {
			t.Errorf("expected 'users' in who output, got keys: %v", keys(data))
		}
		t.Logf("who users: %v", data["users"])
	})

	t.Run("who -q returns quick mode", func(t *testing.T) {
		var result ResultWrapper
		err := c.Call(context.Background(), "goposix.who",
			map[string]interface{}{
				"flags": []interface{}{"-q"},
			},
			&result)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.ExitCode != 0 {
			t.Errorf("expected exit 0, got %d", result.ExitCode)
		}
	})

	t.Run("who -H returns heading", func(t *testing.T) {
		var result ResultWrapper
		err := c.Call(context.Background(), "goposix.who",
			map[string]interface{}{
				"flags": []interface{}{"-H"},
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

func TestTier6_Nice(t *testing.T) {
	socket := startDaemon(t)
	c := client.Dial(socket, 5*time.Second)

	t.Run("nice runs command", func(t *testing.T) {
		var result ResultWrapper
		err := c.Call(context.Background(), "goposix.nice",
			map[string]interface{}{
				"flags": []interface{}{"true"},
			},
			&result)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// nice may fail with EPERM without CAP_SYS_NICE
		if result.ExitCode != 0 {
			t.Logf("nice exit %d (may need CAP_SYS_NICE)", result.ExitCode)
		}
		data, ok := result.Data.(map[string]interface{})
		if ok {
			t.Logf("nice: adjustment=%v command=%v exit_code=%v",
				data["adjustment"], data["command"], data["exit_code"])
		}
	})
}

func TestTier6_Nohup(t *testing.T) {
	socket := startDaemon(t)
	c := client.Dial(socket, 5*time.Second)

	t.Run("nohup runs command", func(t *testing.T) {
		var result ResultWrapper
		err := c.Call(context.Background(), "goposix.nohup",
			map[string]interface{}{
				"flags": []interface{}{"true"},
			},
			&result)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.ExitCode != 0 {
			t.Errorf("expected exit 0, got %d", result.ExitCode)
		}
		data, ok := result.Data.(map[string]interface{})
		if ok {
			t.Logf("nohup: command=%v exit_code=%v",
				data["command"], data["exit_code"])
		}
	})
}
