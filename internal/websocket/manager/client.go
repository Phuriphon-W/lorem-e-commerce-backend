package manager

import (
	"log"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type Client struct {
	ID      uuid.UUID
	UserID  uuid.UUID
	Conn    *websocket.Conn
	Manager WsManager
	Send    chan []byte // Buffered channel of outbound messages
}

// ReadPump listens for incoming messages from the frontend
func (c *Client) ReadPump() {
	defer func() {
		c.Manager.Unregister(c)
		c.Conn.Close()
	}()

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("Unexpected Close Error: %v\n", err)
			}

			break
		}

		// TODO: Add logic to handle message from frontend. Just print for now.
		log.Printf("Received message from %s: %s", c.UserID, string(message))
	}
}

// WritePump pushes messages from the backend to the frontend
func (c *Client) WritePump() {
	defer c.Conn.Close()

	for message := range c.Send {
		err := c.Conn.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			return // Connection dropped
		}
	}
}
