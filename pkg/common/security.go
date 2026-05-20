package common

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// SecurePath resolves a target path against a base directory and ensures
// that the resulting path does not escape the base directory via ../ traversal,
// absolute paths, or symlink traversal.
// If baseDir is "/", all paths are allowed.
func SecurePath(target, baseDir string) (string, error) {
	if baseDir == "" {
		baseDir = "/"
	}

	absBase, err := filepath.Abs(baseDir)
	if err != nil {
		return "", err
	}
	absBase = filepath.Clean(absBase)

	// Resolve symlinks in the base directory itself (e.g., /app/sandbox → /tmp/real).
	realBase, err := resolveSymlinks(absBase)
	if err != nil {
		return "", err
	}

	// Root base directory allows anything.
	if realBase == "/" || realBase == filepath.VolumeName(realBase)+"\\" {
		var absTarget string
		if filepath.IsAbs(target) {
			absTarget = filepath.Clean(target)
		} else {
			absTarget = filepath.Clean(filepath.Join(absBase, target))
		}
		return absTarget, nil
	}

	var absTarget string
	if filepath.IsAbs(target) {
		absTarget = filepath.Clean(target)
	} else {
		absTarget = filepath.Clean(filepath.Join(absBase, target))
	}

	// Resolve symlinks in the target path. For paths that do not exist yet
	// (e.g., creating a new file), resolve the deepest existing parent and
	// append the non-existent tail components.
	realTarget, err := resolveSymlinks(absTarget)
	if err != nil {
		return "", err
	}

	basePrefix := realBase
	if !strings.HasSuffix(basePrefix, string(filepath.Separator)) {
		basePrefix += string(filepath.Separator)
	}

	targetWithSep := realTarget
	if !strings.HasSuffix(targetWithSep, string(filepath.Separator)) {
		targetWithSep += string(filepath.Separator)
	}

	if realTarget != realBase && !strings.HasPrefix(targetWithSep, basePrefix) {
		return "", errors.New("path traversal detected: target escapes base directory")
	}

	return realTarget, nil
}

// resolveSymlinks resolves all symlinks in path. If the path does not exist,
// it walks up to the deepest existing parent, resolves its symlinks, and
// appends the non-existent components. This allows SecurePath to validate
// paths that are being created (e.g., touch, mkdir, shell redirects) while
// still catching symlink escapes in existing parent directories.
func resolveSymlinks(path string) (string, error) {
	resolved, err := filepath.EvalSymlinks(path)
	if err == nil {
		return filepath.Clean(resolved), nil
	}
	if !os.IsNotExist(err) {
		return "", err
	}
	// Path does not exist — walk up to deepest existing ancestor.
	missing := ""
	current := path
	for {
		resolved, err := filepath.EvalSymlinks(current)
		if err == nil {
			if missing == "" {
				return filepath.Clean(resolved), nil
			}
			return filepath.Clean(filepath.Join(resolved, missing)), nil
		}
		if !os.IsNotExist(err) {
			return "", err
		}
		base := filepath.Base(current)
		if missing == "" {
			missing = base
		} else {
			missing = filepath.Join(base, missing)
		}
		parent := filepath.Dir(current)
		if parent == current {
			// Reached the filesystem root and nothing in the chain exists.
			return filepath.Clean(path), nil
		}
		current = parent
	}
}
