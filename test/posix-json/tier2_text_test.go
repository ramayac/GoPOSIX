package posixjson_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ramayac/goposix/pkg/client"
	_ "github.com/ramayac/goposix/pkg/cut"
	_ "github.com/ramayac/goposix/pkg/diff"
	_ "github.com/ramayac/goposix/pkg/find"
	_ "github.com/ramayac/goposix/pkg/grep"
	_ "github.com/ramayac/goposix/pkg/head"
	_ "github.com/ramayac/goposix/pkg/printf"
	_ "github.com/ramayac/goposix/pkg/sort"
	_ "github.com/ramayac/goposix/pkg/tail"
	_ "github.com/ramayac/goposix/pkg/tee"
	_ "github.com/ramayac/goposix/pkg/tr"
	_ "github.com/ramayac/goposix/pkg/uniq"
	_ "github.com/ramayac/goposix/pkg/wc"
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
		err := c.Call(context.Background(), "goposix.grep",
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
		err := c.Call(context.Background(), "goposix.grep",
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
		err := c.Call(context.Background(), "goposix.find",
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
		err := c.Call(context.Background(), "goposix.sort",
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
		err := c.Call(context.Background(), "goposix.uniq",
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
		err := c.Call(context.Background(), "goposix.wc",
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
		err := c.Call(context.Background(), "goposix.head",
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
		err := c.Call(context.Background(), "goposix.tail",
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
		err := c.Call(context.Background(), "goposix.cut",
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
		err := c.Call(context.Background(), "goposix.diff",
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
		err := c.Call(context.Background(), "goposix.diff",
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
		err := c.Call(context.Background(), "goposix.printf",
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

func TestTier2_Tee(t *testing.T) {
	socket := startDaemon(t)
	c := client.Dial(socket, 5*time.Second)

	tmp := t.TempDir()
	f1 := filepath.Join(tmp, "tee_1.txt")
	f2 := filepath.Join(tmp, "tee_2.txt")
	stdinVal := "tee test input\nline 2\n"

	t.Run("tee writes to stdout and files in structured JSON mode", func(t *testing.T) {
		var result ResultWrapper
		err := c.Call(context.Background(), "goposix.tee",
			map[string]interface{}{
				"flags": []interface{}{f1, f2},
				"stdin": stdinVal,
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
		if bytesWritten, ok := data["bytesWritten"]; !ok || int64(bytesWritten.(float64)) != int64(len(stdinVal)) {
			t.Errorf("expected bytesWritten %d, got %v", len(stdinVal), bytesWritten)
		}

		// Verify file 1 content
		b1, err := os.ReadFile(f1)
		if err != nil {
			t.Fatal(err)
		}
		if string(b1) != stdinVal {
			t.Errorf("expected file 1 to have %q, got %q", stdinVal, string(b1))
		}

		// Verify file 2 content
		b2, err := os.ReadFile(f2)
		if err != nil {
			t.Fatal(err)
		}
		if string(b2) != stdinVal {
			t.Errorf("expected file 2 to have %q, got %q", stdinVal, string(b2))
		}
	})

	t.Run("tee -a appends to existing files", func(t *testing.T) {
		extraStdin := "appended line\n"
		var result ResultWrapper
		err := c.Call(context.Background(), "goposix.tee",
			map[string]interface{}{
				"flags": []interface{}{"-a", f1},
				"stdin": extraStdin,
			},
			&result)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.ExitCode != 0 {
			t.Errorf("expected exit 0, got %d", result.ExitCode)
		}

		b1, err := os.ReadFile(f1)
		if err != nil {
			t.Fatal(err)
		}
		expected := stdinVal + extraStdin
		if string(b1) != expected {
			t.Errorf("expected file 1 to have %q, got %q", expected, string(b1))
		}
	})

	t.Run("tee rawOutput returns raw stdout content", func(t *testing.T) {
		f3 := filepath.Join(tmp, "tee_3.txt")
		stdinValRaw := "raw stdout test\n"
		var result struct {
			ExitCode int    `json:"exitCode"`
			Stdout   string `json:"stdout"`
		}
		err := c.Call(context.Background(), "goposix.tee",
			map[string]interface{}{
				"rawOutput": true,
				"flags":     []interface{}{f3},
				"stdin":     stdinValRaw,
			},
			&result)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.ExitCode != 0 {
			t.Errorf("expected exit 0, got %d", result.ExitCode)
		}
		if result.Stdout != stdinValRaw {
			t.Errorf("expected raw stdout %q, got %q", stdinValRaw, result.Stdout)
		}

		b3, err := os.ReadFile(f3)
		if err != nil {
			t.Fatal(err)
		}
		if string(b3) != stdinValRaw {
			t.Errorf("expected file 3 to have %q, got %q", stdinValRaw, string(b3))
		}
	})
}

func TestTier2_Tr(t *testing.T) {
	socket := startDaemon(t)
	c := client.Dial(socket, 5*time.Second)

	t.Run("tr translates characters in structured JSON mode", func(t *testing.T) {
		var result ResultWrapper
		err := c.Call(context.Background(), "goposix.tr",
			map[string]interface{}{
				"flags": []interface{}{"a-z", "A-Z"},
				"stdin": "hello world\n",
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
			t.Fatalf("expected lines in result, got %v", data["lines"])
		}
		if lines[0] != "HELLO WORLD" {
			t.Errorf("expected translated line 'HELLO WORLD', got %q", lines[0])
		}
	})

	t.Run("tr -d deletes characters in structured JSON mode", func(t *testing.T) {
		var result ResultWrapper
		err := c.Call(context.Background(), "goposix.tr",
			map[string]interface{}{
				"flags": []interface{}{"-d", "aeiou"},
				"stdin": "beautiful day\n",
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
			t.Fatalf("expected lines in result, got %v", data["lines"])
		}
		if lines[0] != "btfl dy" {
			t.Errorf("expected 'btfl dy', got %q", lines[0])
		}
	})

	t.Run("tr -s squeezes repeating characters", func(t *testing.T) {
		var result ResultWrapper
		err := c.Call(context.Background(), "goposix.tr",
			map[string]interface{}{
				"flags": []interface{}{"-s", "o"},
				"stdin": "helloooo world\n",
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
			t.Fatalf("expected lines in result, got %v", data["lines"])
		}
		if lines[0] != "hello world" {
			t.Errorf("expected squeezed 'hello world', got %q", lines[0])
		}
	})

	t.Run("tr -c complements translation", func(t *testing.T) {
		var result ResultWrapper
		err := c.Call(context.Background(), "goposix.tr",
			map[string]interface{}{
				"flags": []interface{}{"-c", "a", "x"},
				"stdin": "abaca",
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
			t.Fatalf("expected lines in result, got %v", data["lines"])
		}
		if lines[0] != "axaxa" {
			t.Errorf("expected complemented 'axaxa', got %q", lines[0])
		}
	})

	t.Run("tr rawOutput mode", func(t *testing.T) {
		var result struct {
			ExitCode int    `json:"exitCode"`
			Stdout   string `json:"stdout"`
		}
		err := c.Call(context.Background(), "goposix.tr",
			map[string]interface{}{
				"rawOutput": true,
				"flags":     []interface{}{"a-z", "A-Z"},
				"stdin":     "hello raw\n",
			},
			&result)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.ExitCode != 0 {
			t.Errorf("expected exit 0, got %d", result.ExitCode)
		}
		if result.Stdout != "HELLO RAW\n" {
			t.Errorf("expected 'HELLO RAW\\n', got %q", result.Stdout)
		}
	})
}
