package middleware

import (
	"net/http"

	"github.com/danielgtaylor/huma/v2"
)

// RequireOwnershipOrAdmin checks if the authenticated user matches the path parameters 'id' or 'userId'.
// Admins are always allowed access.
func RequireOwnershipOrAdmin(api huma.API) func(ctx huma.Context, next func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		isAdmin, _ := ctx.Context().Value("isAdmin").(bool)
		if isAdmin {
			next(ctx)
			return
		}

		userIDStr, ok := ctx.Context().Value("userID").(string)
		if !ok || userIDStr == "" {
			huma.WriteErr(api, ctx, http.StatusUnauthorized, "Unauthorized")
			return
		}

		// Check path parameters 'id' or 'userId'
		idParam := ctx.Param("id")
		userIdParam := ctx.Param("userId")

		// If either parameter is present, it must match the authenticated userID
		if idParam != "" && idParam != userIDStr {
			huma.WriteErr(api, ctx, http.StatusForbidden, "Forbidden: You do not own this resource")
			return
		}

		if userIdParam != "" && userIdParam != userIDStr {
			huma.WriteErr(api, ctx, http.StatusForbidden, "Forbidden: You do not own this resource")
			return
		}

		next(ctx)
	}
}
