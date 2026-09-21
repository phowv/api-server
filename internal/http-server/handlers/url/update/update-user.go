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

func UpdateUser(lg *slog.Logger, adminService *service.AdminService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := lg.With(
			slog.String("op", "handlers.view.ViewCollection"),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		jsonMetadata := r.FormValue("metadata")

		var metadata service.PatchUserRequest
		if err := json.Unmarshal([]byte(jsonMetadata), &metadata); err != nil {
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

			if errors.Is(err, service.ErrUserNotFound) {
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, response.Error("user not found"))
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
