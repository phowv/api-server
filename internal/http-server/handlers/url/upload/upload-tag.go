package upload

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"photo-viewer-server/internal/lib/api/response"
	"photo-viewer-server/internal/lib/logger/sl"
	"photo-viewer-server/internal/lib/validatorx"
	"photo-viewer-server/internal/service"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	vlpkg "github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

type TagCreateResponse struct {
	response.Response
	TagUuid uuid.UUID `json:"tag_uuid"`
}

// UploadTag creates a tag with a unique name.
//
//	@Summary		Create a tag
//	@Tags			  tags
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body		body service.SaveTagInput true	"tag_name (2-50, unique), tag_description (≤200)"
//	@Success		201			{object}	upload.TagCreateResponse
//	@Failure		400			{object}	response.Response	"invalid metadata / validation / tag with this name already exists"
//	@Failure		401			{object}	response.Response	"token is empty / invalid token"
//	@Failure		500			{object}	response.Response	"internal error"
//	@Router			/tags [post]
func UploadTag(lg *slog.Logger, validator *vlpkg.Validate, tagService *service.TagService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := lg.With(
			slog.String("op", "handlers.upload.UploadTag"),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		var tagInfo service.SaveTagInput
		if err := json.NewDecoder(r.Body).Decode(&tagInfo); err != nil {
			log.Error("failed to decode metadata", sl.Err(err))

			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, response.Error("invalid metadata"))
			return
		}

		if err := validatorx.NewValidator().Struct(tagInfo); err != nil {
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
			TagUuid:  tagUuid,
		})
	}
}
