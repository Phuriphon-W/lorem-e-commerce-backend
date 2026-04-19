package websocket

import (
	"lorem-backend/internal/websocket/manager"

	"github.com/labstack/echo/v4"
)

type Socket interface {
	HandleConnection(c echo.Context) error
	GetWsManagerInstance() manager.WsManager
}
