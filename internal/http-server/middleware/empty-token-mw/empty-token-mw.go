package emptytokenmw

import (
	"context"
	"net/http"
	"photo-viewer-server/internal/lib/api/response"
	"photo-viewer-server/internal/lib/auth"
	"strings"

	"github.com/go-chi/render"
	"github.com/golang-jwt/jwt/v5"
)

func New(jwtSecret string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")

			parts := strings.Split(authHeader, " ")
			if len(parts) == 2 && parts[0] == "Bearer" {
				tokenString := parts[1]
				claims := &auth.Claims{}

				token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
					return []byte(jwtSecret), nil
				})

				if err == nil && token.Valid {
					render.Status(r, http.StatusForbidden)
					render.JSON(w, r, response.Error("already authenticated"))
					return
				}
			}

			ctx := context.WithValue(r.Context(), "user_role", "anonymous")
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
