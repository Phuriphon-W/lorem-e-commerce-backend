package middleware_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"lorem-backend/internal/api/middleware"
	"lorem-backend/internal/config"
	"lorem-backend/internal/utils"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

// MiddlewareTestSuite tests the VerifyToken middleware via humatest.
type MiddlewareTestSuite struct {
	suite.Suite
	testAPI humatest.TestAPI
	secret  string
}

func (s *MiddlewareTestSuite) SetupTest() {
	s.secret = "test-secret"
	config.GlobalConfig = &config.Config{
		JWTSecret: s.secret,
	}
	_, s.testAPI = humatest.New(s.T(), huma.DefaultConfig("Test API", "1.0.0"))
	s.testAPI.UseMiddleware(middleware.VerifyToken(s.testAPI))

	huma.Register(s.testAPI, huma.Operation{
		Method:      http.MethodGet,
		Path:        "/test",
		OperationID: "test-sentinel",
	}, func(ctx context.Context, input *struct{}) (*struct{}, error) {
		return nil, nil
	})
}

func (s *MiddlewareTestSuite) TestVerifyToken() {
	userID := uuid.New()

	// 1. Success - valid token
	validToken, err := utils.GenerateJWT(userID, s.secret, 10*time.Minute)
	s.Require().NoError(err)

	// 3. Failure - expired JWT
	expiredToken, err := utils.GenerateJWT(userID, s.secret, -10*time.Minute)
	s.Require().NoError(err)

	// 5. Failure - wrong secret
	wrongSecretToken, err := utils.GenerateJWT(userID, "wrong-secret", 10*time.Minute)
	s.Require().NoError(err)

	tests := []struct {
		name         string
		cookieHeader string
		expectedCode int
	}{
		{
			name:         "Success - valid token",
			cookieHeader: fmt.Sprintf("Cookie: authToken=%s", validToken),
			expectedCode: http.StatusNoContent,
		},
		{
			name:         "Failure - missing authToken cookie",
			cookieHeader: "",
			expectedCode: http.StatusUnauthorized,
		},
		{
			name:         "Failure - expired JWT",
			cookieHeader: fmt.Sprintf("Cookie: authToken=%s", expiredToken),
			expectedCode: http.StatusForbidden,
		},
		{
			name:         "Failure - malformed token string",
			cookieHeader: "Cookie: authToken=not.a.jwt",
			expectedCode: http.StatusForbidden,
		},
		{
			name:         "Failure - wrong secret",
			cookieHeader: fmt.Sprintf("Cookie: authToken=%s", wrongSecretToken),
			expectedCode: http.StatusForbidden,
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			s.SetupTest() // Clean slate for each test case

			var resp *http.Response
			if tc.cookieHeader != "" {
				resp = s.testAPI.Get("/test", tc.cookieHeader).Result()
			} else {
				resp = s.testAPI.Get("/test").Result()
			}

			s.Equal(tc.expectedCode, resp.StatusCode)
		})
	}
}

func TestMiddlewareSuite(t *testing.T) {
	suite.Run(t, new(MiddlewareTestSuite))
}
