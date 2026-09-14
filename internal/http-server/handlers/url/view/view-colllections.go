package view

import (
	"errors"
	"log/slog"
	"net/http"
	"photo-viewer-server/internal/lib/api/response"
	"photo-viewer-server/internal/lib/logger/sl"
	"photo-viewer-server/internal/service"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

func ViewCollections(lg *slog.Logger, collectionService *service.CollectionService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := lg.With(
			slog.String("op", "handlers.view.ViewPhotos"),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		ownerLogin := r.URL.Query().Get("owner_login")

		collections, err := collectionService.GetCollections(r.Context(), ownerLogin)
		if err != nil {
			if errors.Is(err, service.ErrUserNotFound) {
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, response.Error("owner not found"))
				return
			}

			log.Error("error get photos", sl.Err(err))
			
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, response.Error("internal error"))
			return
		}

		log.Info("success get photos", slog.Int("length", len(collections)))

		render.JSON(w, r, collections)
	}
}
