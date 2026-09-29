package update

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"photo-viewer-server/internal/lib/api/response"
	"photo-viewer-server/internal/lib/logger/sl"
	"photo-viewer-server/internal/lib/validatorx"
	"photo-viewer-server/internal/service"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

// UpdateUser patches a user's role and quotas (admin only). All fields are optional. Update role possible only to other users.
//
//	@Summary		Update user (role / quotas)
//	@Tags				users
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			user_uuid	path		string	true	"User UUID"
//	@Param			body		body	service.PatchUserRequest	true	"user_role (user|moderator), photos_quota (≥0), collections_quota (≥0)"
//	@Success		200			{object}	response.Response
//	@Failure		400			{object}	response.Response	"invalid metadata / validation / invalid request"
//	@Failure		401			{object}	response.Response	"token is empty / invalid token"
//	@Failure		403			{object}	response.Response	"access denied (admin role required)"
//	@Failure		404			{object}	response.Response	"user not found"
//	@Failure		500			{object}	response.Response	"internal error"
//	@Router			/users/{user_uuid} [patch]
func UpdateUser(lg *slog.Logger, adminService *service.AdminService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := lg.With(
			slog.String("op", "handlers.view.ViewCollection"),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		var metadata service.PatchUserRequest
		if err := json.NewDecoder(r.Body).Decode(&metadata); err != nil {
			log.Error("failed to decode metadata", sl.Err(err))

			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, response.Error("invalid metadata"))
			return
		}

		log.Info("request metadata decoded", slog.Any("metadata", metadata))

		if err := validatorx.NewValidator().Struct(metadata); err != nil {
			validateErr := err.(validator.ValidationErrors)

			log.Error("error validate request metadata", sl.Err(err))

			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, response.ValidationErrors(validateErr))
			return
		}

		userUuidStr := chi.URLParam(r, "user_uuid")
		if userUuidStr == "" {
			log.Info("collection uiid param is empty")

			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, response.Error("user uuid is empty"))
			return
		}

		userUuid, err := uuid.Parse(userUuidStr)
		if err != nil {
			log.Error("failed to convert photo id to int", slog.String("user_uuid_str", userUuidStr))

			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, response.Error("invalid request"))
			return
		}

		err = adminService.PatchUser(r.Context(), userUuid, &metadata)

		if err != nil {
			log.Error("failed to patch user", sl.Err(err))

			switch {
			case errors.Is(err, service.ErrUserNotFound):
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, response.Error("user not found"))
			case errors.Is(err, service.ErrOperationIsNotPermitted):
				render.Status(r, http.StatusForbidden)
				render.JSON(w, r, response.Error("operation is not permitted"))
			default:
				render.Status(r, http.StatusInternalServerError)
				render.JSON(w, r, response.Error("internal error"))
			}
			return
		}

		render.Status(r, http.StatusOK)
		render.JSON(w, r, response.OK())
	}
}
