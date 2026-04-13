package dto

import (
	"net/http"

	"github.com/google/uuid"
)

type UserAddress struct {
	ZipCode     string `json:"zip" doc:"Zip code"`
	Road        string `json:"road" doc:"Road"`
	District    string `json:"district" doc:"District"`
	SubDistrict string `json:"subDistrict" doc:"Sub District"`
	HouseNumber string `json:"houseNumber" doc:"House Number"`
	Province    string `json:"province" doc:"Province"`
}

type UserDto struct {
	ID        uuid.UUID   `json:"id" required:"true" doc:"User ID"`
	Username  string      `json:"username" required:"true" maxLength:"20" doc:"Username" example:"user123"`
	FirstName string      `json:"firstName" required:"true" maxLength:"20" doc:"First Name" example:"John"`
	LastName  string      `json:"lastName" required:"true" maxLength:"20" doc:"Last Name" example:"Doe"`
	Email     string      `json:"email" required:"true" doc:"Email"`
	Telephone string      `json:"telephone" doc:"Phone number"`
	Address   UserAddress `json:"address" doc:"User address details"`
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

// Get current logged in user from authToken
type (
	GetMeInputDto struct {
		AuthToken http.Cookie `cookie:"authToken"`
	}

	GetMeOutputDto struct {
		Body UserDto
	}
)

// Update User
type (
	UpdateMeInputDto struct {
		Body struct {
			FirstName string      `json:"firstName"`
			LastName  string      `json:"lastName"`
			Telephone string      `json:"telephone"`
			Address   UserAddress `json:"address"`
		}
	}

	UpdateMeOutputDtoBody struct {
		Message string `json:"message"`
	}

	UpdateMeOutputDto struct {
		Body UpdateMeOutputDtoBody
	}
)
