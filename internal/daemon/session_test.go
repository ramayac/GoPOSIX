package daemon

import (
	"sync"
	"testing"
	"time"
)

func TestSessionManagerCreateGet(t *testing.T) {
	sm := NewSessionManager(5 * time.Minute)
	defer sm.Stop()

	s := sm.Create()
	if s.ID == "" {
		t.Fatal("expected non-empty session ID")
	}
	if s.CWD != "/" {
		t.Fatalf("expected CWD '/', got %q", s.CWD)
	}

	got, ok := sm.Get(s.ID)
	if !ok {
		t.Fatal("Get returned false for existing session")
	}
	if got.CWD != "/" {
		t.Fatalf("expected CWD '/', got %q", got.CWD)
	}
}

func TestSessionManagerSetCwd(t *testing.T) {
	sm := NewSessionManager(5 * time.Minute)
	defer sm.Stop()

	s := sm.Create()
	sm.SetCwd(s.ID, "/tmp")

	got, ok := sm.Get(s.ID)
	if !ok {
		t.Fatal("Get returned false")
	}
	if got.CWD != "/tmp" {
		t.Fatalf("expected CWD '/tmp', got %q", got.CWD)
	}
}

func TestSessionManagerDestroy(t *testing.T) {
	sm := NewSessionManager(5 * time.Minute)
	defer sm.Stop()

	s := sm.Create()
	if !sm.Destroy(s.ID) {
		t.Fatal("Destroy returned false")
	}
	if _, ok := sm.Get(s.ID); ok {
		t.Fatal("Get returned true for destroyed session")
	}
}

func TestSessionManagerList(t *testing.T) {
	sm := NewSessionManager(5 * time.Minute)
	defer sm.Stop()

	sm.Create()
	sm.Create()

	list := sm.List()
	if len(list) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(list))
	}
}

// TestSessionManagerConcurrentRace proves H3: concurrent Get + SetCwd is a data race.
// Run with: go test -race -run TestSessionManagerConcurrentRace -count=5
func TestSessionManagerConcurrentRace(t *testing.T) {
	sm := NewSessionManager(5 * time.Minute)
	defer sm.Stop()

	s := sm.Create()
	sm.SetCwd(s.ID, "/start")

	var wg sync.WaitGroup
	const iterations = 5000

	// Goroutine A: continuously reads CWD and Env via Get()
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			sess, ok := sm.Get(s.ID)
			if !ok {
				t.Error("Get returned false during concurrent read")
				return
			}
			_ = sess.CWD // concurrent read of CWD
			if sess.Env != nil {
				_ = sess.Env["key"] // concurrent read of map
			}
		}
	}()

	// Goroutine B: continuously mutates CWD and Env via SetCwd/SetEnv equivalent
	// (SetEnv doesn't exist, so we mutate Env directly as SetCwd does)
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			// Simulate mutation that races with Get's returned pointer
			sm.mu.Lock()
			if sess, ok := sm.sessions[s.ID]; ok {
				sess.CWD = "/mutated"
				sess.Env["key"] = "value"
			}
			sm.mu.Unlock()
		}
	}()

	// Concurrent SetCwd from a third goroutine (more realistic trigger)
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			sm.SetCwd(s.ID, "/changed")
		}
	}()

	wg.Wait()
}
