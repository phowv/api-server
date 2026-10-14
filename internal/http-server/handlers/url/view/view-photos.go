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

// ViewPhotos lists photos. Anonymous and non-owner requests get only public photos; the owner gets all of theirs.
//
//	@Summary		List photos
//	@Tags				photos
//	@Produce		json
//	@Param			owner_login	query		string	false	"Owner login; without the param — all public photos"
//	@Success		200			{object}	[]service.PhotoInfo
//	@Failure		404			{object}	response.Response	"owner not found"
//	@Failure		500			{object}	response.Response	"internal error"
//	@Router			/photos [get]
func ViewPhotos(lg *slog.Logger, photoService *service.PhotoService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := lg.With(
			slog.String("op", "handlers.view.ViewPhotos"),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		ownerLogin := r.URL.Query().Get("owner_login")

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
