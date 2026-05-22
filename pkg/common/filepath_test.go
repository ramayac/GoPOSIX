package common

import (
	"os"
	"testing"
)

func TestResolvePath_EmptyCwd(t *testing.T) {
	// When cwd is empty, the path is returned as-is.
	if got := ResolvePath("", "foo.txt"); got != "foo.txt" {
		t.Errorf("expected 'foo.txt', got %q", got)
	}
	if got := ResolvePath("", "/absolute/path"); got != "/absolute/path" {
		t.Errorf("expected '/absolute/path', got %q", got)
	}
}

func TestResolvePath_Absolute(t *testing.T) {
	// Absolute paths are returned as-is regardless of cwd.
	if got := ResolvePath("/some/cwd", "/etc/hosts"); got != "/etc/hosts" {
		t.Errorf("expected '/etc/hosts', got %q", got)
	}
}

func TestResolvePath_Stdin(t *testing.T) {
	// "-" is special and returned as-is.
	if got := ResolvePath("/some/cwd", "-"); got != "-" {
		t.Errorf("expected '-', got %q", got)
	}
}

func TestResolvePath_Relative(t *testing.T) {
	// Relative path joined with cwd.
	if got := ResolvePath("/home/user", "docs/file.txt"); got != "/home/user/docs/file.txt" {
		t.Errorf("expected '/home/user/docs/file.txt', got %q", got)
	}
}

func TestResolvePath_Dot(t *testing.T) {
	// "." joined with cwd.
	wd, _ := os.Getwd()
	got := ResolvePath(wd, ".")
	if got != wd {
		t.Errorf("expected cwd %q, got %q", wd, got)
	}
}

func TestResolvePath_RelativeWithDotDot(t *testing.T) {
	// "../foo" joined with cwd.
	got := ResolvePath("/a/b", "../c")
	if got != "/a/c" {
		t.Errorf("expected '/a/c', got %q", got)
	}
}
