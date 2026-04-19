package websocket

import (
	"lorem-backend/internal/config"
	"lorem-backend/internal/utils"
	"lorem-backend/internal/websocket/manager"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
)

type gorillaWs struct {
	manager  manager.WsManager
	upgrader websocket.Upgrader
}

func NewGorillaWs(mgr manager.WsManager) Socket {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")

			// Only allow connections from frontend url
			return origin == config.GlobalConfig.FrontendURL
		},
	}

	return &gorillaWs{
		manager:  mgr,
		upgrader: upgrader,
	}
}

func (g *gorillaWs) HandleConnection(c echo.Context) error {
	// Get UserID from context
	val := c.Get("userID")
	userIDStr, ok := val.(string)
	if !ok {
		return c.JSON(http.StatusUnauthorized, utils.CreateErrorResponse(http.StatusUnauthorized, "Missing User ID"))
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, utils.CreateErrorResponse(http.StatusBadRequest, "Failed to parse User ID"))
	}

	// Upgrade the standard HTTP request to a WebSocket
	conn, err := g.upgrader.Upgrade(c.Response().Writer, c.Request(), nil)
	if err != nil {
		// upgrader handles the error response automatically
		return err
	}

	client := &manager.Client{
		ID:      uuid.New(),
		UserID:  userID,
		Conn:    conn,
		Manager: g.manager,
		Send:    make(chan []byte, 256),
	}
	g.manager.Register(client)

	go client.WritePump()
	go client.ReadPump()

	return nil
}

func (g *gorillaWs) GetWsManagerInstance() manager.WsManager {
	return g.manager
}
