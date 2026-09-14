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
	"github.com/google/uuid"
)

func ViewTags(lg *slog.Logger, tagService *service.TagService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := lg.With(
			slog.String("op", "handlers.view.ViewTags"),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		photoUuidStr := r.URL.Query().Get("photo_uuid")

		photoUuid := uuid.Nil
		var err error

		if photoUuidStr != "" {
			photoUuid, err = uuid.Parse(photoUuidStr)

			if err != nil {
				log.Error("error parse photo uuid")

				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, response.Error("invalid photo uuid"))
				return
			}
		}

		tags, err := tagService.GetTags(r.Context(), photoUuid)

		if err != nil {
			log.Error("error get tags", sl.Err(err))

			if errors.Is(err, service.ErrPhotoNotFound) {				
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, response.Error("photo not found"))
				return
			}

			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, response.Error("internal error"))
			return
		}

		render.Status(r, http.StatusOK)
		render.JSON(w, r, tags)
	}
}
