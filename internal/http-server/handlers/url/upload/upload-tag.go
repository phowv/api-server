package upload

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
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

type TagCreateResponse struct {
	response.Response
  TagUuid uuid.UUID `json:"tag_uuid"`
}

func UploadTag(lg *slog.Logger, tagService *service.TagService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := lg.With(
			slog.String("op", "handlers.upload.UploadTag"),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		jsonMetadata := r.FormValue("metadata")

		var tagInfo service.SaveTagInput
		if err := json.Unmarshal([]byte(jsonMetadata), &tagInfo); err != nil {
			log.Error("failed to decode metadata", sl.Err(err))

			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, response.Error("invalid metadata"))
			return
		}

		if err := validator.New().Struct(tagInfo); err != nil {
			validateErr := err.(validator.ValidationErrors)

			log.Error("error validate request metadata", sl.Err(err))

			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, response.ValidationErrors(validateErr))
			return
		}

		log.Info("request metadata decoded", slog.Any("metadata", tagInfo))

		tagUuid, err := tagService.SaveTag(r.Context(), tagInfo)

		if err != nil {
			log.Error("failed to save tag", sl.Err(err))

			if errors.Is(err, service.ErrTagAlreadyExists) {
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, response.Error("tag with this name already exists"))
				return
			}

			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, response.Error("internal error"))
			return
		}

		log.Info("saved tag", slog.Any("tag_uuid", tagUuid))

		render.Status(r, http.StatusCreated)
		render.JSON(w, r, TagCreateResponse{
			Response: response.OK(),
			TagUuid: tagUuid,
		})
	}
}
