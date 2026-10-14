package upload

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"photo-viewer-server/internal/lib/api/response"
	"photo-viewer-server/internal/lib/logger/sl"
	"photo-viewer-server/internal/service"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	vlpkg "github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

type CollecionCreateResponse struct {
	response.Response
	CollectionUuid uuid.UUID `json:"collection_uuid"`
}

// UploadCollection creates a collection, optionally prefilling it with photos.
//
//	@Summary		Create a collection
//	@Tags				collections
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body		body	service.SaveCollectionInputMetadata	true	"title (≤50), description (≤200), access_level (private|protected|public), photo_uuids []"
//	@Success		201			{object}	upload.CollecionCreateResponse
//	@Failure		400			{object}	response.Response	"invalid metadata / validation"
//	@Failure		401			{object}	response.Response	"token is empty / invalid token"
//	@Failure		403			{object}	response.Response	"quota is not enough / collection already exists"
//	@Failure		500			{object}	response.Response	"internal error (incl. unknown photo in photo_uuids)"
//	@Router			/collections [post]
func UploadCollection(lg *slog.Logger, validator *vlpkg.Validate, collectionService *service.CollectionService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := lg.With(
			slog.String("op", "handlers.upload.UploadCollection"),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		var metadata service.SaveCollectionInputMetadata
		if err := json.NewDecoder(r.Body).Decode(&metadata); err != nil {
			log.Error("failed to decode metadata", sl.Err(err))

			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, response.Error("invalid metadata"))
			return
		}

		log.Info("request metadata decoded", slog.Any("metadata", metadata))

		if err := validator.Struct(metadata); err != nil {
			validateErr, ok := err.(vlpkg.ValidationErrors)
			if !ok {
				log.Error("unknown validation error", sl.Err(err))

				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, response.Error("validation error"))
				return
			}

			log.Error("error validate request metadata", sl.Err(err))

			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, response.ValidationErrors(validateErr))
			return
		}

		userUuid := r.Context().Value("user_uuid").(uuid.UUID)

		collectionUuid, err := collectionService.SaveCollection(r.Context(), &metadata, userUuid)
		if err != nil {
			log.Error("failed to save photo", sl.Err(err))

			if errors.Is(err, service.ErrUserQuotaIsNotEnough) {
				render.Status(r, http.StatusForbidden)
				render.JSON(w, r, response.Error("quota is not enough"))
				return
			} else if errors.Is(err, service.ErrCollectionAlreadyExists) {
				render.Status(r, http.StatusForbidden)
				render.JSON(w, r, response.Error("collection already exists"))
				return
			}

			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, response.Error("internal error"))
			return
		}

		log.Info("saved collection", slog.Any("collection_uuid", collectionUuid))

		render.Status(r, http.StatusCreated)
		render.JSON(w, r, CollecionCreateResponse{
			Response:       response.OK(),
			CollectionUuid: collectionUuid,
		})
	}
}
