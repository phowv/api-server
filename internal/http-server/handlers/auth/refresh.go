package auth

import (
	"fmt"
	"log/slog"
	"net/http"
	"photo-viewer-server/internal/lib/api/response"
	"photo-viewer-server/internal/lib/auth"
	"photo-viewer-server/internal/lib/logger/sl"
	"photo-viewer-server/internal/service"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/golang-jwt/jwt/v5"
)

// RefreshUser rotates the session using the refresh_token cookie.
//
//	@Summary		Refresh access token
//	@Description	Requires the refresh_token cookie. Revokes the current session and issues a new access token and cookie.
//	@Tags				auth
//	@Produce		json
//	@Success		200			{object}	auth.AccessTokenResponse
//	@Failure		401			{object}	response.Response	"missing authentication / invalid token"
//	@Failure		403			{object}	response.Response	"already authenticated"
//	@Failure		500			{object}	response.Response	"failed to create tokens"
//	@Router			/auth/refresh [post]
func RefreshUser(lg *slog.Logger, apiPrefix string, jwtAccessSecret string, jwtRefreshSecret string, userService *service.UserService, authConfig *auth.AuthConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := lg.With(
			slog.String("op", "handlers.auth.RefreshUser"),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		cookie, err := r.Cookie("refresh_token")
		if err != nil {
			log.Debug("missing refresh token cookie")

			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, response.Error("missing authentication"))
			return
		}

		log.Info("parsed cookie refresh token")

		refreshTokenString := cookie.Value
		refreshClaims := &auth.RefreshClaims{}

		refreshToken, err := jwt.ParseWithClaims(refreshTokenString, refreshClaims, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected singing method: %v", token.Header["alg"])
			}

			return []byte(jwtRefreshSecret), nil
		}, jwt.WithValidMethods([]string{jwt.SigningMethodHS512.Name}))

		if err != nil || !refreshToken.Valid {
			log.Debug("invalid refresh token")

			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, response.Error("invalid token"))
			return
		}

		SessionUuid := refreshClaims.SessionUuid
		userUuid := refreshClaims.UserUuid

		log.Debug("parsed user uuid", slog.Any("user_uuid", userUuid))

		user, err := userService.RefreshSession(r.Context(), SessionUuid, userUuid, refreshTokenString)
		if err != nil {
			log.Error("failed to authenticate session", sl.Err(err))

			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, response.Error("invalid token"))
			return
		}
		log.Debug("success authenticate user session")

		tokens, err := createJwtTokens(r.Context(), userService, user, apiPrefix, jwtAccessSecret, jwtRefreshSecret, authConfig)
		if err != nil {
			log.Error("failed to create jwt token pair", sl.Err(err))

			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, response.Error("failed to create tokens"))
			return
		}

		sendJwtTokens(w, r, tokens)
	}
}
