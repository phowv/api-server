package update

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"photo-viewer-server/internal/lib/api/response"
	"photo-viewer-server/internal/lib/logger/sl"
	"photo-viewer-server/internal/lib/validatorx"
	"photo-viewer-server/internal/service"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

type AddPhotoToCollectionResponse struct {
	response.Response
	CollectionUuid uuid.UUID `json:"collection_uuid"`
	PhotoUuid      uuid.UUID `json:"photo_uuid"`
}

type AddPhotoToCollectionRequest struct {
	PhotoUuid      uuid.UUID `json:"photo_uuid" validate:"required"`
}

// AddPhotoToCollection adds a photo to a collection.
//
//	@Summary		Add photo to collection
//	@Tags				collections
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			collection_uuid	path		string	true	"Collection UUID"
//	@Param			body		body	AddPhotoToCollectionRequest	true	"Photo UUID"
//	@Success		201			{object}	update.AddPhotoToCollectionResponse
//	@Failure		400			{object}	response.Response	"invalid request / photo in collection already exists"
//	@Failure		401			{object}	response.Response	"token is empty / invalid token"
//	@Failure		403			{object}	response.Response	"add to collection is not permitted"
//	@Failure		404			{object}	response.Response	"collection not found / photo not found"
//	@Failure		500			{object}	response.Response	"internal error"
//	@Router			/collection/{collection_uuid}/photos [post]
func AddPhotoToCollection(lg *slog.Logger, collectionService *service.CollectionService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := lg.With(
			slog.String("op", "handlers.update.AddPhotoToCollection"),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		collectionUuidStr := chi.URLParam(r, "collection_uuid")
		if collectionUuidStr == "" {
			log.Error("collection id param is empty")

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

		var metadata AddPhotoToCollectionRequest

	  if err := json.NewDecoder(r.Body).Decode(&metadata); err != nil {
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

		err = collectionService.AddPhotoToCollection(r.Context(), collectionUuid, metadata.PhotoUuid)
		if err != nil {
			renderServiceError(w, r, err)
			return
		}

		log.Info("success add photo to collection", slog.Any("photo_uuid", metadata.PhotoUuid), slog.Any("collection_uuid", collectionUuid))

		render.Status(r, http.StatusCreated)
		render.JSON(w, r, &AddPhotoToCollectionResponse{
			Response:       response.OK(),
			CollectionUuid: collectionUuid,
			PhotoUuid:      metadata.PhotoUuid,
		})
	}
}

// RemovePhotoFromCollection removes a photo from a collection.
//
//	@Summary		Remove photo from collection
//	@Tags				collections
//	@Produce		json
//	@Security		BearerAuth
//	@Param			collection_uuid	path		string	true	"Collection UUID"
//	@Param			photo_uuid		path		string	true	"Photo UUID"
//	@Success		200			{object}	response.Response
//	@Failure		400			{object}	response.Response	"invalid request"
//	@Failure		401			{object}	response.Response	"token is empty / invalid token"
//	@Failure		403			{object}	response.Response	"remove from collection is not permitted"
//	@Failure		404			{object}	response.Response	"collection not found / photo not found"
//	@Failure		500			{object}	response.Response	"internal error"
//	@Router			/collection/{collection_uuid}/photo/{photo_uuid} [delete]
func RemovePhotoFromCollection(lg *slog.Logger, collectionService *service.CollectionService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := lg.With(
			slog.String("op", "handlers.update.RemovePhotoFromCollection"),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		collectionUuidStr := chi.URLParam(r, "collection_uuid")
		if collectionUuidStr == "" {
			log.Error("collection id param is empty")

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

		photoUuidStr := chi.URLParam(r, "photo_uuid")

		if photoUuidStr == "" {
			log.Error("photo param is empty")

			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, response.Error("photo uuid is empty"))
			return
		}

		photoUuid, err := uuid.Parse(photoUuidStr)
		if err != nil {
			log.Error("failed to convert photo uuid str to uuid", slog.String("photo_uuid_str", photoUuidStr))

			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, response.Error("invalid request"))
			return
		}

		err = collectionService.RemovePhotoFromCollection(r.Context(), collectionUuid, photoUuid)
		if err != nil {
			renderServiceError(w, r, err)
			return
		}

		log.Info("success remove photo from collection", slog.Any("photo_uuid", photoUuid), slog.Any("collection_uuid", collectionUuid))

		render.Status(r, http.StatusOK)
		render.JSON(w, r, response.OK())
	}
}

func renderServiceError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, service.ErrCollectionNotFound):
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, response.Error("collection not found"))
	case errors.Is(err, service.ErrPhotoNotFound):
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, response.Error("photo not found"))
	case errors.Is(err, service.ErrCollectionActionIsNotPermitted):
		render.Status(r, http.StatusForbidden)
		render.JSON(w, r, response.Error("add to collection is not permitted"))
	case errors.Is(err, service.ErrPhotoInCollectionAlreadyExists):
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, response.Error("photo in collection already exists"))
	case errors.Is(err, service.ErrCollectionActionIsNotPermitted):
		render.Status(r, http.StatusForbidden)
		render.JSON(w, r, response.Error("remove from collection is not permitted"))
	default:
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, response.Error("internal error"))
	}
}
