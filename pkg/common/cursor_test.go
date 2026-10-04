package common

import "testing"

func TestTokenCursor(t *testing.T) {
	c := NewTokenCursor([]string{"a", "b", "c"})

	if c.Done() {
		t.Fatal("fresh cursor should not be done")
	}
	if got := c.Peek(); got != "a" {
		t.Fatalf("Peek() = %q, want %q", got, "a")
	}
	if got := c.Next(); got != "a" {
		t.Fatalf("Next() = %q, want %q", got, "a")
	}
	if got := c.Peek(); got != "b" {
		t.Fatalf("Peek() = %q, want %q", got, "b")
	}
	if got := c.Next(); got != "b" {
		t.Fatalf("Next() = %q, want %q", got, "b")
	}
	if got := c.Next(); got != "c" {
		t.Fatalf("Next() = %q, want %q", got, "c")
	}
	if !c.Done() {
		t.Fatal("cursor should be done after consuming all tokens")
	}
	if got := c.Next(); got != "" {
		t.Fatalf("Next() after done = %q, want empty", got)
	}
}

func TestTokenCursorEmpty(t *testing.T) {
	c := NewTokenCursor(nil)
	if !c.Done() {
		t.Fatal("empty cursor should be done")
	}
	if got := c.Peek(); got != "" {
		t.Fatalf("Peek() = %q, want empty", got)
	}
}
