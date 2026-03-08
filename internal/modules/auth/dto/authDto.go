package dto

import (
	"net/http"

	"github.com/google/uuid"
)

// User register
type (
	RegisterUserInputDto struct {
		Body struct {
			Username  string `json:"username" required:"true" maxLength:"20" doc:"Username" example:"user123"`
			FirstName string `json:"firstName" required:"true" maxLength:"20" doc:"First Name" example:"John"`
			LastName  string `json:"lastName" required:"true" maxLength:"20" doc:"Last Name" example:"Doe"`
			Email     string `json:"email" required:"true" doc:"E-mail Address" example:"example@mail.com"`
			Password  string `json:"password" required:"true" doc:"Password"`
		}
	}

	RegisterUserOutputDtoBody struct {
		ID       uuid.UUID `json:"id" doc:"Created User ID"`
		Username string    `json:"username" required:"true" maxLength:"20" doc:"Username" example:"user123"`
		JwtToken string    `json:"jwtToken" doc:"JWT Access Token for the session"`
	}

	RegisterUserOutputDto struct {
		AuthToken http.Cookie `header:"Set-Cookie" required:"true" doc:"Cookie Session Token"`
		Body      RegisterUserOutputDtoBody
	}
)

// User sign in
type (
	SignInUserInputDto struct {
		Body struct {
			Email    string `json:"email" required:"true" doc:"E-mail Address" example:"example@mail.com"`
			Password string `json:"password" required:"true" doc:"Password"`
		}
	}

	SignInUserOutputDtoBody struct {
		ID       uuid.UUID `json:"id" doc:"Created User ID"`
		Username string    `json:"username" required:"true" maxLength:"20" doc:"Username" example:"user123"`
		JwtToken string    `json:"jwtToken" doc:"JWT Access Token for the session"`
	}

	SignInUserOutputDto struct {
		AuthToken http.Cookie `header:"Set-Cookie" required:"true" doc:"Cookie Session Token"`
		Body      SignInUserOutputDtoBody
	}
)

// User sign out
type (
	SignOutUserInputDto struct{}

	SignOutUserOutputDto struct {
		AuthToken http.Cookie `header:"Set-Cookie" required:"true" doc:"Cleared Cookie Session Token"`
	}
)
