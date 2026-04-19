package middleware

import (
	"context"
	"lorem-backend/internal/config"
	"lorem-backend/internal/utils"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/labstack/echo/v4"
)

func VerifyToken(api huma.API) func(ctx huma.Context, next func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		authToken, err := huma.ReadCookie(ctx, "authToken")
		if err != nil {
			huma.WriteErr(api, ctx, http.StatusUnauthorized, "Unauthorized")
			return
		}

		claims, err := utils.VerifyJWT(authToken.Value, config.GlobalConfig.JWTSecret)
		if err != nil {
			huma.WriteErr(api, ctx, http.StatusForbidden, "Forbidden")
			return
		}

		userID := claims["id"].(string)
		newCtx := context.WithValue(ctx.Context(), "userID", userID)
		newHumaCtx := huma.WithContext(ctx, newCtx)

		// Continue to the next function
		next(newHumaCtx)
	}
}

func VerifyTokenEcho(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		cookie, err := c.Cookie("authToken")
		if err != nil {
			return c.JSON(http.StatusUnauthorized, utils.CreateErrorResponse(http.StatusUnauthorized, "Unauthorized"))
		}

		claims, err := utils.VerifyJWT(cookie.Value, config.GlobalConfig.JWTSecret)
		if err != nil {
			return c.JSON(http.StatusForbidden, utils.CreateErrorResponse(http.StatusForbidden, "Forbidden"))
		}

		// Set the userID in the standard Echo context
		userID := claims["id"].(string)
		c.Set("userID", userID)

		return next(c)
	}
}

// TODO: Implement Role Checking Middleware
