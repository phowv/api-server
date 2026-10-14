package jwtmiddleware

import (
	"net/http"
	jwttoken "photo-viewer-server/internal/lib/api/jwt-token"
	"photo-viewer-server/internal/lib/auth"
)

func New(jwtSecret string) func(next http.Handler) http.Handler {
	jwtSecretBytes := []byte(jwtSecret)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := jwttoken.ExtractToken(jwtSecretBytes, w, r)
			if claims == nil {
				return
			}

			ctx := auth.ApplyAccessTokenClaims(r.Context(), claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
