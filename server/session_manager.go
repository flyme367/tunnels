package server

import (
	"sync"
)

type SessionManager struct {
	//deviceID =>*Session
	sessions map[string]*Session
	mu       sync.RWMutex
	// look     int32
}

func NewManager() *SessionManager {
	return &SessionManager{
		sessions: make(map[string]*Session),
		// look:     1,
	}
}

func (m *SessionManager) GetOrCreate(deviceID string) (s *Session) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if _, ok := m.sessions[deviceID]; !ok {
		NewSession(deviceID)
	} else {
		s = m.sessions[deviceID]
	}
	return
}
