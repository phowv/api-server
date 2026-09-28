package auth

import(
	"encoding/json"
	"log/slog"
	"net/http"
	"photo-viewer-server/internal/lib/api/response"
	"photo-viewer-server/internal/lib/logger/sl"
	"photo-viewer-server/internal/lib/random"
	"photo-viewer-server/internal/service"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	vlpkg "github.com/go-playground/validator/v10"
)

// VerifyUser confirms the email verification code and activates the user.
//
//	@Summary		Verify user email
//	@Tags				auth
//	@Accept			json
//	@Produce		json
//	@Param			body		body		service.UserVerifyCredentials	true	"login (2-50), code"
//	@Success		200			{object}	response.Response
//	@Failure		400			{object}	response.Response	"invalid metadata / validation"
//	@Failure		403			{object}	response.Response	"invalid code / already authenticated"
//	@Router			/auth/verify [post]
func VerifyUser(lg *slog.Logger, validator *vlpkg.Validate, userService *service.UserService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := lg.With(
			slog.String("op", "handlers.auth.VerifyUser"),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		var userVerifyCredentials service.UserVerifyCredentials
		if err := json.NewDecoder(r.Body).Decode(&userVerifyCredentials); err != nil {
			log.Error("failed to decode metadata", sl.Err(err))

			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, response.Error("invalid metadata"))
			return
		}

		log.Debug("request metadata decoded", slog.String("login", userVerifyCredentials.Login))

		if err := validator.Struct(userVerifyCredentials); err != nil {
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

		err := userService.VerifyUser(r.Context(), userVerifyCredentials)
		if err != nil {
			log.Error("failed to verify user", sl.Err(err))

			duration, err := random.CryptoRandInt64(100, 1500)
			if err != nil {
				duration = 750
			}
			time.Sleep(time.Duration(duration) * time.Millisecond)

			render.Status(r, http.StatusForbidden)
			render.JSON(w, r, response.Error("invalid code"))
			return
		}

		render.Status(r, http.StatusOK)
		render.JSON(w, r, response.OK())
	}
}
