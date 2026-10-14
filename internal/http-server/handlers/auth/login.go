package auth

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"photo-viewer-server/internal/lib/api/response"
	"photo-viewer-server/internal/lib/auth"
	"photo-viewer-server/internal/lib/logger/sl"
	"photo-viewer-server/internal/lib/random"
	"photo-viewer-server/internal/service"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	vlpkg "github.com/go-playground/validator/v10"
)

// LoginUser authenticates by login/password and issues an access token.
//
//	@Summary		Login
//	@Description	Returns an access token in the body and sets an HttpOnly refresh_token cookie (Path=/api/v1/auth/refresh).
//	@Tags				auth
//	@Accept			json
//	@Produce		json
//	@Param			body		body		service.UserAuthCredentials	true	"login (2-50), password"
//	@Success		200			{object}	auth.AccessTokenResponse
//	@Failure		400			{object}	response.Response	"invalid metadata / validation"
//	@Failure		403			{object}	response.Response	"invalid credentials / user not active / already authenticated"
//	@Failure		500			{object}	response.Response	"failed to create tokens"
//	@Router			/auth/login [post]
func LoginUser(lg *slog.Logger, validator *vlpkg.Validate, apiPrefix string, jwtAccessSecret string, jwtRefreshSecret string, userService *service.UserService, authConfig *auth.AuthConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := lg.With(
			slog.String("op", "handlers.auth.LoginUser"),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		var userCredentials service.UserAuthCredentials
		if err := json.NewDecoder(r.Body).Decode(&userCredentials); err != nil {
			log.Error("failed to decode metadata", sl.Err(err))

			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, response.Error("invalid metadata"))
			return
		}

		log.Debug("request metadata decoded", slog.String("login", userCredentials.Login))

		if err := validator.Struct(userCredentials); err != nil {
			validateErr, ok := err.(vlpkg.ValidationErrors)
			if !ok {
				log.Error("unknown validation error", sl.Err(err))

				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, response.Error("validation error"))
				return
			}

			log.Error("error validate request metadata", sl.Err(err))

			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, response.ValidationErrors(validateErr))
			return
		}

		user, err := userService.AuthenticateUser(r.Context(), userCredentials)

		if err != nil {
			log.Error("failed to authenticate user", sl.Err(err))

			duration, err := random.CryptoRandInt64(100, 1500)
			if err != nil {
				duration = 750
			}
			time.Sleep(time.Duration(duration) * time.Millisecond)

			render.Status(r, http.StatusForbidden)
			render.JSON(w, r, response.Error("invalid credentials"))
			return
		}

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
