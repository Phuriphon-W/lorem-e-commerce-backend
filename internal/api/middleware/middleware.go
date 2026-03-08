package middleware

import (
	"context"
	"lorem-backend/internal/config"
	"lorem-backend/internal/utils"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
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
		huma.WithContext(ctx, newCtx)

		// Continue to the next function
		next(ctx)
	}
}

// TODO: Implement Role Checking Middleware
