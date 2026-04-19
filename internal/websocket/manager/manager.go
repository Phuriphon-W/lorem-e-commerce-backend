package manager

import (
	"github.com/google/uuid"
)

type WsManager interface {
	Register(client *Client)
	Unregister(client *Client)
	SendToUser(userID uuid.UUID, message []byte)
}
