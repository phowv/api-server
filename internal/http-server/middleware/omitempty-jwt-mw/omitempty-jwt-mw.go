package omitemptyjwtmw

import (
	"context"
	"net/http"
	"photo-viewer-server/internal/lib/auth"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)
func New(jwtSecret string) func(next http.Handler) http.Handler {
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
			claims := &auth.Claims{}

			token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
				return []byte(jwtSecret), nil
			})

			if err != nil || !token.Valid {
				next.ServeHTTP(w, r)
				return
			}
			
			ctx := context.WithValue(r.Context(), "user_uuid", claims.UserUuid)
			ctx = context.WithValue(ctx, "user_role", claims.Role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

