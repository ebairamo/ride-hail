package wsm

import (
	"fmt"
	"github.com/gorilla/websocket"
	"sync"
)

type WSManager struct {
	conns map[string]*websocket.Conn
	mu    sync.Mutex
}

func NewWSManager() *WSManager {
	return &WSManager{
		conns: make(map[string]*websocket.Conn),
	}
}

type HandlerWS interface {
	AddConn(id string, conn *websocket.Conn)
	Send(id string, msg []byte) error
	RemoveConn(rideID string)
}

type ServiceWS interface {
	Broadcast(msg []byte)
	RemoveConn(id string)
}

func (m *WSManager) AddConn(id string, conn *websocket.Conn) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.conns[id] = conn
}

func (m *WSManager) RemoveConn(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if conn, ok := m.conns[id]; ok {
		conn.Close()
		delete(m.conns, id)
	}
}

func (m *WSManager) Send(id string, msg []byte) error {
	m.mu.Lock()
	conn, ok := m.conns[id]
	m.mu.Unlock()

	if !ok {
		return fmt.Errorf("no websocket for id %s", id)
	}

	return conn.WriteMessage(websocket.TextMessage, msg)
}

func (m *WSManager) Broadcast(msg []byte) []string {
	m.mu.Lock()
	defer m.mu.Unlock()

	errMsg := make([]string, 0)
	for id, conn := range m.conns {
		if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			errMsg = append(errMsg, fmt.Errorf("failed to send to %s: %w", id, err).Error())
			conn.Close()
			delete(m.conns, id)
		}
	}
	return errMsg
}
