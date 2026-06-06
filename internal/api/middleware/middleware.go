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

		userID, _ := claims["id"].(string)
		var isAdmin bool
		if val, ok := claims["isAdmin"].(bool); ok {
			isAdmin = val
		} else if valFloat, ok := claims["isAdmin"].(float64); ok {
			isAdmin = valFloat != 0
		}

		newCtx := context.WithValue(ctx.Context(), "userID", userID)
		newCtx = context.WithValue(newCtx, "isAdmin", isAdmin)
		newHumaCtx := huma.WithContext(ctx, newCtx)

		// Continue to the next function
		next(newHumaCtx)
	}
}

// RequireAdmin ensures that the authenticated user is an admin.
func RequireAdmin(api huma.API) func(ctx huma.Context, next func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		isAdmin, ok := ctx.Context().Value("isAdmin").(bool)
		if !ok || !isAdmin {
			huma.WriteErr(api, ctx, http.StatusForbidden, "Forbidden: Admin access required")
			return
		}
		next(ctx)
	}
}

// RequireCustomer ensures that the authenticated user is not an admin.
func RequireCustomer(api huma.API) func(ctx huma.Context, next func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		isAdmin, ok := ctx.Context().Value("isAdmin").(bool)
		if ok && isAdmin {
			huma.WriteErr(api, ctx, http.StatusForbidden, "Forbidden: Customer access required")
			return
		}
		next(ctx)
	}
}
