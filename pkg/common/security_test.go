package common

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSecurePath(t *testing.T) {
	tests := []struct {
		target  string
		baseDir string
		wantErr bool
	}{
		{"test.txt", "/app/sandbox", false},
		{"./test.txt", "/app/sandbox", false},
		{"sub/dir/test.txt", "/app/sandbox", false},
		{"../sandbox/test.txt", "/app/sandbox", false},
		{"../../app/sandbox/test.txt", "/app/sandbox", false},

		{"../outside.txt", "/app/sandbox", true},
		{"../../etc/shadow", "/app/sandbox", true},
		{"/etc/shadow", "/app/sandbox", true},
		{"/app/sandbox_other", "/app/sandbox", true},

		{"/etc/shadow", "/", false},
		{"../../etc/shadow", "/", false},
	}

	for _, tt := range tests {
		t.Run(tt.target+"_in_"+tt.baseDir, func(t *testing.T) {
			_, err := SecurePath(tt.target, tt.baseDir)
			if (err != nil) != tt.wantErr {
				t.Errorf("SecurePath(%q, %q) error = %v, wantErr %v", tt.target, tt.baseDir, err, tt.wantErr)
			}
		})
	}
}

func TestSecurePathSymlinks(t *testing.T) {
	// Create a temp directory structure:
	//   tmpdir/
	//     sandbox/          ← base directory
	//       legit/           ← real subdirectory
	//         file.txt
	//       escape -> /etc   ← symlink pointing outside base
	//       inner -> legit   ← symlink pointing inside base
	//     outside/           ← outside the base
	tmpDir := t.TempDir()
	sandbox := filepath.Join(tmpDir, "sandbox")
	outsideDir := filepath.Join(tmpDir, "outside")
	legitDir := filepath.Join(sandbox, "legit")
	escapeLink := filepath.Join(sandbox, "escape")
	innerLink := filepath.Join(sandbox, "inner")

	for _, d := range []string{sandbox, outsideDir, legitDir} {
		if err := os.MkdirAll(d, 0755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(legitDir, "file.txt"), []byte("ok"), 0644); err != nil {
		t.Fatal(err)
	}
	// Symlink pointing outside the base (to outsideDir, which is a safe stand-in for /etc).
	if err := os.Symlink(outsideDir, escapeLink); err != nil {
		t.Fatal(err)
	}
	// Symlink pointing inside the base.
	if err := os.Symlink(legitDir, innerLink); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		target  string
		wantErr bool
	}{
		// Normal paths still work.
		{"normal file", "legit/file.txt", false},
		{"normal dir", "legit", false},

		// Symlink inside base — allowed.
		{"symlink inside base to file", "inner/file.txt", false},
		{"symlink inside base to dir", "inner", false},

		// Symlink outside base — blocked.
		{"symlink to outside", "escape", true},

		// Non-existent leaf through symlink outside base — blocked.
		{"new file via escape symlink", "escape/newfile.txt", true},

		// Non-existent leaf inside base — allowed.
		{"new file in legit", "legit/newfile.txt", false},
		{"new file via inner symlink", "inner/newfile.txt", false},

		// Non-existent deep path through escape symlink — blocked.
		{"deep new file via escape", "escape/sub/deep/nope.txt", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := SecurePath(tt.target, sandbox)
			if (err != nil) != tt.wantErr {
				t.Errorf("SecurePath(%q, %q) error = %v, wantErr %v", tt.target, sandbox, err, tt.wantErr)
			}
		})
	}

	// Verify the resolved path for the inner symlink points to the real location.
	t.Run("resolved path for inner symlink", func(t *testing.T) {
		got, err := SecurePath("inner/file.txt", sandbox)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Should resolve to the real path, not the symlink path.
		expected := filepath.Join(legitDir, "file.txt")
		if got != expected {
			t.Errorf("resolved path = %q, want %q", got, expected)
		}
	})
}
