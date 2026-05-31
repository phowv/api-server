package update

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"photo-viewer-server/internal/lib/api/response"
	"photo-viewer-server/internal/lib/logger/sl"
	"photo-viewer-server/internal/service"
	"photo-viewer-server/internal/storage"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
)

type Response struct {
	response.Response
  PhotoId int `json:"photo_id"`
}

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

		if err := validator.New().Struct(metadata); err != nil {
			validateErr := err.(validator.ValidationErrors)

			log.Error("error validate request metadata", sl.Err(err))

			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, response.ValidationErrors(validateErr))
			return
		}

		photoIdStr := chi.URLParam(r, "photo_id")
		if photoIdStr == "" {
			log.Info("photo id param is empty")

			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, response.Error("photo id is empty"))
			return
		}

		photoId, err := strconv.Atoi(photoIdStr)
		if err != nil {
			log.Error("failed to convert photo id to int", slog.String("photo_id_str", photoIdStr))

			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, response.Error("invalid request"))
			return
		}

		err = photoService.UpdatePhotoInfo(r.Context(), photoId, metadata)

		if err != nil {
			if errors.Is(err, storage.ErrPhotoNotFound) {
				log.Info("photo not found", slog.Int("photo_id", photoId))

				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, response.Error("photo not found"))
				return
			}

			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, response.Error("internal error"))
			return
		}

		log.Info("success update photo info", slog.Int("photo_id", photoId))

		render.Status(r, http.StatusOK)
		responseOk(w, r, photoId)
	}
}

func responseOk(w http.ResponseWriter, r *http.Request, photoId int) {
	render.JSON(w, r, Response{
		Response: response.OK(),
		PhotoId: photoId,
	})
}
