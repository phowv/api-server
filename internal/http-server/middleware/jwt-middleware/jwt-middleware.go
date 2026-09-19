package jwtmiddleware

import (
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
				render.Status(r, http.StatusUnauthorized)
				render.JSON(w, r, response.Error("token is empty"))
				return
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				render.Status(r, http.StatusUnauthorized)
				render.JSON(w, r, response.Error("invalid header format"))
				return
			}

			tokenString := parts[1]
			claims, err := auth.ParseAccessToken(tokenString, jwtSecretBytes)

			if err != nil {
				render.Status(r, http.StatusUnauthorized)
				render.JSON(w, r, response.Error("invalid token"))
				return
			}
			
			ctx := auth.ApplyAccessTokenClaims(r.Context(), claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
