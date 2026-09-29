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

// Remove collection access secret deletes the owner's collection access secret.
//
//	@Summary		Delete a collection access secret
//	@Tags				collections
//	@Produce		json
//	@Security		BearerAuth
//	@Param			collection_uuid	path		string	true	"Collection UUID"
//	@Success		200			{object}	response.Response
//	@Failure		400			{object}	response.Response	"invalid request"
//	@Failure		401			{object}	response.Response	"token is empty / invalid token"
//	@Failure		403			{object}	response.Response	"invalid authorization (not the owner)"
//	@Failure		404			{object}	response.Response	"not found"
//	@Failure		500			{object}	response.Response	"internal error"
//	@Router			/collection/{collection_uuid}/secret [delete]
func RemoveCollectionAccessSecret(lg *slog.Logger, collectionService *service.CollectionService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := lg.With(
			slog.String("op", "handlers.remove.RemoveCollectionAccessSecret"),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		collectionUuidStr := chi.URLParam(r, "collection_uuid")
		if collectionUuidStr == "" {
			log.Info("collection uuid param is empty")

			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, response.Error("photo id is empty"))
			return
		}

		collectionUuid, err := uuid.Parse(collectionUuidStr)
		if err != nil {
			log.Error("failed to convert photo id to int", slog.String("collection_uuid_str", collectionUuidStr))

			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, response.Error("invalid request"))
			return
		}

		userUuid, ok := r.Context().Value("user_uuid").(uuid.UUID)
		if !ok {
			log.Error("invalid user_uuid in ctx", slog.Any("user_uuid", r.Context().Value("user_uuid")))

			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, response.Error("internal error"))
			return
		}

		err = collectionService.DeleteCollectionAccessSecret(r.Context(), collectionUuid, userUuid)

		if err != nil {
			log.Info("failed to generate collection access link", slog.Any("collection_uuid", collectionUuid), sl.Err(err))

			if errors.Is(err, service.ErrCollectionNotFound) {
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, response.Error("collection not found"))
				return

			} else if errors.Is(err, service.ErrAccessLinkNotFound) {
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, response.Error("access link not found"))
				return

			} else if errors.Is(err, service.ErrForbidden) {
				render.Status(r, http.StatusForbidden)
				render.JSON(w, r, response.Error("collection access denied"))
				return
			}

			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, response.Error("internal error"))
			return
		}

		log.Info("success remove photo access link", slog.Any("collection_uuid", collectionUuid))

		render.Status(r, http.StatusOK)
		render.JSON(w, r, response.OK())
	}
}
