package daemon

import (
	"crypto/rand"
	"encoding/hex"
	"os"
	"sync"
	"time"

	"github.com/ramayac/goposix/pkg/common"
)

type Session struct {
	ID         string            `json:"sessionId"`
	CWD        string            `json:"cwd"`
	BaseDir    string            `json:"baseDir"`
	Env        map[string]string `json:"env"`
	LastActive time.Time         `json:"lastActive"`
}

// copy returns a deep copy of the session. Must be called while holding sm.mu.
func (s *Session) copy() *Session {
	c := &Session{
		ID:         s.ID,
		CWD:        s.CWD,
		BaseDir:    s.BaseDir,
		LastActive: s.LastActive,
	}
	if s.Env != nil {
		c.Env = make(map[string]string, len(s.Env))
		for k, v := range s.Env {
			c.Env[k] = v
		}
	}
	return c
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
		BaseDir:    "/",
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
	if !ok {
		return nil, false
	}
	s.LastActive = time.Now()
	return s.copy(), true
}

func (sm *SessionManager) SetCwd(id string, path string) bool {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	s, ok := sm.sessions[id]
	if !ok {
		return false
	}

	base := "/"
	if s.BaseDir != "" {
		base = s.BaseDir
	}

	securePath, err := common.SecurePath(path, base)
	if err != nil {
		return false
	}

	fi, err := os.Stat(securePath)
	if err != nil || !fi.IsDir() {
		return false
	}

	s.CWD = securePath
	s.LastActive = time.Now()

	if s.BaseDir == "/" && securePath != "/" {
		s.BaseDir = securePath
	}
	return true
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

	list := make([]*Session, 0, len(sm.sessions))
	for _, s := range sm.sessions {
		list = append(list, s.copy())
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
	select {
	case <-sm.done:
		// already stopped
	default:
		close(sm.done)
	}
}
