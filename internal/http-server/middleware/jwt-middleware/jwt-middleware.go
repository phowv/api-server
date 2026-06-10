package jwtmiddleware

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
			claims := &auth.Claims{}

			token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
				return []byte(jwtSecret), nil
			})

			if err != nil || !token.Valid {
				render.Status(r, http.StatusUnauthorized)
				render.JSON(w, r, response.Error("invalid token"))
				return
			}
			
			ctx := context.WithValue(r.Context(), "user_id", claims.UserId)
			ctx = context.WithValue(ctx, "user_role", claims.Role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
