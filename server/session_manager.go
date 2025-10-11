package server

type SessionManager struct {
	//deviceID =>*Session
	sessions map[string]*Session
	// mu       sync.RWMutex
	look spinLock
}

func NewManager() *SessionManager {
	return &SessionManager{
		sessions: make(map[string]*Session),
		look:     spinLock{
			// state: 1,
		},
	}
}

func (m *SessionManager) GetOrCreate(deviceID string) (s *Session) {
	m.look.Lock()
	defer m.look.Unlock()
	// m.mu.RLock()
	// defer m.mu.RUnlock()

	if _, ok := m.sessions[deviceID]; !ok {
		s = NewSession(deviceID)
	} else {
		s = m.sessions[deviceID]
	}
	return
}
