package view

import (
	"errors"
	"log/slog"
	"net/http"
	"photo-viewer-server/internal/lib/api/response"
	"photo-viewer-server/internal/lib/logger/sl"
	"photo-viewer-server/internal/service"
	"photo-viewer-server/internal/storage"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

func ViewPhotos(lg *slog.Logger, photoService *service.PhotoService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := lg.With(
			slog.String("op", "handlers.view.ViewPhotos"),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		ownerLogin := r.URL.Query().Get("owner_login")

		if ownerLogin == "" {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, response.Error("owner login must be not empty"))
			return
		}

		photos, err := photoService.GetPhotos(r.Context(), ownerLogin)
		if err != nil {
			if errors.Is(err, storage.ErrUserNotFound) {
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, response.Error("owner not found"))
				return
			}

			log.Error("error get photos", sl.Err(err))
			
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, response.Error("internal error"))
			return
		}

		log.Info("success get photos", slog.Int("length", len(photos)))

		render.JSON(w, r, photos)
	}
}
