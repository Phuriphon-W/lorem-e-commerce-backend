package dto

import (
	"github.com/google/uuid"
)

type UserDto struct {
	ID        uuid.UUID `path:"id" required:"true" doc:"User ID"`
	Username  string    `json:"username" required:"true" maxLength:"20" doc:"Username" example:"user123"`
	FirstName string    `json:"firstName" required:"true" maxLength:"20" doc:"First Name" example:"John"`
	LastName  string    `json:"lastName" required:"true" maxLength:"20" doc:"Last Name" example:"Doe"`
}

// Get user by ID
type (
	GetUserByIdInputDto struct {
		ID uuid.UUID `path:"id" required:"true" doc:"User ID"`
	}

	GetUserByIdOutputDto struct {
		Body UserDto
	}
)
