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

// ViewPhotos lists photos. Anonymous and non-owner requests forbidden. For authorized user get all own photos
//
//	@Summary		List self photos
//	@Tags				photos
//	@Produce		json
//	@Success		200			{object}	[]service.PhotoInfo
//	@Failure		500			{object}	response.Response	"internal error"
//	@Router			/me/photos [get]
func ViewSelfPhotos(lg *slog.Logger, photoService *service.PhotoService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := lg.With(
			slog.String("op", "handlers.view.ViewSelfPhotos"),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		photos, err := photoService.GetSelfPhotos(r.Context())
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
