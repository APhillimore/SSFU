package wsserver

import (
	"slices"
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type Connection interface {
	ID() string
}

type WsConnection struct {
	id   string
	conn *websocket.Conn
}

func (c *WsConnection) ID() string {
	return c.id
}

func NewWsConnection(conn *websocket.Conn) *WsConnection {
	return &WsConnection{
		id:   uuid.New().String(),
		conn: conn,
	}
}

func (c *WsConnection) Conn() *websocket.Conn {
	return c.conn
}

func (c *WsConnection) Close() error {
	return c.conn.Close()
}

type connectionList[T Connection] struct {
	connections []T
	mu          sync.RWMutex
}

func (hl *connectionList[T]) Add(connection T) T {
	hl.mu.Lock()
	defer hl.mu.Unlock()
	hl.connections = append(hl.connections, connection)
	return connection
}

func (hl *connectionList[T]) Remove(connection T) bool {
	hl.mu.Lock()
	defer hl.mu.Unlock()
	originalLen := len(hl.connections)
	hl.connections = slices.DeleteFunc(hl.connections, func(c T) bool {
		return c.ID() == connection.ID()
	})
	return len(hl.connections) < originalLen
}

func (hl *connectionList[T]) Count() int {
	hl.mu.RLock()
	defer hl.mu.RUnlock()
	return len(hl.connections)
}
