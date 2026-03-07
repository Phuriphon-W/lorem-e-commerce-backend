package dto

import (
	"github.com/google/uuid"
)

type (
	// User registers
	CreateUserInputDto struct {
		Body struct {
			Username  string `json:"username" required:"true" maxLength:"20" doc:"Username" example:"user123"`
			FirstName string `json:"firstName" required:"true" maxLength:"20" doc:"First Name" example:"John"`
			LastName  string `json:"lastName" required:"true" maxLength:"20" doc:"Last Name" example:"Doe"`
			Email     string `json:"email" required:"true" doc:"E-mail Address" example:"example@mail.com"`
			Password  string `json:"password" required:"true" doc:"Password"`
		}
	}

	CreateUserOutputDtoBody struct {
		ID uuid.UUID `json:"id" doc:"Created User ID"`
	}

	CreateUserOutputDto struct {
		Body CreateUserOutputDtoBody
	}
)
