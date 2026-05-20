package dispatch

import (
	"io"
	"testing"
)

func TestRegisterAndLookup(t *testing.T) {
	// Use a fresh blank registry by temporarily swapping.
	old := registry
	registry = map[string]Command{}
	defer func() { registry = old }()

	Register(Command{Name: "test-cmd", Usage: "a test", Run: func([]string, io.Reader, io.Writer) int { return 0 }})
	cmd, ok := Lookup("test-cmd")
	if !ok {
		t.Fatal("expected to find test-cmd")
	}
	if cmd.Name != "test-cmd" {
		t.Errorf("name: got %q, want test-cmd", cmd.Name)
	}
}

func TestLookupMissing(t *testing.T) {
	_, ok := Lookup("does-not-exist-xyz")
	if ok {
		t.Error("expected Lookup to return false for unknown command")
	}
}

func TestDuplicateRegistrationPanics(t *testing.T) {
	old := registry
	registry = map[string]Command{}
	defer func() { registry = old }()

	Register(Command{Name: "dup", Run: func([]string, io.Reader, io.Writer) int { return 0 }})
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on duplicate registration")
		}
	}()
	Register(Command{Name: "dup", Run: func([]string, io.Reader, io.Writer) int { return 0 }})
}

func TestListAllSorted(t *testing.T) {
	old := registry
	registry = map[string]Command{}
	defer func() { registry = old }()

	for _, name := range []string{"z", "a", "m"} {
		n := name
		Register(Command{Name: n, Run: func([]string, io.Reader, io.Writer) int { return 0 }})
	}
	all := ListAll()
	if len(all) != 3 {
		t.Fatalf("expected 3 commands, got %d", len(all))
	}
	if all[0].Name != "a" || all[1].Name != "m" || all[2].Name != "z" {
		t.Errorf("not sorted: %v", all)
	}
}

func TestListAllEmpty(t *testing.T) {
	old := registry
	registry = map[string]Command{}
	defer func() { registry = old }()

	all := ListAll()
	if len(all) != 0 {
		t.Errorf("expected 0 commands, got %d", len(all))
	}
}
