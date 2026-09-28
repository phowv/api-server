package remove

import (
	"errors"
	"log/slog"
	"net/http"
	"photo-viewer-server/internal/lib/api/response"
	"photo-viewer-server/internal/lib/logger/sl"
	"photo-viewer-server/internal/service"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/google/uuid"
)

// RemoveCollection deletes the owner's collection (photo files are not deleted).
//
//	@Summary		Delete a collection
//	@Tags				collections
//	@Produce		json
//	@Security		BearerAuth
//	@Param			collection_uuid	path		string	true	"Collection UUID"
//	@Success		200			{object}	response.Response
//	@Failure		400			{object}	response.Response	"invalid request"
//	@Failure		401			{object}	response.Response	"token is empty / invalid token"
//	@Failure		403			{object}	response.Response	"not permitted (not the owner)"
//	@Failure		404			{object}	response.Response	"not found"
//	@Failure		500			{object}	response.Response	"internal error"
//	@Router			/collection/{collection_uuid} [delete]
func RemoveCollection(lg *slog.Logger, collectionService *service.CollectionService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := lg.With(
			slog.String("op", "handlers.remove.RemoveCollection"),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		collectionUuidStr := chi.URLParam(r, "collection_uuid")
		if collectionUuidStr == "" {
			log.Info("collection uuid param is empty")

			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, response.Error("collection uuid is empty"))
			return
		}

		collectionUuid, err := uuid.Parse(collectionUuidStr)
		if err != nil {
			log.Error("failed to convert collection uuid str to uuid", slog.String("collection_uuid_str", collectionUuidStr))

			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, response.Error("invalid request"))
			return
		}

		err = collectionService.DeleteCollection(r.Context(), collectionUuid)
		if err != nil {
			log.Error("error remove collection", sl.Err(err))

			if errors.Is(err, service.ErrCollectionNotFound) {
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, response.Error("not found"))
				return

			} else if errors.Is(err, service.ErrCollectionActionIsNotPermitted) {
				render.Status(r, http.StatusForbidden)
				render.JSON(w, r, response.Error("not permitted"))
				return
			}

			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, response.Error("internal error"))
			return
		}

		render.JSON(w, r, response.OK())
	}
}
