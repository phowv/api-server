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

// RemoveTag deletes the tag (admin only).
//
//	@Summary		Delete a tag
//	@Tags				tags
//	@Produce		json
//	@Security		BearerAuth
//	@Param			tag_uuid	path		string	true	"Tag UUID"
//	@Success		200			{object}	response.Response
//	@Failure		400			{object}	response.Response	"invalid request"
//	@Failure		401			{object}	response.Response	"token is empty / invalid token"
//	@Failure		404			{object}	response.Response	"not found"
//	@Failure		500			{object}	response.Response	"internal error"
//	@Router			/tags/{photo_uuid} [delete]
func RemoveTag(lg *slog.Logger, tagService *service.TagService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := lg.With(
			slog.String("op", "handlers.remove.RemovePhoto"),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		tagUuidStr := chi.URLParam(r, "tag_uuid")
		if tagUuidStr == "" {
			log.Info("tag id param is empty")

			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, response.Error("tag uuid is empty"))
			return
		}

		tagUuid, err := uuid.Parse(tagUuidStr)
		if err != nil {
			log.Error("failed to convert tag uuid str to uuid", slog.String("tag_uuid_str", tagUuidStr))

			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, response.Error("invalid request"))
			return
		}

		err = tagService.DeleteTag(r.Context(), tagUuid)
		if err != nil {
			log.Error("error remove tag", sl.Err(err))

			if errors.Is(err, service.ErrTagDoesNotExists) {
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, response.Error("not found"))
				return
			}

			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, response.Error("internal error"))
			return
		}

		render.JSON(w, r, response.OK())
	}
}

