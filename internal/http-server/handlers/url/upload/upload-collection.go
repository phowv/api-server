package upload

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"photo-viewer-server/internal/lib/api/response"
	"photo-viewer-server/internal/lib/logger/sl"
	"photo-viewer-server/internal/lib/validatorx"
	"photo-viewer-server/internal/service"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

type CollecionCreateResponse struct {
	response.Response
	CollectionUuid uuid.UUID `json:"collection_uuid"`
}

func UploadCollection(lg *slog.Logger, collectionService *service.CollectionService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {	
		log := lg.With(
			slog.String("op", "handlers.upload.UploadCollection"),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		jsonMetadata := r.FormValue("metadata")

		var metadata service.SaveCollectionInputMetadata
		if err := json.Unmarshal([]byte(jsonMetadata), &metadata); err != nil {
			log.Error("failed to decode metadata", sl.Err(err))

			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, response.Error("invalid metadata"))
			return
		}

		log.Info("request metadata decoded", slog.Any("metadata", metadata))

		if err := validatorx.NewValidator().Struct(metadata); err != nil {
			validateErr := err.(validator.ValidationErrors)

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
			Response: response.OK(),
			CollectionUuid: collectionUuid,
		})
	}
}
