package handler

import (
	"context"
	"lorem-backend/internal/config"
	"lorem-backend/internal/database"
	"lorem-backend/internal/modules/auth/dto"
	"lorem-backend/internal/modules/auth/repository"
	"lorem-backend/internal/utils"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
)

type authHandlerImpl struct {
	authRepository repository.AuthRepository
	jwtSecret      string
	jwtExpire      string
}

func NewAuthHandlerImpl(authRepo repository.AuthRepository) AuthHandler {
	return &authHandlerImpl{
		authRepository: authRepo,
		jwtSecret:      config.GlobalConfig.JWTSecret,
		jwtExpire:      config.GlobalConfig.JWTExpire,
	}
}

func (a *authHandlerImpl) RegisterUser(ctx context.Context, input *dto.RegisterUserInputDto) (*dto.RegisterUserOutputDto, error) {
	// Check if user with this email already exist
	userData, err := a.authRepository.GetUserByEmail(ctx, input.Body.Email)
	if userData != nil {
		return nil, huma.Error409Conflict("An account with this email address already exists.")
	}

	// Check if user with this username already exist
	usernameData, err := a.authRepository.GetUserByUsername(ctx, input.Body.Username)
	if usernameData != nil {
		return nil, huma.Error409Conflict("An account with this username already exist.")
	}

	hashed, err := utils.HashPassword(input.Body.Password)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to hash password", err)
	}

	data := &database.User{
		Username:     input.Body.Username,
		FirstName:    input.Body.FirstName,
		LastName:     input.Body.LastName,
		Email:        input.Body.Email,
		PasswordHash: hashed,
	}

	userID, username, err := a.authRepository.RegisterUser(ctx, data)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to create user", err)
	}

	duration, err := time.ParseDuration(a.jwtExpire)
	if err != nil {
		// Fallback to a default if the string is invalid
		duration = 24 * time.Hour
	}

	token, err := utils.GenerateJWT(userID, a.jwtSecret, duration)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to generate session token", err)
	}

	res := &dto.RegisterUserOutputDto{
		AuthToken: http.Cookie{
			Name:     "authToken",
			Value:    token,
			Path:     "/",
			HttpOnly: true,
			MaxAge:   int(duration.Seconds()),
		},
		Body: dto.RegisterUserOutputDtoBody{
			ID:       userID,
			Username: username,
		},
	}

	return res, nil
}

func (a *authHandlerImpl) SignInUser(ctx context.Context, input *dto.SignInUserInputDto) (*dto.SignInUserOutputDto, error) {
	// Get userID, username, and hashed password
	data, err := a.authRepository.GetUserByEmail(ctx, input.Body.Email)
	if err != nil {
		return nil, huma.Error404NotFound("Wrong E-mail or Password")
	}

	// Verify password
	isMatched := utils.VerifyPassword(input.Body.Password, data.PasswordHash)

	if !isMatched {
		return nil, huma.Error404NotFound("Wrong E-mail or Password")
	}

	// Create token valid duration
	duration, err := time.ParseDuration(a.jwtExpire)
	if err != nil {
		// Fallback to a default if the string is invalid
		duration = 24 * time.Hour
	}

	// Generate JWT
	token, err := utils.GenerateJWT(data.ID, a.jwtSecret, duration)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to generate session token", err)
	}

	res := &dto.SignInUserOutputDto{
		AuthToken: http.Cookie{
			Name:     "authToken",
			Value:    token,
			Path:     "/",
			HttpOnly: true,
			MaxAge:   int(duration.Seconds()),
			SameSite: http.SameSiteNoneMode,
			Secure:   true,
		},
		Body: dto.SignInUserOutputDtoBody{
			ID:       data.ID,
			Username: data.Username,
		},
	}

	return res, nil
}

func (a *authHandlerImpl) SignOutUser(ctx context.Context, input *dto.SignOutUserInputDto) (*dto.SignOutUserOutputDto, error) {
	cookie := http.Cookie{
		Name:     "authToken",
		Value:    "", // Clear the value
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1, // -1 means "delete immediately"
		SameSite: http.SameSiteNoneMode,
		Secure:   true,
	}

	res := &dto.SignOutUserOutputDto{
		AuthToken: cookie,
	}

	return res, nil
}
