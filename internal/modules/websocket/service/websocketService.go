package service

import (
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
)

type WSPayload struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

type WebsocketService interface {
	AddClient(userID uuid.UUID, conn *websocket.Conn)
	RemoveClient(userID uuid.UUID, conn *websocket.Conn)
	SendToUser(userID uuid.UUID, message WSPayload)
	WebsocketHandler(c echo.Context) error
}
