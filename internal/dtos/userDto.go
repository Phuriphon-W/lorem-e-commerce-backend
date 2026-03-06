package dtos

import (
	"github.com/google/uuid"
)

type (
	// User registers
	CreateUserRequestDto struct {
		Body struct {
			Username  string `json:"username" required:"true" maxLength:"20" doc:"Username"`
			FirstName string `json:"firstName" `
			LastName  string `json:"lastName" `
			Email     string `json:"email" `
			Password  string `json:"password"`
		}
	}

	CreateUserResponseDtoBody struct {
		ID uuid.UUID `json:"id" doc:"Created User ID"`
	}

	CreateUserResponseDto struct {
		Body CreateUserResponseDtoBody
	}
)
