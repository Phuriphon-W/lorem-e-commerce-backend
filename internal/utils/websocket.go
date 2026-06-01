package utils

import (
	"log"
	"net/http"
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for simplicity
	},
}

type WSPayload struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

type WSHub struct {
	sync.RWMutex
	clients map[uuid.UUID][]*websocket.Conn
}

var DefaultWSHub = &WSHub{
	clients: make(map[uuid.UUID][]*websocket.Conn),
}

func (h *WSHub) AddClient(userID uuid.UUID, conn *websocket.Conn) {
	h.Lock()
	defer h.Unlock()
	h.clients[userID] = append(h.clients[userID], conn)
}

func (h *WSHub) RemoveClient(userID uuid.UUID, conn *websocket.Conn) {
	h.Lock()
	defer h.Unlock()
	conns := h.clients[userID]
	for i, c := range conns {
		if c == conn {
			h.clients[userID] = append(conns[:i], conns[i+1:]...)
			break
		}
	}
	if len(h.clients[userID]) == 0 {
		delete(h.clients, userID)
	}
}

func (h *WSHub) SendToUser(userID uuid.UUID, message interface{}) {
	h.RLock()
	defer h.RUnlock()
	conns := h.clients[userID]
	for _, c := range conns {
		err := c.WriteJSON(message)
		if err != nil {
			log.Printf("WS error: %v", err)
			c.Close()
		}
	}
}

func WebsocketHandler(c echo.Context) error {
	userIDStr := c.QueryParam("userId")
	if userIDStr == "" {
		return c.String(http.StatusBadRequest, "userId is required")
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.String(http.StatusBadRequest, "invalid userId")
	}

	ws, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return err
	}

	DefaultWSHub.AddClient(userID, ws)
	defer DefaultWSHub.RemoveClient(userID, ws)
	defer ws.Close()

	for {
		_, _, err := ws.ReadMessage()
		if err != nil {
			break
		}
	}

	return nil
}
