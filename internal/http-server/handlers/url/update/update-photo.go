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
	"photo-viewer-server/internal/storage"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

type Response struct {
	response.Response
	PhotoUuid uuid.UUID `json:"photo_uuid"`
}

// UpdatePhoto patches photo metadata (title, description, access level, dates).
//
//	@Summary		Update photo metadata
//	@Tags				photos
//	@Accept			multipart/form-data
//	@Produce		json
//	@Security		BearerAuth
//	@Param			photo_uuid	path		string	true	"Photo UUID"
//	@Param			metadata	formData	string	true	"JSON: title (≤50), description (≤200), created_at, took_at, access_level (private|protected|public)"
//	@Success		200			{object}	update.Response
//	@Failure		400			{object}	response.Response	"invalid metadata / validation / invalid request"
//	@Failure		401			{object}	response.Response	"token is empty / invalid token"
//	@Failure		404			{object}	response.Response	"photo not found"
//	@Failure		500			{object}	response.Response	"internal error"
//	@Router			/photo/{photo_uuid} [patch]
func UpdatePhoto(lg *slog.Logger, photoService *service.PhotoService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := lg.With(
			slog.String("op", "handlers.update.UpdatePhoto"),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		jsonMetadata := r.FormValue("metadata")

		var metadata service.PhotoMetadata
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

		photoIdStr := chi.URLParam(r, "photo_uuid")
		if photoIdStr == "" {
			log.Info("photo id param is empty")

			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, response.Error("photo id is empty"))
			return
		}

		photoUuid, err := uuid.Parse(photoIdStr)
		if err != nil {
			log.Error("failed to convert photo id to int", slog.String("photo_id_str", photoIdStr))

			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, response.Error("invalid request"))
			return
		}

		userUuid := r.Context().Value("user_uuid").(uuid.UUID)

		err = photoService.UpdatePhotoInfo(r.Context(), photoUuid, metadata, userUuid)

		if err != nil {
			if errors.Is(err, storage.ErrPhotoNotFound) {
				log.Info("photo not found", slog.Any("photo_uuid", photoUuid))

				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, response.Error("photo not found"))
				return
			}

			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, response.Error("internal error"))
			return
		}

		log.Info("success update photo info", slog.Any("photo_uuid", photoUuid))

		render.Status(r, http.StatusOK)
		responseOk(w, r, photoUuid)
	}
}

func responseOk(w http.ResponseWriter, r *http.Request, photoUuid uuid.UUID) {
	render.JSON(w, r, Response{
		Response:  response.OK(),
		PhotoUuid: photoUuid,
	})
}
