package common

import (
	"path/filepath"
)

// ResolvePath resolves a relative path against the context-specific cwd.
// If cwd is empty, the path is absolute, or the path is "-", the original path is returned.
func ResolvePath(cwd, path string) string {
	if cwd == "" || filepath.IsAbs(path) || path == "-" {
		return path
	}
	return filepath.Join(cwd, path)
}
