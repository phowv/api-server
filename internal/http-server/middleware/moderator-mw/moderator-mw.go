package moderatormw

import (
	"net/http"
	jwttoken "photo-viewer-server/internal/lib/api/jwt-token"
	"photo-viewer-server/internal/lib/api/response"
	"photo-viewer-server/internal/lib/auth"

	"github.com/go-chi/render"
)
func New(jwtSecret string) func(next http.Handler) http.Handler {
	jwtSecretBytes := []byte(jwtSecret)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := jwttoken.ExtractToken(jwtSecretBytes, w, r)
			if claims == nil {
				return
			}

			if auth.IsLessThanModerator(claims.Role) {
				render.Status(r, http.StatusForbidden)
				render.JSON(w, r, response.Error("access denied"))
				return
			}
			
			ctx := auth.ApplyAccessTokenClaims(r.Context(), claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

