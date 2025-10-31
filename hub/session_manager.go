package hub

import (
	"fmt"
	"log/slog"
	"sync"
	pl "tunnels/protocol"
)

type SessionManager struct {
	//deviceID =>*Session
	sessions map[string]*Session
	//
	mu sync.Mutex
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
		fmt.Println("create session")
		// m.mu.Lock()
		m.sessions[deviceID] = NewSession(deviceID)
		// m.mu.Unlock()
	}
	return m.sessions[deviceID]
}

func (m *SessionManager) Remove(deviceID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	fmt.Println("Remove 函数触发", deviceID)
	if s, ok := m.sessions[deviceID]; ok {
		slog.Info("Removing session", "DeviceID", deviceID)
		// fmt.Printf("Removing session %s\n", deviceID)
		s.handleDisconnect(pl.STATUS_CLOSED)
		delete(m.sessions, deviceID)
		fmt.Println("删除成功")
	}
}
