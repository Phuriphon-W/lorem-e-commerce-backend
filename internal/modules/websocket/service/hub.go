package service

import (
	"context"

	"github.com/google/uuid"
)

type userMessage struct {
	UserID  uuid.UUID
	Payload WSPayload
}

type Hub struct {
	clients    map[uuid.UUID]map[*Client]struct{}
	register   chan *Client
	unregister chan *Client
	send       chan userMessage
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[uuid.UUID]map[*Client]struct{}),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		send:       make(chan userMessage, 256),
	}
}

func NewWebsocketService() WebsocketService {
	return NewHub()
}

func (h *Hub) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			// Graceful shutdown: clean up all connections
			for _, clients := range h.clients {
				for client := range clients {
					close(client.send)
					client.conn.Close()
				}
			}
			return
		case client := <-h.register:
			if h.clients[client.userID] == nil {
				h.clients[client.userID] = make(map[*Client]struct{})
			}
			h.clients[client.userID][client] = struct{}{}
		case client := <-h.unregister:
			if clients, ok := h.clients[client.userID]; ok {
				if _, ok := clients[client]; ok {
					delete(clients, client)
					close(client.send)
					if len(clients) == 0 {
						delete(h.clients, client.userID)
					}
				}
			}
		case msg := <-h.send:
			if clients, ok := h.clients[msg.UserID]; ok {
				for client := range clients {
					select {
					case client.send <- msg.Payload:
					default:
						// Buffer full, drop the message and close connection
						close(client.send)
						delete(clients, client)
						client.conn.Close()
					}
				}
				if len(clients) == 0 {
					delete(h.clients, msg.UserID)
				}
			}
		}
	}
}

func (h *Hub) SendToUser(userID uuid.UUID, message WSPayload) {
	select {
	case h.send <- userMessage{UserID: userID, Payload: message}:
	default:
		// Drop message if hub send channel is full or no goroutine is reading (e.g. hub stopped)
	}
}
