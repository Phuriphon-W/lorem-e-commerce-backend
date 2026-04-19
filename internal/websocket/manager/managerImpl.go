package manager

import (
	"fmt"
	"sync"

	"github.com/google/uuid"
)

type wsManagerImpl struct {
	clients map[uuid.UUID]map[*Client]bool
	mu      sync.RWMutex
}

func NewWsManagerImpl() WsManager {
	return &wsManagerImpl{
		clients: make(map[uuid.UUID]map[*Client]bool),
	}
}

func (m *wsManagerImpl) Register(client *Client) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.clients[client.UserID] == nil {
		m.clients[client.UserID] = make(map[*Client]bool)
	}
	m.clients[client.UserID][client] = true
	fmt.Printf("User %s connected\n", client.UserID)
}

func (m *wsManagerImpl) Unregister(client *Client) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if connections, ok := m.clients[client.UserID]; ok {
		if _, exists := connections[client]; exists {
			delete(connections, client)
			close(client.Send)

			// Clean up if user has no more active connections
			if len(connections) == 0 {
				delete(m.clients, client.UserID)
			}
			fmt.Printf("User %s disconnected\n", client.UserID)
		}
	}
}

// SendToUser allows any part of the backend to push data to a user
func (m *wsManagerImpl) SendToUser(userID uuid.UUID, message []byte) {
	m.mu.RLock()
	defer m.mu.RLock()

	if connecions, ok := m.clients[userID]; ok {
		for client := range connecions {
			client.Send <- message
		}
	}
}
