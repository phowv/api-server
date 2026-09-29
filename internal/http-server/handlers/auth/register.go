package auth

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"photo-viewer-server/internal/lib/api/response"
	"photo-viewer-server/internal/lib/logger/sl"
	"photo-viewer-server/internal/service"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	vlpkg "github.com/go-playground/validator/v10"
)

// RegisterUser creates a new inactive user and sends a verification code.
//
//	@Summary		Register a new user
//	@Description	Account is inactive until the verification code is confirmed via /auth/verify.
//	@Tags				auth
//	@Accept			json
//	@Produce		json
//	@Param			body		body		service.UserData	true	"login (2-50), email (2-50), password (8-50), description"
//	@Success		201			{object}	response.Response
//	@Failure		400			{object}	response.Response	"invalid metadata / validation / user already exists / password too short"
//	@Failure		403			{object}	response.Response	"already authenticated"
//	@Router			/auth/register [post]
func RegisterUser(lg *slog.Logger, validator *vlpkg.Validate, userService *service.UserService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := lg.With(
			slog.String("op", "handlers.auth.RegisterUser"),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		var userData service.UserData
		if err := json.NewDecoder(r.Body).Decode(&userData); err != nil {
			log.Error("failed to decode metadata", sl.Err(err))

			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, response.Error("invalid metadata"))
			return
		}

		log.Debug("request metadata decoded", slog.String("login", userData.Login), slog.String("email", userData.Email))

		if err := validator.Struct(userData); err != nil {
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

		_, err := userService.CreateUser(r.Context(), userData)

		if err != nil {
			log.Error("error create user", sl.Err(err))

			if errors.Is(err, service.ErrUserExists) {
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, response.Error("error user already exists"))
				return

			} else if errors.Is(err, service.ErrUserPasswordTooShort) {
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, response.Error("error user password too short, must be longer than 8 symbols"))
				return
			}

			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, response.Error("error register new user"))
			return
		}

		render.Status(r, http.StatusCreated)
		render.JSON(w, r, response.OK())
	}
}
