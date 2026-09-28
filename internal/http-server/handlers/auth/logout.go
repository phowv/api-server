package auth

import (
	"errors"
	"log/slog"
	"net/http"
	"photo-viewer-server/internal/lib/api/response"
	"photo-viewer-server/internal/lib/logger/sl"
	"photo-viewer-server/internal/service"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/google/uuid"
)

// LogoutUser revokes the current session.
//
//	@Summary		Logout
//	@Description	Revokes the session bound to the current access token.
//	@Tags				auth
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200			{object}	response.Response
//	@Failure		401			{object}	response.Response	"token is empty / invalid token"
//	@Failure		404			{object}	response.Response	"session not found"
//	@Failure		500			{object}	response.Response	"internal error"
//	@Router			/auth/logout [post]
func LogoutUser(lg *slog.Logger, userService *service.UserService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := lg.With(
			slog.String("op", "handlers.auth.LogoutUser"),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		sessionUuid, ok := r.Context().Value("session_uuid").(uuid.UUID)
		if !ok {
			log.Error("failed to get session_uuid from context")

			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, response.Error("invalid session_uuid"))
			return
		}

		err := userService.RevokeSession(r.Context(), sessionUuid)

		if err != nil {
			log.Error("failed to get revoke session", sl.Err(err))

			if errors.Is(err, service.ErrSessionNotFound) {
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, response.Error("session not found"))
				return
			}

			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, response.Error("internal error"))
			return
		}

		render.Status(r, http.StatusOK)
		render.JSON(w, r, response.OK())
	}
}

