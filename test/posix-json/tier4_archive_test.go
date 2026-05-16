package posixjson_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ramayac/korego/pkg/client"
	_ "github.com/ramayac/korego/pkg/gzip"
	_ "github.com/ramayac/korego/pkg/md5sum"
	_ "github.com/ramayac/korego/pkg/sha256sum"
	_ "github.com/ramayac/korego/pkg/tar"
)

func TestTier4_Tar(t *testing.T) {
	socket := startDaemon(t)
	c := client.Dial(socket, 5*time.Second)

	tmp := t.TempDir()

	// Create test files to archive
	file1 := filepath.Join(tmp, "archive_me.txt")
	file2 := filepath.Join(tmp, "also_archive.txt")
	os.WriteFile(file1, []byte("hello tar\n"), 0644)
	os.WriteFile(file2, []byte("world tar\n"), 0644)

	archivePath := filepath.Join(tmp, "test.tar")

	t.Run("tar -c creates archive", func(t *testing.T) {
		var result ResultWrapper
		err := c.Call(context.Background(), "korego.tar",
			map[string]interface{}{
				"flags": []interface{}{
					"-cf", archivePath,
					"-C", tmp,
					"archive_me.txt",
					"also_archive.txt",
				},
			},
			&result)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.ExitCode != 0 {
			t.Errorf("expected exit 0 from tar -c, got %d", result.ExitCode)
		}
		if _, statErr := os.Stat(archivePath); statErr != nil {
			t.Errorf("archive not created: %v", statErr)
		}
	})

	t.Run("tar -t lists archive contents", func(t *testing.T) {
		var result ResultWrapper
		err := c.Call(context.Background(), "korego.tar",
			map[string]interface{}{
				"flags": []interface{}{"-tf", archivePath},
			},
			&result)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.ExitCode != 0 {
			t.Errorf("expected exit 0 from tar -t, got %d", result.ExitCode)
		}
		// tar -t renders []TarFileStat directly (array)
		switch d := result.Data.(type) {
		case []interface{}:
			t.Logf("tar lists %d files", len(d))
		case nil:
			t.Errorf("tar list returned nil data")
		}
	})
}

func TestTier4_Gzip(t *testing.T) {
	socket := startDaemon(t)
	c := client.Dial(socket, 5*time.Second)

	tmp := t.TempDir()
	fpath := filepath.Join(tmp, "gzip_test.txt")
	if err := os.WriteFile(fpath, []byte("compress me\ncompress me\n"), 0644); err != nil {
		t.Fatal(err)
	}

	t.Run("gzip compresses file", func(t *testing.T) {
		var result ResultWrapper
		err := c.Call(context.Background(), "korego.gzip",
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
		gzPath := fpath + ".gz"
		if _, statErr := os.Stat(gzPath); statErr != nil {
			t.Errorf("compressed file not created: %v", statErr)
		}
		// gzip should return structured stats
		data, ok := result.Data.(map[string]interface{})
		if ok {
			if oldSize, hasOld := data["originalSize"]; hasOld {
				t.Logf("gzip original size: %v", oldSize)
			}
			if newSize, hasNew := data["compressedSize"]; hasNew {
				t.Logf("gzip compressed size: %v", newSize)
			}
		}
	})
}

func TestTier4_Sha256sum(t *testing.T) {
	socket := startDaemon(t)
	c := client.Dial(socket, 5*time.Second)

	tmp := t.TempDir()
	fpath := filepath.Join(tmp, "hash_test.txt")
	if err := os.WriteFile(fpath, []byte("sha256 hash me\n"), 0644); err != nil {
		t.Fatal(err)
	}

	t.Run("sha256sum computes hash", func(t *testing.T) {
		var result ResultWrapper
		err := c.Call(context.Background(), "korego.sha256sum",
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
		data, ok := result.Data.([]interface{})
		if !ok || len(data) == 0 {
			t.Fatalf("expected non-empty array, got %T", result.Data)
		}
		entry, ok := data[0].(map[string]interface{})
		if !ok {
			t.Fatalf("expected map entry, got %T", data[0])
		}
		if hash, ok := entry["hash"]; !ok || hash == "" {
			t.Errorf("expected non-empty 'hash' in sha256sum output")
		}
		if file, ok := entry["file"]; !ok || file == "" {
			t.Errorf("expected non-empty 'file' in sha256sum output")
		}
	})
}

func TestTier4_Md5sum(t *testing.T) {
	socket := startDaemon(t)
	c := client.Dial(socket, 5*time.Second)

	tmp := t.TempDir()
	fpath := filepath.Join(tmp, "md5_test.txt")
	if err := os.WriteFile(fpath, []byte("md5 hash me\n"), 0644); err != nil {
		t.Fatal(err)
	}

	t.Run("md5sum computes hash", func(t *testing.T) {
		var result ResultWrapper
		err := c.Call(context.Background(), "korego.md5sum",
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
		data, ok := result.Data.([]interface{})
		if !ok || len(data) == 0 {
			t.Fatalf("expected non-empty array, got %T", result.Data)
		}
		entry, ok := data[0].(map[string]interface{})
		if !ok {
			t.Fatalf("expected map entry, got %T", data[0])
		}
		if hash, ok := entry["hash"]; !ok || hash == "" {
			t.Errorf("expected non-empty 'hash' in md5sum output")
		}
	})
}
