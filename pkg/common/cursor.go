// Package common provides shared helpers for goposix utilities.
package common

// TokenCursor walks a slice of string tokens. Several expression parsers in
// the tree (expr, testcmd) share this cursor pattern.
type TokenCursor struct {
	tokens []string
	pos    int
}

// NewTokenCursor returns a cursor over the given tokens.
func NewTokenCursor(tokens []string) *TokenCursor {
	return &TokenCursor{tokens: tokens, pos: 0}
}

// Peek returns the current token, or "" when the cursor is exhausted.
func (c *TokenCursor) Peek() string {
	if c.pos >= len(c.tokens) {
		return ""
	}
	return c.tokens[c.pos]
}

// Next returns the current token and advances the cursor. It returns ""
// when the cursor is exhausted.
func (c *TokenCursor) Next() string {
	tok := c.Peek()
	c.pos++
	return tok
}

// Done reports whether the cursor is exhausted.
func (c *TokenCursor) Done() bool {
	return c.pos >= len(c.tokens)
}

// Lookahead returns the token n positions ahead of the cursor, or "" when
// that position is out of range. Lookahead(0) is equivalent to Peek.
func (c *TokenCursor) Lookahead(n int) string {
	i := c.pos + n
	if i < 0 || i >= len(c.tokens) {
		return ""
	}
	return c.tokens[i]
}

// Has reports whether a token exists n positions ahead of the cursor.
func (c *TokenCursor) Has(n int) bool {
	i := c.pos + n
	return i >= 0 && i < len(c.tokens)
}
