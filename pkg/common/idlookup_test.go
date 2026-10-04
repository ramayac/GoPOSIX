package common

import "testing"

func TestLookupGIDNumeric(t *testing.T) {
	if got := LookupGID("12345"); got != 12345 {
		t.Fatalf("LookupGID(12345) = %d, want 12345", got)
	}
}

func TestLookupGIDByName(t *testing.T) {
	// The root group exists on every POSIX system.
	if got := LookupGID("root"); got != 0 {
		t.Fatalf("LookupGID(root) = %d, want 0", got)
	}
}

func TestLookupGIDUnknown(t *testing.T) {
	if got := LookupGID("no-such-group-xyz"); got != -1 {
		t.Fatalf("LookupGID(unknown) = %d, want -1", got)
	}
}

func TestLookupUIDNumeric(t *testing.T) {
	if got := LookupUID("12345"); got != 12345 {
		t.Fatalf("LookupUID(12345) = %d, want 12345", got)
	}
}

func TestLookupUIDByName(t *testing.T) {
	// The root user exists on every POSIX system.
	if got := LookupUID("root"); got != 0 {
		t.Fatalf("LookupUID(root) = %d, want 0", got)
	}
}

func TestLookupUIDUnknown(t *testing.T) {
	if got := LookupUID("no-such-user-xyz"); got != -1 {
		t.Fatalf("LookupUID(unknown) = %d, want -1", got)
	}
}

func TestHumanSize(t *testing.T) {
	cases := []struct {
		n     int64
		round bool
		want  string
	}{
		{0, false, "0B"},
		{512, false, "512B"},
		{1024, false, "1.0K"},
		{1536, false, "1.5K"},
		{1048576, false, "1.0M"},
		{1280, false, "1.2K"}, // truncated, ls style
		{1280, true, "1.3K"},  // rounded, du style
		{1048576, true, "1.0M"},
		{1536 * 1024, true, "1.5M"},
	}
	for _, c := range cases {
		if got := HumanSize(c.n, c.round); got != c.want {
			t.Errorf("HumanSize(%d, %v) = %q, want %q", c.n, c.round, got, c.want)
		}
	}
}
