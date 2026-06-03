package handler

import (
	"context"
	"errors"
	"testing"
	"time"

	"lorem-backend/internal/config"
	"lorem-backend/internal/database"
	"lorem-backend/internal/modules/auth/dto"
	"lorem-backend/internal/utils"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

// MockAuthRepository is a mock type for AuthRepository
type MockAuthRepository struct {
	mock.Mock
}

func (m *MockAuthRepository) RegisterUser(ctx context.Context, user *database.User) (uuid.UUID, string, error) {
	args := m.Called(ctx, user)
	return args.Get(0).(uuid.UUID), args.String(1), args.Error(2)
}

func (m *MockAuthRepository) GetUserByEmail(ctx context.Context, email string) (*struct {
	ID           uuid.UUID
	Username     string
	PasswordHash string
}, error) {
	args := m.Called(ctx, email)
	if args.Get(0) != nil {
		return args.Get(0).(*struct {
			ID           uuid.UUID
			Username     string
			PasswordHash string
		}), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockAuthRepository) GetUserByUsername(ctx context.Context, username string) (*struct {
	ID       uuid.UUID
	Username string
}, error) {
	args := m.Called(ctx, username)
	if args.Get(0) != nil {
		return args.Get(0).(*struct {
			ID       uuid.UUID
			Username string
		}), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockAuthRepository) UpdatePassword(ctx context.Context, userID uuid.UUID, newPasswordHash string) error {
	args := m.Called(ctx, userID, newPasswordHash)
	return args.Error(0)
}

// MockEmailService is a mock type for EmailService
type MockEmailService struct {
	mock.Mock
}

func (m *MockEmailService) SendResetPasswordEmail(toEmail, userName, resetLink string) error {
	args := m.Called(toEmail, userName, resetLink)
	return args.Error(0)
}

type AuthHandlerTestSuite struct {
	suite.Suite
	mockRepo  *MockAuthRepository
	mockEmail *MockEmailService
	handler   AuthHandler
	ctx       context.Context
}

func (s *AuthHandlerTestSuite) SetupTest() {
	config.GlobalConfig = &config.Config{
		JWTSecret:   "test-secret-at-least-thirty-two-bytes-long",
		JWTExpire:   "24h",
		FrontendURL: "http://localhost:3000",
	}
	s.mockRepo = new(MockAuthRepository)
	s.mockEmail = new(MockEmailService)
	s.handler = NewAuthHandlerImpl(s.mockRepo, s.mockEmail)
	s.ctx = context.Background()
}

func (s *AuthHandlerTestSuite) TestRegisterUser() {
	userID := uuid.New()
	username := "johndoe"
	email := "john@example.com"
	password := "password123"

	input := &dto.RegisterUserInputDto{
		Body: struct {
			Username  string `json:"username" required:"true" maxLength:"20" doc:"Username" example:"user123"`
			FirstName string `json:"firstName" required:"true" maxLength:"20" doc:"First Name" example:"John"`
			LastName  string `json:"lastName" required:"true" maxLength:"20" doc:"Last Name" example:"Doe"`
			Email     string `json:"email" required:"true" doc:"E-mail Address" example:"example@mail.com"`
			Password  string `json:"password" required:"true" doc:"Password"`
		}{
			Username:  username,
			FirstName: "John",
			LastName:  "Doe",
			Email:     email,
			Password:  password,
		},
	}

	testCases := []struct {
		name          string
		setupMock     func()
		jwtExpire     string
		expectedError bool
		verify        func(res *dto.RegisterUserOutputDto)
	}{
		{
			name: "Success - registers new user",
			setupMock: func() {
				s.mockRepo.On("GetUserByEmail", mock.Anything, email).Return(nil, nil).Once()
				s.mockRepo.On("GetUserByUsername", mock.Anything, username).Return(nil, nil).Once()
				s.mockRepo.On("RegisterUser", mock.Anything, mock.MatchedBy(func(u *database.User) bool {
					return u.Username == username && u.Email == email
				})).Return(userID, username, nil).Once()
			},
			jwtExpire:     "1h",
			expectedError: false,
			verify: func(res *dto.RegisterUserOutputDto) {
				s.NotNil(res)
				s.Equal(userID, res.Body.ID)
				s.Equal(username, res.Body.Username)
				s.Equal("authToken", res.AuthToken.Name)
				s.NotEmpty(res.AuthToken.Value)
				s.Equal(3600, res.AuthToken.MaxAge)
			},
		},
		{
			name: "Success - registers new user with invalid duration configuration fallback",
			setupMock: func() {
				s.mockRepo.On("GetUserByEmail", mock.Anything, email).Return(nil, nil).Once()
				s.mockRepo.On("GetUserByUsername", mock.Anything, username).Return(nil, nil).Once()
				s.mockRepo.On("RegisterUser", mock.Anything, mock.MatchedBy(func(u *database.User) bool {
					return u.Username == username && u.Email == email
				})).Return(userID, username, nil).Once()
			},
			jwtExpire:     "invalid-duration",
			expectedError: false,
			verify: func(res *dto.RegisterUserOutputDto) {
				s.NotNil(res)
				s.Equal(userID, res.Body.ID)
				s.Equal("authToken", res.AuthToken.Name)
				s.Equal(86400, res.AuthToken.MaxAge)
			},
		},
		{
			name: "Failure - email already exists",
			setupMock: func() {
				s.mockRepo.On("GetUserByEmail", mock.Anything, email).Return(&struct {
					ID           uuid.UUID
					Username     string
					PasswordHash string
				}{ID: userID, Username: username}, nil).Once()
			},
			jwtExpire:     "24h",
			expectedError: true,
			verify: func(res *dto.RegisterUserOutputDto) {
				s.Nil(res)
			},
		},
		{
			name: "Failure - username already exists",
			setupMock: func() {
				s.mockRepo.On("GetUserByEmail", mock.Anything, email).Return(nil, nil).Once()
				s.mockRepo.On("GetUserByUsername", mock.Anything, username).Return(&struct {
					ID       uuid.UUID
					Username string
				}{ID: userID, Username: username}, nil).Once()
			},
			jwtExpire:     "24h",
			expectedError: true,
			verify: func(res *dto.RegisterUserOutputDto) {
				s.Nil(res)
			},
		},
		{
			name: "Failure - database error on RegisterUser",
			setupMock: func() {
				s.mockRepo.On("GetUserByEmail", mock.Anything, email).Return(nil, nil).Once()
				s.mockRepo.On("GetUserByUsername", mock.Anything, username).Return(nil, nil).Once()
				s.mockRepo.On("RegisterUser", mock.Anything, mock.Anything).Return(uuid.Nil, "", errors.New("db error")).Once()
			},
			jwtExpire:     "24h",
			expectedError: true,
			verify: func(res *dto.RegisterUserOutputDto) {
				s.Nil(res)
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()
			config.GlobalConfig.JWTExpire = tc.jwtExpire
			tc.setupMock()

			res, err := s.handler.RegisterUser(s.ctx, input)

			if tc.expectedError {
				s.Error(err)
			} else {
				s.NoError(err)
			}
			tc.verify(res)
			s.mockRepo.AssertExpectations(s.T())
		})
	}
}

func (s *AuthHandlerTestSuite) TestSignInUser() {
	userID := uuid.New()
	username := "johndoe"
	email := "john@example.com"
	password := "password123"
	hashedPassword, _ := utils.HashPassword(password)

	input := &dto.SignInUserInputDto{
		Body: struct {
			Email    string `json:"email" required:"true" doc:"E-mail Address" example:"example@mail.com"`
			Password string `json:"password" required:"true" doc:"Password"`
		}{
			Email:    email,
			Password: password,
		},
	}

	testCases := []struct {
		name          string
		setupMock     func()
		jwtExpire     string
		expectedError bool
		verify        func(res *dto.SignInUserOutputDto)
	}{
		{
			name: "Success - signs in user",
			setupMock: func() {
				s.mockRepo.On("GetUserByEmail", mock.Anything, email).Return(&struct {
					ID           uuid.UUID
					Username     string
					PasswordHash string
				}{ID: userID, Username: username, PasswordHash: hashedPassword}, nil).Once()
			},
			jwtExpire:     "1h",
			expectedError: false,
			verify: func(res *dto.SignInUserOutputDto) {
				s.NotNil(res)
				s.Equal(userID, res.Body.ID)
				s.Equal(username, res.Body.Username)
				s.Equal("authToken", res.AuthToken.Name)
				s.NotEmpty(res.AuthToken.Value)
				s.Equal(3600, res.AuthToken.MaxAge)
			},
		},
		{
			name: "Success - signs in user with invalid duration fallback",
			setupMock: func() {
				s.mockRepo.On("GetUserByEmail", mock.Anything, email).Return(&struct {
					ID           uuid.UUID
					Username     string
					PasswordHash string
				}{ID: userID, Username: username, PasswordHash: hashedPassword}, nil).Once()
			},
			jwtExpire:     "invalid-duration",
			expectedError: false,
			verify: func(res *dto.SignInUserOutputDto) {
				s.NotNil(res)
				s.Equal(userID, res.Body.ID)
				s.Equal(86400, res.AuthToken.MaxAge)
			},
		},
		{
			name: "Failure - user not found (wrong email)",
			setupMock: func() {
				s.mockRepo.On("GetUserByEmail", mock.Anything, email).Return(nil, errors.New("not found")).Once()
			},
			jwtExpire:     "24h",
			expectedError: true,
			verify: func(res *dto.SignInUserOutputDto) {
				s.Nil(res)
			},
		},
		{
			name: "Failure - wrong password",
			setupMock: func() {
				wrongHash, _ := utils.HashPassword("wrongpassword")
				s.mockRepo.On("GetUserByEmail", mock.Anything, email).Return(&struct {
					ID           uuid.UUID
					Username     string
					PasswordHash string
				}{ID: userID, Username: username, PasswordHash: wrongHash}, nil).Once()
			},
			jwtExpire:     "24h",
			expectedError: true,
			verify: func(res *dto.SignInUserOutputDto) {
				s.Nil(res)
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()
			config.GlobalConfig.JWTExpire = tc.jwtExpire
			tc.setupMock()

			res, err := s.handler.SignInUser(s.ctx, input)

			if tc.expectedError {
				s.Error(err)
			} else {
				s.NoError(err)
			}
			tc.verify(res)
			s.mockRepo.AssertExpectations(s.T())
		})
	}
}

func (s *AuthHandlerTestSuite) TestSignOutUser() {
	res, err := s.handler.SignOutUser(s.ctx, &dto.SignOutUserInputDto{})
	s.NoError(err)
	s.NotNil(res)
	s.Equal("authToken", res.AuthToken.Name)
	s.Equal("", res.AuthToken.Value)
	s.Equal(-1, res.AuthToken.MaxAge)
}

func (s *AuthHandlerTestSuite) TestForgotPassword() {
	userID := uuid.New()
	username := "johndoe"
	email := "john@example.com"

	input := &dto.ForgotPasswordInputDto{
		Body: struct {
			Email string `json:"email" required:"true" doc:"E-mail Address" example:"example@mail.com"`
		}{
			Email: email,
		},
	}

	testCases := []struct {
		name        string
		setupMock   func(done chan bool) // Accept channel for signaling
		expectEmail bool                 // Explicitly declare if this test case expects an email to be sent
		verify      func(res *dto.ForgotPasswordOutputDto)
	}{
		{
			name: "Success - sends reset link via email asynchronously",
			setupMock: func(done chan bool) {
				s.mockRepo.On("GetUserByEmail", mock.Anything, email).Return(&struct {
					ID           uuid.UUID
					Username     string
					PasswordHash string
				}{ID: userID, Username: username, PasswordHash: "hashed"}, nil).Once()

				s.mockEmail.On("SendResetPasswordEmail", email, username, mock.Anything).
					Return(nil).
					Run(func(args mock.Arguments) {
						done <- true // Signal email was sent
					}).
					Once()
			},
			expectEmail: true, // We expect email to be sent
			verify: func(res *dto.ForgotPasswordOutputDto) {
				s.NotNil(res)
				s.Equal("If your email is registered, you will receive a password reset link shortly.", res.Body.Message)
			},
		},
		{
			name: "Success - user not found (gracefully returns success message, no email sent)",
			setupMock: func(done chan bool) {
				s.mockRepo.On("GetUserByEmail", mock.Anything, email).Return(nil, errors.New("not found")).Once()
			},
			expectEmail: false, // No email should be sent
			verify: func(res *dto.ForgotPasswordOutputDto) {
				s.NotNil(res)
				s.Equal("If your email is registered, you will receive a password reset link shortly.", res.Body.Message)
			},
		},
		{
			name: "Success - repository error (gracefully returns success message, no email sent)",
			setupMock: func(done chan bool) {
				s.mockRepo.On("GetUserByEmail", mock.Anything, email).Return(nil, errors.New("db error")).Once()
			},
			expectEmail: false, // No email should be sent
			verify: func(res *dto.ForgotPasswordOutputDto) {
				s.NotNil(res)
				s.Equal("If your email is registered, you will receive a password reset link shortly.", res.Body.Message)
			},
		},
		{
			name: "Success - email service failure logged (still returns success response)",
			setupMock: func(done chan bool) {
				s.mockRepo.On("GetUserByEmail", mock.Anything, email).Return(&struct {
					ID           uuid.UUID
					Username     string
					PasswordHash string
				}{ID: userID, Username: username, PasswordHash: "hashed"}, nil).Once()

				// Even when SMTP returns error, the mock email service still sends a signal
				s.mockEmail.On("SendResetPasswordEmail", email, username, mock.Anything).
					Return(errors.New("smtp connection error")).
					Run(func(args mock.Arguments) {
						done <- true // Signal email call completed
					}).
					Once()
			},
			expectEmail: true, // We expect email call to happen
			verify: func(res *dto.ForgotPasswordOutputDto) {
				s.NotNil(res)
				s.Equal("If your email is registered, you will receive a password reset link shortly.", res.Body.Message)
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()
			done := make(chan bool, 1)
			tc.setupMock(done)

			res, err := s.handler.ForgotPassword(s.ctx, input)

			s.NoError(err)
			tc.verify(res)

			// Wait for the asynchronous email call to complete via the channel if expected.
			if tc.expectEmail {
				select {
				case <-done:
					// Signal received successfully. Proceed.
				case <-time.After(1 * time.Second):
					s.Fail("Timed out waiting for SendResetPasswordEmail to be called")
				}
			}

			s.mockRepo.AssertExpectations(s.T())
			s.mockEmail.AssertExpectations(s.T())
		})
	}
}

func (s *AuthHandlerTestSuite) TestResetPassword() {
	userID := uuid.New()
	secret := "test-secret-at-least-thirty-two-bytes-long"
	validToken, _ := utils.GenerateJWT(userID, secret, 10*time.Minute)

	testCases := []struct {
		name          string
		token         string
		setupMock     func()
		expectedError bool
		verify        func(res *dto.ResetPasswordOutputDto)
	}{
		{
			name:  "Success - resets password",
			token: validToken,
			setupMock: func() {
				s.mockRepo.On("UpdatePassword", mock.Anything, userID, mock.Anything).Return(nil).Once()
			},
			expectedError: false,
			verify: func(res *dto.ResetPasswordOutputDto) {
				s.NotNil(res)
				s.Equal("Password has been successfully updated.", res.Body.Message)
			},
		},
		{
			name:          "Failure - invalid token",
			token:         "invalid-token-string",
			setupMock:     func() {},
			expectedError: true,
			verify: func(res *dto.ResetPasswordOutputDto) {
				s.Nil(res)
			},
		},
		{
			name: "Failure - missing ID claim",
			token: func() string {
				claims := jwt.MapClaims{
					"exp": time.Now().Add(10 * time.Minute).Unix(),
				}
				token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
				tStr, _ := token.SignedString([]byte(secret))
				return tStr
			}(),
			setupMock:     func() {},
			expectedError: true,
			verify: func(res *dto.ResetPasswordOutputDto) {
				s.Nil(res)
			},
		},
		{
			name: "Failure - invalid user ID UUID format in claims",
			token: func() string {
				claims := jwt.MapClaims{
					"id":  "not-a-valid-uuid",
					"exp": time.Now().Add(10 * time.Minute).Unix(),
				}
				token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
				tStr, _ := token.SignedString([]byte(secret))
				return tStr
			}(),
			setupMock:     func() {},
			expectedError: true,
			verify: func(res *dto.ResetPasswordOutputDto) {
				s.Nil(res)
			},
		},
		{
			name:  "Failure - repository UpdatePassword database error",
			token: validToken,
			setupMock: func() {
				s.mockRepo.On("UpdatePassword", mock.Anything, userID, mock.Anything).Return(errors.New("db write failure")).Once()
			},
			expectedError: true,
			verify: func(res *dto.ResetPasswordOutputDto) {
				s.Nil(res)
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()
			tc.setupMock()

			input := &dto.ResetPasswordInputDto{
				Body: struct {
					Token    string `json:"token" required:"true" doc:"Reset Token"`
					Password string `json:"password" required:"true" doc:"New Password"`
				}{
					Token:    tc.token,
					Password: "newpassword123",
				},
			}

			res, err := s.handler.ResetPassword(s.ctx, input)

			if tc.expectedError {
				s.Error(err)
			} else {
				s.NoError(err)
			}
			tc.verify(res)
			s.mockRepo.AssertExpectations(s.T())
		})
	}
}

func TestAuthHandlerSuite(t *testing.T) {
	suite.Run(t, new(AuthHandlerTestSuite))
}
