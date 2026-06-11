package emptytokenmw

import (
	"context"
	"net/http"
	"photo-viewer-server/internal/lib/api/response"

	"github.com/go-chi/render"
)

func New() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")

			if authHeader != "" {
				render.Status(r, http.StatusForbidden)
				render.JSON(w, r, response.Error("already authenticated"))
				return
			}

			ctx := context.WithValue(r.Context(), "user_role", "anonymous")
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
