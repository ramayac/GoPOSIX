package daemon

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

type Session struct {
	ID         string            `json:"sessionId"`
	CWD        string            `json:"cwd"`
	Env        map[string]string `json:"env"`
	LastActive time.Time         `json:"lastActive"`
}

type SessionManager struct {
	mu                   sync.Mutex
	sessions             map[string]*Session
	ttl                  time.Duration
	totalSessionsCreated int64
	done                 chan struct{}
}

func NewSessionManager(ttl time.Duration) *SessionManager {
	sm := &SessionManager{
		sessions: make(map[string]*Session),
		ttl:      ttl,
		done:     make(chan struct{}),
	}
	go sm.cleanupLoop()
	return sm
}

func (sm *SessionManager) Create() *Session {
	b := make([]byte, 8)
	rand.Read(b)
	id := hex.EncodeToString(b)

	sm.mu.Lock()
	defer sm.mu.Unlock()

	s := &Session{
		ID:         id,
		CWD:        "/",
		Env:        make(map[string]string),
		LastActive: time.Now(),
	}
	sm.sessions[id] = s
	sm.totalSessionsCreated++
	return s
}

func (sm *SessionManager) Get(id string) (*Session, bool) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	s, ok := sm.sessions[id]
	if ok {
		s.LastActive = time.Now()
	}
	return s, ok
}

func (sm *SessionManager) SetCwd(id string, path string) bool {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	s, ok := sm.sessions[id]
	if ok {
		s.CWD = path
		s.LastActive = time.Now()
	}
	return ok
}

func (sm *SessionManager) Destroy(id string) bool {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	_, ok := sm.sessions[id]
	if ok {
		delete(sm.sessions, id)
	}
	return ok
}

// TotalCreated returns the total number of sessions ever created.
func (sm *SessionManager) TotalCreated() int64 {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	return sm.totalSessionsCreated
}

func (sm *SessionManager) List() []*Session {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	var list []*Session
	for _, s := range sm.sessions {
		list = append(list, s)
	}
	return list
}

func (sm *SessionManager) cleanupLoop() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			sm.mu.Lock()
			now := time.Now()
			for id, s := range sm.sessions {
				if now.Sub(s.LastActive) > sm.ttl {
					delete(sm.sessions, id)
				}
			}
			sm.mu.Unlock()
		case <-sm.done:
			return
		}
	}
}

func (sm *SessionManager) Stop() {
	close(sm.done)
}
