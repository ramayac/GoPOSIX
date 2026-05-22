package which

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWhichLookup(t *testing.T) {
	// Create a temporary directory structure for testing lookups
	tmpDir, err := os.MkdirTemp("", "which-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	binDir1 := filepath.Join(tmpDir, "bin1")
	binDir2 := filepath.Join(tmpDir, "bin2")
	if err := os.MkdirAll(binDir1, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(binDir2, 0755); err != nil {
		t.Fatal(err)
	}

	// Create dummy executable files
	file1 := filepath.Join(binDir1, "mycmd")
	file2 := filepath.Join(binDir2, "mycmd")
	file3 := filepath.Join(binDir2, "othercmd")

	if err := os.WriteFile(file1, []byte(""), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file2, []byte(""), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file3, []byte(""), 0755); err != nil {
		t.Fatal(err)
	}

	pathDirs := []string{binDir1, binDir2}

	// Test single lookup (first match)
	matches := findCommand("mycmd", pathDirs, false)
	if len(matches) != 1 || matches[0] != file1 {
		t.Errorf("expected %q, got %v", file1, matches)
	}

	// Test all lookup (-a)
	matchesAll := findCommand("mycmd", pathDirs, true)
	if len(matchesAll) != 2 || matchesAll[0] != file1 || matchesAll[1] != file2 {
		t.Errorf("expected [%q, %q], got %v", file1, file2, matchesAll)
	}

	// Test lookup with slash
	matchesSlash := findCommand(file3, pathDirs, false)
	if len(matchesSlash) != 1 || matchesSlash[0] != file3 {
		t.Errorf("expected %q, got %v", file3, matchesSlash)
	}
}

func TestWhichCLISingle(t *testing.T) {
	var stdout, stderr bytes.Buffer
	// "ls" is guaranteed to exist on any standard POSIX test system
	code := run([]string{"ls"}, nil, &stdout, &stderr, "")
	if code != 0 {
		t.Errorf("expected exit 0, got %d, stderr: %q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "ls") {
		t.Errorf("expected ls path in stdout, got: %q", stdout.String())
	}
}

func TestWhichCLINotFound(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"nonexistent_command_12345"}, nil, &stdout, &stderr, "")
	if code != 1 {
		t.Errorf("expected exit 1 for nonexistent command, got %d", code)
	}
	if stdout.Len() != 0 {
		t.Errorf("expected no stdout, got %q", stdout.String())
	}
}

func TestWhichCLIJSON(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"--json", "ls"}, nil, &stdout, &stderr, "")
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
	if !strings.Contains(stdout.String(), `"matches"`) {
		t.Errorf("expected JSON matches envelope, got: %q", stdout.String())
	}
}
