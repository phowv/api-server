package create

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

type CollectionAccessSecretResponse struct {
	response.Response
	service.CollectionAccessLinkSecret
}

// Create collection access secret for grant access to protected collections.
//
//	@Summary		Grant collection access
//	@Tags				collections
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			collection_uuid	path		string	true	"Collection UUID"
//	@Param			body		body	service.AccessLinkRequest	true	"expires_duration (ns duration)"
//	@Success		200			{object}	create.CollectionAccessSecretResponse
//	@Failure		400			{object}	response.Response	"invalid metadata / validation / invalid request"
//	@Failure		401			{object}	response.Response	"token is empty / invalid token"
//	@Failure		403			{object}	response.Response	"access link already exists"
//	@Failure		404			{object}	response.Response	"photo not found"
//	@Failure		500			{object}	response.Response	"internal error"
//	@Router			/collection/{collection_uuid}/secret [post]
func CreateCollectionAccessSecret(lg *slog.Logger, collectionService *service.CollectionService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := lg.With(
			slog.String("op", "handlers.create.CreateCollectionAccessSecret"),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		var metadata service.AccessLinkRequest
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

		userUuid := r.Context().Value("user_uuid").(uuid.UUID)

		secret, err := collectionService.GenerateCollectionAccessSecret(r.Context(), &metadata, collectionUuid, userUuid)

		if err != nil {
			log.Info("failed to generate collection access link", slog.Any("collection_uuid", collectionUuid), sl.Err(err))

			if errors.Is(err, service.ErrCollectionNotFound) {
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, response.Error("collection not found"))
				return

			} else if errors.Is(err, service.ErrAccessLinkAlreadyExists) {
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, response.Error("collection access link already exists"))
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

		log.Info("success create collection access link", slog.Any("collection_uuid", collectionUuid))

		render.Status(r, http.StatusOK)
		render.JSON(w, r, CollectionAccessSecretResponse{
			Response:                   response.OK(),
			CollectionAccessLinkSecret: *secret,
		})
	}
}
