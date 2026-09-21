package jwttoken

import (
	"net/http"
	"photo-viewer-server/internal/lib/api/response"
	"photo-viewer-server/internal/lib/auth"
	"strings"

	"github.com/go-chi/render"
)

func ExtractToken(secret []byte, w http.ResponseWriter, r *http.Request) *auth.Claims {
	authHeader := r.Header.Get("Authorization")

	if authHeader == "" {
		render.Status(r, http.StatusUnauthorized)
		render.JSON(w, r, response.Error("token is empty"))
		return nil
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		render.Status(r, http.StatusUnauthorized)
		render.JSON(w, r, response.Error("invalid header format"))
		return nil
	}

	tokenString := parts[1]
	claims, err := auth.ParseAccessToken(tokenString, secret)

	if err != nil {
		render.Status(r, http.StatusUnauthorized)
		render.JSON(w, r, response.Error("invalid token"))
		return nil
	}

	return claims
}
