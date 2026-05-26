package handler

import (
	"context"
	"log"
	"lorem-backend/internal/config"
	"lorem-backend/internal/database"
	"lorem-backend/internal/modules/auth/dto"
	"lorem-backend/internal/modules/auth/repository"
	"lorem-backend/internal/utils"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
)

type authHandlerImpl struct {
	authRepository repository.AuthRepository
	emailService   utils.EmailService
	jwtSecret      string
	jwtExpire      string
}

func NewAuthHandlerImpl(authRepo repository.AuthRepository, emailSvc utils.EmailService) AuthHandler {
	return &authHandlerImpl{
		authRepository: authRepo,
		emailService:   emailSvc,
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

func (a *authHandlerImpl) ForgotPassword(ctx context.Context, input *dto.ForgotPasswordInputDto) (*dto.ForgotPasswordOutputDto, error) {
	// Define the generic success response we will return no matter what happens
	successResponse := &dto.ForgotPasswordOutputDto{
		Body: dto.ForgotPasswordOutputDtoBody{
			Message: "If your email is registered, you will receive a password reset link shortly.",
		},
	}

	// Check if user exists
	userData, err := a.authRepository.GetUserByEmail(ctx, input.Body.Email)
	if err != nil || userData == nil {
		// Do not leak that the user does not exist. Just return generic success message.
		return successResponse, nil
	}

	// Generate reset token and send e-mail in the background
	go func(userID uuid.UUID, username string, userEmail string) {
		// Generate reset token (JWT) valid for 10 minutes
		resetDuration := 10 * time.Minute
		token, err := utils.GenerateJWT(userData.ID, a.jwtSecret, resetDuration)
		if err != nil {
			log.Printf("Failed to generate reset token: %v\n", err)
			return // Stop execution for this goroutine
		}

		// Send the reset link via email
		resetLink := config.GlobalConfig.FrontendURL + "/reset-password?token=" + token
		err = a.emailService.SendResetPasswordEmail(userEmail, username, resetLink)
		if err != nil {
			log.Printf("Failed to send password reset email to %v: %v\n", userEmail, err)
		}
		log.Printf("Password Reset Email Successfully Send to %v\n", userEmail)
	}(userData.ID, userData.Username, input.Body.Email)

	return successResponse, nil
}

func (a *authHandlerImpl) ResetPassword(ctx context.Context, input *dto.ResetPasswordInputDto) (*dto.ResetPasswordOutputDto, error) {
	// Verify token
	claims, err := utils.VerifyJWT(input.Body.Token, a.jwtSecret)
	if err != nil {
		return nil, huma.Error401Unauthorized("Invalid or expired reset token.")
	}

	userIDStr, ok := claims["id"].(string)
	if !ok {
		return nil, huma.Error401Unauthorized("Invalid reset token payload.")
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, huma.Error401Unauthorized("Invalid reset token user format.")
	}

	// Hash new password
	hashed, err := utils.HashPassword(input.Body.Password)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to hash new password", err)
	}

	// Update password in DB
	err = a.authRepository.UpdatePassword(ctx, userID, hashed)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to update password", err)
	}

	return &dto.ResetPasswordOutputDto{
		Body: dto.ResetPasswordOutputDtoBody{
			Message: "Password has been successfully updated.",
		},
	}, nil
}
