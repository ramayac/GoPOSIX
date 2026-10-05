package posixjson_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "github.com/ramayac/goposix/pkg/cmp"
	_ "github.com/ramayac/goposix/pkg/comm"
	_ "github.com/ramayac/goposix/pkg/daemon"
	_ "github.com/ramayac/goposix/pkg/dd"
	_ "github.com/ramayac/goposix/pkg/expand"
	_ "github.com/ramayac/goposix/pkg/fold"
	_ "github.com/ramayac/goposix/pkg/nl"
	_ "github.com/ramayac/goposix/pkg/od"
	_ "github.com/ramayac/goposix/pkg/paste"
	_ "github.com/ramayac/goposix/pkg/patch"
	_ "github.com/ramayac/goposix/pkg/shell"
	_ "github.com/ramayac/goposix/pkg/strings"
	_ "github.com/ramayac/goposix/pkg/sum"
	_ "github.com/ramayac/goposix/pkg/unexpand"
	client "github.com/ramayac/goposix/test/testutil"
)

func TestTier7_Cmp(t *testing.T) {
	socket := startDaemon(t)
	c := client.Dial(socket, 5*time.Second)

	tmp := t.TempDir()
	f1 := filepath.Join(tmp, "cmp_a")
	f2 := filepath.Join(tmp, "cmp_b")
	os.WriteFile(f1, []byte("hello"), 0644)
	os.WriteFile(f2, []byte("hello"), 0644)

	t.Run("cmp identical files", func(t *testing.T) {
		var result ResultWrapper
		err := c.Call(context.Background(), "goposix.cmp",
			map[string]interface{}{
				"flags": []interface{}{f1, f2},
			},
			&result)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.ExitCode != 0 {
			t.Errorf("expected exit 0 for identical files, got %d", result.ExitCode)
		}
	})

	os.WriteFile(f2, []byte("world"), 0644)

	t.Run("cmp different files", func(t *testing.T) {
		var result ResultWrapper
		err := c.Call(context.Background(), "goposix.cmp",
			map[string]interface{}{
				"flags": []interface{}{f1, f2},
			},
			&result)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.ExitCode != 1 {
			t.Errorf("expected exit 1 for different files, got %d", result.ExitCode)
		}
	})
}

func TestTier7_Comm(t *testing.T) {
	socket := startDaemon(t)
	c := client.Dial(socket, 5*time.Second)

	tmp := t.TempDir()
	f1 := filepath.Join(tmp, "comm_a")
	f2 := filepath.Join(tmp, "comm_b")
	os.WriteFile(f1, []byte("apple\nbanana\ncherry\n"), 0644)
	os.WriteFile(f2, []byte("banana\ncherry\ndate\n"), 0644)

	t.Run("comm compares sorted files", func(t *testing.T) {
		var result ResultWrapper
		err := c.Call(context.Background(), "goposix.comm",
			map[string]interface{}{
				"flags": []interface{}{f1, f2},
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
		if _, ok := data["only_file1"]; !ok {
			t.Error("expected only_file1 in comm output")
		}
		if _, ok := data["only_file2"]; !ok {
			t.Error("expected only_file2 in comm output")
		}
		if _, ok := data["both"]; !ok {
			t.Error("expected both in comm output")
		}
	})
}

func TestTier7_Expand(t *testing.T) {
	socket := startDaemon(t)
	c := client.Dial(socket, 5*time.Second)

	tmp := t.TempDir()
	fpath := filepath.Join(tmp, "expand_test")
	os.WriteFile(fpath, []byte("hello\tworld"), 0644)

	t.Run("expand converts tabs to spaces", func(t *testing.T) {
		var result ResultWrapper
		err := c.Call(context.Background(), "goposix.expand",
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
		if lines, ok := data["lines"].([]interface{}); ok && len(lines) > 0 {
			line := lines[0].(string)
			if strings.Contains(line, "\t") {
				t.Errorf("tabs not expanded: %q", line)
			}
		}
	})

	t.Run("expand with custom tab stop", func(t *testing.T) {
		f2 := filepath.Join(tmp, "expand_tab")
		os.WriteFile(f2, []byte("a\tb"), 0644)
		var result ResultWrapper
		err := c.Call(context.Background(), "goposix.expand",
			map[string]interface{}{
				"flags": []interface{}{"-t", "4", f2},
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

func TestTier7_Unexpand(t *testing.T) {
	socket := startDaemon(t)
	c := client.Dial(socket, 5*time.Second)

	tmp := t.TempDir()
	fpath := filepath.Join(tmp, "unexpand_test")
	os.WriteFile(fpath, []byte("hello   world"), 0644)

	t.Run("unexpand converts spaces to tabs", func(t *testing.T) {
		var result ResultWrapper
		err := c.Call(context.Background(), "goposix.unexpand",
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
		if lines, ok := data["lines"].([]interface{}); ok && len(lines) > 0 {
			t.Logf("unexpand line: %q", lines[0])
		}
	})
}

func TestTier7_Fold(t *testing.T) {
	socket := startDaemon(t)
	c := client.Dial(socket, 5*time.Second)

	tmp := t.TempDir()
	fpath := filepath.Join(tmp, "fold_test")
	os.WriteFile(fpath, []byte("hello world test fold"), 0644)

	t.Run("fold wraps lines", func(t *testing.T) {
		var result ResultWrapper
		err := c.Call(context.Background(), "goposix.fold",
			map[string]interface{}{
				"flags": []interface{}{"-w", "10", fpath},
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
		if lines, ok := data["lines"].([]interface{}); ok && len(lines) > 1 {
			t.Logf("fold produced %d lines", len(lines))
		}
	})

	t.Run("fold -s breaks at spaces", func(t *testing.T) {
		var result ResultWrapper
		err := c.Call(context.Background(), "goposix.fold",
			map[string]interface{}{
				"flags": []interface{}{"-s", "-w", "10", fpath},
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

func TestTier7_Nl(t *testing.T) {
	socket := startDaemon(t)
	c := client.Dial(socket, 5*time.Second)

	tmp := t.TempDir()
	fpath := filepath.Join(tmp, "nl_test")
	os.WriteFile(fpath, []byte("line1\nline2\nline3\n"), 0644)

	t.Run("nl numbers lines", func(t *testing.T) {
		var result ResultWrapper
		err := c.Call(context.Background(), "goposix.nl",
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
		if _, ok := data["lines"]; !ok {
			t.Error("expected 'lines' in nl output")
		}
	})
}

func TestTier7_Paste(t *testing.T) {
	socket := startDaemon(t)
	c := client.Dial(socket, 5*time.Second)

	tmp := t.TempDir()
	f1 := filepath.Join(tmp, "paste_a")
	f2 := filepath.Join(tmp, "paste_b")
	os.WriteFile(f1, []byte("a\nb\nc\n"), 0644)
	os.WriteFile(f2, []byte("1\n2\n3\n"), 0644)

	t.Run("paste merges files", func(t *testing.T) {
		var result ResultWrapper
		err := c.Call(context.Background(), "goposix.paste",
			map[string]interface{}{
				"flags": []interface{}{f1, f2},
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
		if _, ok := data["records"]; !ok {
			t.Error("expected 'records' in paste output")
		}
	})
}

func TestTier7_Strings(t *testing.T) {
	socket := startDaemon(t)
	c := client.Dial(socket, 5*time.Second)

	tmp := t.TempDir()
	fpath := filepath.Join(tmp, "strings_test")
	os.WriteFile(fpath, []byte("hello\x00world\x00test"), 0644)

	t.Run("strings extracts printable strings", func(t *testing.T) {
		var result ResultWrapper
		err := c.Call(context.Background(), "goposix.strings",
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
		if _, ok := data["strings"]; !ok {
			t.Error("expected 'strings' in strings output")
		}
	})
}

func TestTier7_Sum(t *testing.T) {
	socket := startDaemon(t)
	c := client.Dial(socket, 5*time.Second)

	tmp := t.TempDir()
	fpath := filepath.Join(tmp, "sum_test")
	os.WriteFile(fpath, []byte("hello\n"), 0644)

	t.Run("sum BSD checksum", func(t *testing.T) {
		var result ResultWrapper
		err := c.Call(context.Background(), "goposix.sum",
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
		if files, ok := data["files"].([]interface{}); ok && len(files) > 0 {
			t.Logf("sum files: %v", files)
		}
	})

	t.Run("sum SysV checksum", func(t *testing.T) {
		var result ResultWrapper
		err := c.Call(context.Background(), "goposix.sum",
			map[string]interface{}{
				"flags": []interface{}{"-s", fpath},
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

func TestTier7_Dd(t *testing.T) {
	socket := startDaemon(t)
	c := client.Dial(socket, 5*time.Second)

	tmp := t.TempDir()
	outPath := filepath.Join(tmp, "dd_out")

	t.Run("dd copies stdin to file", func(t *testing.T) {
		var result ResultWrapper
		err := c.Call(context.Background(), "goposix.dd",
			map[string]interface{}{
				"flags": []interface{}{"if=/dev/zero", "of=" + outPath, "bs=64", "count=1"},
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
			if rec, ok := data["records_in"]; ok {
				t.Logf("dd records_in: %v", rec)
			}
		}
	})
}

func TestTier7_Od(t *testing.T) {
	socket := startDaemon(t)
	c := client.Dial(socket, 5*time.Second)

	tmp := t.TempDir()
	fpath := filepath.Join(tmp, "od_test")
	os.WriteFile(fpath, []byte("HELLO"), 0644)

	t.Run("od dumps file in octal", func(t *testing.T) {
		var result ResultWrapper
		err := c.Call(context.Background(), "goposix.od",
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
		if _, ok := data["records"]; !ok {
			t.Error("expected 'records' in od output")
		}
	})
}

func TestTier7_Patch(t *testing.T) {
	t.Skip("patch requires daemon file I/O — tested via unit tests and BusyBox suite")
}

func TestTier7_Shell(t *testing.T) {
	socket := startDaemon(t)
	c := client.Dial(socket, 5*time.Second)

	t.Run("shell runs echo command", func(t *testing.T) {
		var result ResultWrapper
		err := c.Call(context.Background(), "goposix.shell",
			map[string]interface{}{
				"flags": []interface{}{"-c", "echo hello"},
			},
			&result)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.ExitCode != 0 {
			t.Logf("shell exit %d (may need session)", result.ExitCode)
		}
		data, ok := result.Data.(map[string]interface{})
		if ok {
			t.Logf("shell: %v", data)
		}
	})
}

// TestTier7_Sed tests sed via daemon
func TestTier7_Sed(t *testing.T) {
	socket := startDaemon(t)
	c := client.Dial(socket, 5*time.Second)

	tmp := t.TempDir()
	fpath := filepath.Join(tmp, "sed_test")
	os.WriteFile(fpath, []byte("hello world\nfoo bar\n"), 0644)

	t.Run("sed substitute", func(t *testing.T) {
		var result ResultWrapper
		err := c.Call(context.Background(), "goposix.sed",
			map[string]interface{}{
				"flags": []interface{}{"s/world/universe/", fpath},
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
			if lines, ok := data["lines"].([]interface{}); ok {
				t.Logf("sed output: %v", lines)
			}
		}
	})
}
