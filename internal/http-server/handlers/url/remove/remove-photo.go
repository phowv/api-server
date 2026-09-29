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

// RemovePhoto deletes the owner's photo and its files.
//
//	@Summary		Delete a photo
//	@Tags				photos
//	@Produce		json
//	@Security		BearerAuth
//	@Param			photo_uuid	path		string	true	"Photo UUID"
//	@Success		200			{object}	response.Response
//	@Failure		400			{object}	response.Response	"invalid request"
//	@Failure		401			{object}	response.Response	"token is empty / invalid token"
//	@Failure		403			{object}	response.Response	"invalid authorization (not the owner)"
//	@Failure		404			{object}	response.Response	"not found"
//	@Failure		500			{object}	response.Response	"internal error"
//	@Router			/photo/{photo_uuid} [delete]
func RemovePhoto(lg *slog.Logger, photoService *service.PhotoService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := lg.With(
			slog.String("op", "handlers.remove.RemovePhoto"),
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

		userUuid, ok := r.Context().Value("user_uuid").(uuid.UUID)
		if !ok {
			log.Error("invalid user_uuid in ctx", slog.Any("user_uuid", r.Context().Value("user_uuid")))

			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, response.Error("internal error"))
			return
		}

		err = photoService.DeletePhoto(r.Context(), photoUuid, userUuid)
		if err != nil {
			log.Error("error remove photo", sl.Err(err))

			if errors.Is(err, service.ErrPhotoNotFound) {
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, response.Error("not found"))
				return

			} else if errors.Is(err, service.ErrUserInvalidAuthorization) {
				render.Status(r, http.StatusForbidden)
				render.JSON(w, r, response.Error("invalid authorization"))
				return
			}

			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, response.Error("internal error"))
			return
		}

		render.JSON(w, r, response.OK())
	}
}
