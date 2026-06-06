package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/stretchr/testify/assert"
	"golang.org/x/time/rate"
)

func TestRateLimiterMiddleware(t *testing.T) {
	e := echo.New()

	rateLimitLimit := 3
	rateLimitPeriod := 2 * time.Second

	rateLimiter := middleware.RateLimiterWithConfig(middleware.RateLimiterConfig{
		Skipper: func(c echo.Context) bool {
			path := c.Path()
			return path != "/auth/signin" && path != "/auth/register" && path != "/auth/forgot-password"
		},
		Store: middleware.NewRateLimiterMemoryStoreWithConfig(
			middleware.RateLimiterMemoryStoreConfig{
				Rate:      rate.Limit(float64(rateLimitLimit) / rateLimitPeriod.Seconds()),
				Burst:     rateLimitLimit,
				ExpiresIn: rateLimitPeriod,
			},
		),
		IdentifierExtractor: func(ctx echo.Context) (string, error) {
			return ctx.RealIP(), nil
		},
		ErrorHandler: func(context echo.Context, err error) error {
			return context.JSON(http.StatusTooManyRequests, map[string]string{"message": "Too many requests"})
		},
		DenyHandler: func(context echo.Context, identifier string, err error) error {
			return context.JSON(http.StatusTooManyRequests, map[string]string{"message": "Too many requests"})
		},
	})

	e.Use(rateLimiter)

	e.GET("/auth/signin", func(c echo.Context) error {
		return c.String(http.StatusOK, "success")
	})

	e.GET("/health", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	// 1. First 3 requests to signin should succeed (200)
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/auth/signin", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	}

	// 2. 4th request to signin should fail with 429
	req := httptest.NewRequest(http.MethodGet, "/auth/signin", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)

	// 3. Request to /health should NOT be rate limited (since it is skipped)
	reqHealth := httptest.NewRequest(http.MethodGet, "/health", nil)
	recHealth := httptest.NewRecorder()
	e.ServeHTTP(recHealth, reqHealth)
	assert.Equal(t, http.StatusOK, recHealth.Code)
}
