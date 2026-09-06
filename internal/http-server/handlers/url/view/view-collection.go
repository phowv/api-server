package view

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"photo-viewer-server/internal/lib/api/response"
	"photo-viewer-server/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/google/uuid"
)

func ViewCollection(lg *slog.Logger, collectionService *service.CollectionService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := lg.With(
			slog.String("op", "handlers.view.ViewCollection"),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		collectionUuidStr := chi.URLParam(r, "collection_uuid")
		if collectionUuidStr == "" {
			log.Info("collection uiid param is empty")

			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, response.Error("collection uuid is empty"))
			return
		}

		accessKey := r.URL.Query().Get("collection_access_secret")
		r.WithContext(context.WithValue(r.Context(), "collection_access_secret", accessKey))

		collectionUuid, err := uuid.Parse(collectionUuidStr)
		if err != nil {
			log.Error("failed to convert collection uuid str to uuid", slog.String("collection_uuid_str", collectionUuidStr))

			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, response.Error("invalid request"))
			return
		}

		collectionInfo, err := collectionService.GetCollection(r.Context(), collectionUuid)
		if err != nil {
			if errors.Is(err, service.ErrCollectionNotFound) {
				log.Info("collection not found", slog.Any("collection_uuid", collectionUuid))

				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, response.Error("collection not found"))
				return

			} else if errors.Is(err, service.ErrCollectionIsNotPermitted) {
				render.Status(r, http.StatusForbidden)
				render.JSON(w, r, response.Error("collection is not permitted"))
				return
			}

			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, response.Error("internal error"))
			return
		}

		log.Info("photo found", slog.Any("collection_uuid", collectionUuid))
		render.Status(r, http.StatusOK)
		render.JSON(w, r, collectionInfo)
	}
}
