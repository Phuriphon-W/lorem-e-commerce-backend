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

type OwnershipTestSuite struct {
	suite.Suite
	testAPI humatest.TestAPI
	secret  string
}

func (s *OwnershipTestSuite) SetupTest() {
	s.secret = "test-secret"
	config.GlobalConfig = &config.Config{
		JWTSecret: s.secret,
	}
	_, s.testAPI = humatest.New(s.T(), huma.DefaultConfig("Test API", "1.0.0"))
	s.testAPI.UseMiddleware(middleware.VerifyToken(s.testAPI))

	huma.Register(s.testAPI, huma.Operation{
		Method:      http.MethodGet,
		Path:        "/user/{id}",
		OperationID: "ownership-sentinel-id",
		Middlewares: huma.Middlewares{middleware.RequireOwnershipOrAdmin(s.testAPI)},
	}, func(ctx context.Context, input *struct {
		ID string `path:"id"`
	}) (*struct{}, error) {
		return nil, nil
	})

	huma.Register(s.testAPI, huma.Operation{
		Method:      http.MethodGet,
		Path:        "/user/{userId}/orders",
		OperationID: "ownership-sentinel-userid",
		Middlewares: huma.Middlewares{middleware.RequireOwnershipOrAdmin(s.testAPI)},
	}, func(ctx context.Context, input *struct {
		UserID string `path:"userId"`
	}) (*struct{}, error) {
		return nil, nil
	})
}

func (s *OwnershipTestSuite) TestRequireOwnershipOrAdmin() {
	userA := uuid.New()
	userB := uuid.New()

	tokenA, err := utils.GenerateJWT(userA, false, s.secret, 10*time.Minute)
	s.Require().NoError(err)

	adminToken, err := utils.GenerateJWT(uuid.New(), true, s.secret, 10*time.Minute)
	s.Require().NoError(err)

	// Case 1: User A requests their own resource -> Success (204)
	resp := s.testAPI.Get(fmt.Sprintf("/user/%s", userA), fmt.Sprintf("Cookie: authToken=%s", tokenA)).Result()
	s.Equal(http.StatusNoContent, resp.StatusCode)

	// Case 2: User A requests User B's resource -> Forbidden (403)
	resp = s.testAPI.Get(fmt.Sprintf("/user/%s", userB), fmt.Sprintf("Cookie: authToken=%s", tokenA)).Result()
	s.Equal(http.StatusForbidden, resp.StatusCode)

	// Case 3: Admin requests User B's resource -> Success (204)
	resp = s.testAPI.Get(fmt.Sprintf("/user/%s", userB), fmt.Sprintf("Cookie: authToken=%s", adminToken)).Result()
	s.Equal(http.StatusNoContent, resp.StatusCode)

	// Case 4: User A requests their own orders list -> Success (204)
	resp = s.testAPI.Get(fmt.Sprintf("/user/%s/orders", userA), fmt.Sprintf("Cookie: authToken=%s", tokenA)).Result()
	s.Equal(http.StatusNoContent, resp.StatusCode)

	// Case 5: User A requests User B's orders list -> Forbidden (403)
	resp = s.testAPI.Get(fmt.Sprintf("/user/%s/orders", userB), fmt.Sprintf("Cookie: authToken=%s", tokenA)).Result()
	s.Equal(http.StatusForbidden, resp.StatusCode)

	// Case 6: Admin requests User B's orders list -> Success (204)
	resp = s.testAPI.Get(fmt.Sprintf("/user/%s/orders", userB), fmt.Sprintf("Cookie: authToken=%s", adminToken)).Result()
	s.Equal(http.StatusNoContent, resp.StatusCode)
}

func TestOwnershipSuite(t *testing.T) {
	suite.Run(t, new(OwnershipTestSuite))
}
