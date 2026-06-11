package service

import (
	"lorem-backend/internal/config"
	"lorem-backend/internal/utils"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			return true // Allow non-browser clients (testing tools like curl, Postman)
		}
		// Validate against configured FrontendURL
		return origin == config.GlobalConfig.FrontendURL
	},
}

func (h *Hub) WebsocketHandler(c echo.Context) error {
	// 1. Authenticate via authToken cookie
	cookie, err := c.Cookie("authToken")
	if err != nil {
		return c.JSON(http.StatusUnauthorized, utils.CreateErrorResponse(http.StatusUnauthorized, "Authentication token missing"))
	}

	claims, err := utils.VerifyJWT(cookie.Value, config.GlobalConfig.JWTSecret)
	if err != nil {
		return c.JSON(http.StatusForbidden, utils.CreateErrorResponse(http.StatusForbidden, "Invalid authentication token"))
	}

	userIDStr, ok := claims["id"].(string)
	if !ok {
		return c.JSON(http.StatusUnauthorized, utils.CreateErrorResponse(http.StatusUnauthorized, "Invalid token claims"))
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, utils.CreateErrorResponse(http.StatusUnauthorized, "Invalid user ID in token"))
	}

	// 2. Upgrade to WebSocket
	ws, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return err
	}

	// 3. Register client and start pumps
	client := &Client{
		hub:    h,
		userID: userID,
		conn:   ws,
		send:   make(chan WSPayload, 256),
	}

	h.register <- client

	go client.writePump()
	client.readPump() // Blocks until the connection is closed

	return nil
}
