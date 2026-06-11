package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type WSPayload struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

type WebsocketService interface {
	SendToUser(userID uuid.UUID, message WSPayload)
	WebsocketHandler(c echo.Context) error
	Run(ctx context.Context)
}
