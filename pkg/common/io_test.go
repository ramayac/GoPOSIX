package common

import (
	"bytes"
	"strings"
	"testing"
)

func TestLimitWriter_UnderLimit(t *testing.T) {
	var buf bytes.Buffer
	lw := &LimitWriter{W: &buf, Limit: 10}

	n, err := lw.Write([]byte("hello"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 5 {
		t.Errorf("expected 5 bytes written, got %d", n)
	}
	if buf.String() != "hello" {
		t.Errorf("expected 'hello', got %q", buf.String())
	}
}

func TestLimitWriter_ExactlyLimit(t *testing.T) {
	var buf bytes.Buffer
	lw := &LimitWriter{W: &buf, Limit: 5}

	n, err := lw.Write([]byte("hello"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 5 {
		t.Errorf("expected 5 bytes written, got %d", n)
	}
}

func TestLimitWriter_OverLimitSingleWrite(t *testing.T) {
	var buf bytes.Buffer
	lw := &LimitWriter{W: &buf, Limit: 5}

	n, err := lw.Write([]byte("hello world"))
	if err == nil {
		t.Fatal("expected limit error")
	}
	if !strings.Contains(err.Error(), "output limit exceeded") {
		t.Errorf("expected 'output limit exceeded' error, got: %v", err)
	}
	if n != 5 {
		t.Errorf("expected 5 bytes written, got %d", n)
	}
	if buf.String() != "hello" {
		t.Errorf("expected 'hello', got %q", buf.String())
	}
}

func TestLimitWriter_OverLimitMultipleWrites(t *testing.T) {
	var buf bytes.Buffer
	lw := &LimitWriter{W: &buf, Limit: 5}

	n1, err1 := lw.Write([]byte("abc"))
	if err1 != nil {
		t.Fatalf("n1 error: %v", err1)
	}
	if n1 != 3 {
		t.Errorf("n1 = %d", n1)
	}

	n2, err2 := lw.Write([]byte("def"))
	if err2 == nil {
		t.Fatal("expected error on second write")
	}
	if n2 != 2 {
		t.Errorf("expected 2 bytes on second write, got %d", n2)
	}

	n3, err3 := lw.Write([]byte("ghi"))
	if err3 == nil {
		t.Fatal("expected error on third write")
	}
	if n3 != 0 {
		t.Errorf("expected 0 bytes on third write, got %d", n3)
	}
}
