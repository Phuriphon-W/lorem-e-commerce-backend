package handler

import (
	"context"
	"fmt"
	"lorem-backend/internal/config"
	"lorem-backend/internal/database"
	"lorem-backend/internal/modules/auth/dto"
	"lorem-backend/internal/modules/auth/repository"
	"lorem-backend/internal/utils"
	"net/http"
	"time"
)

type authHandlerImpl struct {
	authRepository repository.AuthRepository
	jwtSecret      string
	jwtExpire      string
}

func NewAuthHandlerImpl(authRepo repository.AuthRepository, cfg *config.Config) AuthHandler {
	return &authHandlerImpl{
		authRepository: authRepo,
		jwtSecret:      cfg.JWTSecret,
		jwtExpire:      cfg.JWTExpire,
	}
}

func (a *authHandlerImpl) RegisterUser(ctx context.Context, input *dto.RegisterUserInputDto) (*dto.RegisterUserOutputDto, error) {
	hashed, err := utils.HashPassword(input.Body.Password)
	if err != nil {
		return nil, fmt.Errorf("Failed to hash password: %v", err)
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
		return nil, fmt.Errorf("Failed to create user: %v", err)
	}

	duration, err := time.ParseDuration(a.jwtExpire)
	if err != nil {
		// Fallback to a default if the string is invalid
		duration = 24 * time.Hour
	}

	token, err := utils.GenerateJWT(userID, a.jwtSecret, duration)
	if err != nil {
		return nil, fmt.Errorf("Failed to generate session token: %v", err)
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
			JwtToken: token,
		},
	}

	return res, nil
}
