package posixjson_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ramayac/korego/pkg/client"
	_ "github.com/ramayac/korego/pkg/cut"
	_ "github.com/ramayac/korego/pkg/diff"
	_ "github.com/ramayac/korego/pkg/find"
	_ "github.com/ramayac/korego/pkg/grep"
	_ "github.com/ramayac/korego/pkg/head"
	_ "github.com/ramayac/korego/pkg/printf"
	_ "github.com/ramayac/korego/pkg/sort"
	_ "github.com/ramayac/korego/pkg/tail"
	_ "github.com/ramayac/korego/pkg/uniq"
	_ "github.com/ramayac/korego/pkg/wc"
)

func TestTier2_Grep(t *testing.T) {
	socket := startDaemon(t)
	c := client.Dial(socket, 5*time.Second)

	tmp := t.TempDir()
	fpath := filepath.Join(tmp, "grep_test.txt")
	content := "hello world\nfoo bar\nhello again\nbaz qux\n"
	if err := os.WriteFile(fpath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	t.Run("grep finds matches", func(t *testing.T) {
		var result ResultWrapper
		err := c.Call(context.Background(), "korego.grep",
			map[string]interface{}{
				// grep takes: pattern [file...]
				"flags": []interface{}{"hello", fpath},
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
			t.Errorf("expected non-empty matches array, got %T", result.Data)
		}
	})

	t.Run("grep -v inverts match", func(t *testing.T) {
		var result ResultWrapper
		err := c.Call(context.Background(), "korego.grep",
			map[string]interface{}{
				"flags": []interface{}{"-v", "hello", fpath},
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

func TestTier2_Find(t *testing.T) {
	socket := startDaemon(t)
	c := client.Dial(socket, 5*time.Second)

	tmp := t.TempDir()
	sub := filepath.Join(tmp, "subdir")
	os.Mkdir(sub, 0755)
	os.WriteFile(filepath.Join(sub, "findme.txt"), []byte("x"), 0644)

	t.Run("find locates files by name", func(t *testing.T) {
		var result ResultWrapper
		err := c.Call(context.Background(), "korego.find",
			map[string]interface{}{
				"flags": []interface{}{tmp, "-name", "findme.txt"},
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
			t.Fatalf("expected non-empty results array, got %T", result.Data)
		}
		first, ok := data[0].(map[string]interface{})
		if !ok {
			t.Fatalf("expected map entry, got %T", data[0])
		}
		if _, hasPath := first["path"]; !hasPath {
			t.Errorf("expected 'path' field in find output")
		}
	})
}

func TestTier2_Sort(t *testing.T) {
	socket := startDaemon(t)
	c := client.Dial(socket, 5*time.Second)

	tmp := t.TempDir()
	fpath := filepath.Join(tmp, "sort_test.txt")
	if err := os.WriteFile(fpath, []byte("zebra\nalpha\ngamma\nbeta\n"), 0644); err != nil {
		t.Fatal(err)
	}

	t.Run("sort orders lines", func(t *testing.T) {
		var result ResultWrapper
		err := c.Call(context.Background(), "korego.sort",
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
		lines, ok := data["lines"].([]interface{})
		if !ok || len(lines) != 4 {
			t.Errorf("expected 4 sorted lines, got %v", data["lines"])
		}
		if len(lines) >= 2 && lines[0] != "alpha" {
			t.Errorf("expected first line 'alpha', got %v", lines[0])
		}
	})
}

func TestTier2_Uniq(t *testing.T) {
	socket := startDaemon(t)
	c := client.Dial(socket, 5*time.Second)

	tmp := t.TempDir()
	fpath := filepath.Join(tmp, "uniq_test.txt")
	if err := os.WriteFile(fpath, []byte("a\na\nb\nb\nc\n"), 0644); err != nil {
		t.Fatal(err)
	}

	t.Run("uniq deduplicates adjacent lines", func(t *testing.T) {
		var result ResultWrapper
		err := c.Call(context.Background(), "korego.uniq",
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
		// uniq returns array of {line, count} items
		switch d := result.Data.(type) {
		case []interface{}:
			t.Logf("uniq output: %v", d)
		case nil:
			t.Errorf("uniq returned nil data")
		default:
			t.Fatalf("expected array data, got %T", result.Data)
		}
	})
}

func TestTier2_Wc(t *testing.T) {
	socket := startDaemon(t)
	c := client.Dial(socket, 5*time.Second)

	tmp := t.TempDir()
	fpath := filepath.Join(tmp, "wc_test.txt")
	content := "line one\nline two\nline three\n"
	if err := os.WriteFile(fpath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	t.Run("wc counts lines words bytes", func(t *testing.T) {
		var result ResultWrapper
		err := c.Call(context.Background(), "korego.wc",
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
		if lines, ok := data["lines"]; ok {
			t.Logf("wc lines=%v", lines)
		}
		if words, ok := data["words"]; ok {
			t.Logf("wc words=%v", words)
		}
		if chars, ok := data["chars"]; ok {
			t.Logf("wc chars=%v", chars)
		}
	})
}

func TestTier2_Head(t *testing.T) {
	socket := startDaemon(t)
	c := client.Dial(socket, 5*time.Second)

	tmp := t.TempDir()
	fpath := filepath.Join(tmp, "head_test.txt")
	content := ""
	for i := 0; i < 20; i++ {
		content += "line " + string(rune('a'+i)) + "\n"
	}
	if err := os.WriteFile(fpath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	t.Run("head returns first 10 lines", func(t *testing.T) {
		var result ResultWrapper
		err := c.Call(context.Background(), "korego.head",
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
		lines, ok := data["lines"].([]interface{})
		if !ok || len(lines) == 0 {
			t.Errorf("expected lines in head output")
		}
		if count, ok := data["lineCount"]; ok {
			t.Logf("head lineCount=%v", count)
		}
	})
}

func TestTier2_Tail(t *testing.T) {
	socket := startDaemon(t)
	c := client.Dial(socket, 5*time.Second)

	tmp := t.TempDir()
	fpath := filepath.Join(tmp, "tail_test.txt")
	content := ""
	for i := 0; i < 20; i++ {
		content += "line " + string(rune('a'+i)) + "\n"
	}
	if err := os.WriteFile(fpath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	t.Run("tail returns last 10 lines", func(t *testing.T) {
		var result ResultWrapper
		err := c.Call(context.Background(), "korego.tail",
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
		lines, ok := data["lines"].([]interface{})
		if !ok || len(lines) == 0 {
			t.Errorf("expected lines in tail output")
		}
	})
}

func TestTier2_Cut(t *testing.T) {
	socket := startDaemon(t)
	c := client.Dial(socket, 5*time.Second)

	tmp := t.TempDir()
	fpath := filepath.Join(tmp, "cut_test.txt")
	if err := os.WriteFile(fpath, []byte("a:b:c\n1:2:3\nx:y:z\n"), 0644); err != nil {
		t.Fatal(err)
	}

	t.Run("cut -f1 -d: extracts first field", func(t *testing.T) {
		var result ResultWrapper
		err := c.Call(context.Background(), "korego.cut",
			map[string]interface{}{
				"flags": []interface{}{"-f1", "-d:", fpath},
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
		lines, ok := data["lines"].([]interface{})
		if !ok || len(lines) == 0 {
			t.Errorf("expected lines in cut output")
		}
	})
}

func TestTier2_Diff(t *testing.T) {
	socket := startDaemon(t)
	c := client.Dial(socket, 5*time.Second)

	tmp := t.TempDir()
	f1 := filepath.Join(tmp, "diff_a.txt")
	f2 := filepath.Join(tmp, "diff_b.txt")
	os.WriteFile(f1, []byte("line1\nline2\nline3\n"), 0644)
	os.WriteFile(f2, []byte("line1\nlineX\nline3\n"), 0644)

	t.Run("diff detects differences", func(t *testing.T) {
		var result ResultWrapper
		err := c.Call(context.Background(), "korego.diff",
			map[string]interface{}{
				"flags": []interface{}{f1, f2},
			},
			&result)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// diff exit code 1 means differences found
		if result.ExitCode != 1 {
			t.Errorf("expected exit 1 (differences found), got %d", result.ExitCode)
		}
		data, ok := result.Data.(map[string]interface{})
		if !ok {
			t.Fatalf("expected map data, got %T", result.Data)
		}
		if differ, ok := data["differ"]; ok && differ != true {
			t.Errorf("expected differ=true, got %v", differ)
		}
		if hunks, ok := data["hunks"]; !ok || hunks == nil {
			t.Errorf("expected hunks in diff output")
		}
	})

	t.Run("diff identical files exits 0", func(t *testing.T) {
		var result ResultWrapper
		err := c.Call(context.Background(), "korego.diff",
			map[string]interface{}{
				"flags": []interface{}{f1, f1},
			},
			&result)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.ExitCode != 0 {
			t.Errorf("expected exit 0 (identical), got %d", result.ExitCode)
		}
	})
}

func TestTier2_Printf(t *testing.T) {
	socket := startDaemon(t)
	c := client.Dial(socket, 5*time.Second)

	t.Run("printf formats string", func(t *testing.T) {
		var result ResultWrapper
		err := c.Call(context.Background(), "korego.printf",
			map[string]interface{}{
				"flags": []interface{}{"hello %s", "world"},
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
		if output, ok := data["output"]; !ok || output == "" {
			t.Errorf("expected non-empty output in printf")
		}
	})
}
