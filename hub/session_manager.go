package hub

import (
	"sync"
)

type SessionManager struct {
	//deviceID =>*Session
	sessions map[string]*Session
	mu       sync.Mutex
	// mu spinLock
}

func NewManager() *SessionManager {
	return &SessionManager{
		sessions: make(map[string]*Session),
		// mu:       spinLock{
		// 	// state: 1,
		// },
	}
}

func (m *SessionManager) GetOrCreate(deviceID string) *Session {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.sessions[deviceID]; !ok {
		// m.mu.Lock()
		m.sessions[deviceID] = NewSession(deviceID)
		// m.mu.Unlock()
	}
	return m.sessions[deviceID]
}

// func (m *SessionManager) Remove(deviceID string) {
// 	m.mu.Lock()
// 	defer m.mu.Unlock()
// 	if s, ok := m.sessions[deviceID]; ok {
// 		fmt.Printf("Removing session %s\n", deviceID)
// 		s.handleDisconnect(STATUS_CLOSED)
// 		delete(m.sessions, deviceID)
// 	}
// }
