package omitemptyjwtmw

import (
	"context"
	"errors"
	"net/http"
	"photo-viewer-server/internal/lib/api/response"
	"photo-viewer-server/internal/lib/auth"
	"strings"

	"github.com/go-chi/render"
)
func New(jwtSecret string) func(next http.Handler) http.Handler {
	jwtSecretBytes := []byte(jwtSecret)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")

			if authHeader == "" {
				next.ServeHTTP(w, r)
				return
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				next.ServeHTTP(w, r)
				return
			}

			tokenString := parts[1]
			claims, err := auth.ParseAccessToken(tokenString, jwtSecretBytes)

			if err != nil {
				if errors.Is(err, auth.ErrInvalidToken) {
					render.Status(r, http.StatusUnauthorized)
					render.JSON(w, r, response.Error("invalid token"))
				}

				next.ServeHTTP(w, r)
				return
			}

			ctx := context.WithValue(r.Context(), "user_uuid", claims.UserUuid)
			ctx = context.WithValue(ctx, "user_role", claims.Role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

