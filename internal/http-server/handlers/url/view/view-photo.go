package view

import (
	"errors"
	"log/slog"
	"net/http"
	"photo-viewer-server/internal/lib/api/response"
	"photo-viewer-server/internal/lib/logger/sl"
	"photo-viewer-server/internal/service"
	"photo-viewer-server/internal/storage"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/google/uuid"
)

func ViewPhoto(lg *slog.Logger, photoService *service.PhotoService, isSmall bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := lg.With(
			slog.String("op", "handlers.view.ViewPhotos"),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

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

		rawPhoto, err := photoService.GetPhoto(r.Context(), photoUuid, isSmall)
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

		log.Info("photo found", slog.Any("photo_uuid", photoUuid))

		w.Header().Set("Content-Type", "image/jpeg")
		w.WriteHeader(http.StatusOK)
		_, err = w.Write(rawPhoto.Content)
		if err != nil {
			log.Error("failed to write photo content", sl.Err(err))
		}
	}
}
