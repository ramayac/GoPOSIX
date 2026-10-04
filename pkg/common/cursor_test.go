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

func TestTokenCursorLookahead(t *testing.T) {
	c := NewTokenCursor([]string{"a", "b", "c"})

	if !c.Has(0) {
		t.Fatal("Has(0) should be true at start")
	}
	if got := c.Lookahead(0); got != "a" {
		t.Fatalf("Lookahead(0) = %q, want a", got)
	}
	if got := c.Lookahead(1); got != "b" {
		t.Fatalf("Lookahead(1) = %q, want b", got)
	}
	if got := c.Lookahead(2); got != "c" {
		t.Fatalf("Lookahead(2) = %q, want c", got)
	}
	if c.Has(3) {
		t.Fatal("Has(3) should be false")
	}
	if got := c.Lookahead(3); got != "" {
		t.Fatalf("Lookahead(3) = %q, want empty", got)
	}

	// Advance and check lookahead moves with the cursor.
	c.Next()
	if got := c.Lookahead(1); got != "c" {
		t.Fatalf("Lookahead(1) after advance = %q, want c", got)
	}
	if !c.Has(1) {
		t.Fatal("Has(1) should be true after one advance")
	}
	if c.Has(2) {
		t.Fatal("Has(2) should be false after one advance")
	}
}
