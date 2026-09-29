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

type PhotoAccessSecretResponse struct {
	response.Response
	service.PhotoAccessLinkSecret
}

// Create photo access secret for grant access to protected photos.
//
//	@Summary		Grant photo access
//	@Tags				photos
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			photo_uuid	path		string	true	"Photo UUID"
//	@Param			body		body	service.AccessLinkRequest	true	"expires_duration (ns duration)"
//	@Success		200			{object}	update.Response
//	@Failure		400			{object}	response.Response	"invalid metadata / validation / invalid request"
//	@Failure		401			{object}	response.Response	"token is empty / invalid token"
//	@Failure		403			{object}	response.Response	"access link already exists"
//	@Failure		404			{object}	response.Response	"photo not found"
//	@Failure		500			{object}	response.Response	"internal error"
//	@Router			/photo/{photo_uuid}/secret [post]
func CreatePhotoAccessSecret(lg *slog.Logger, photoService *service.PhotoService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := lg.With(
			slog.String("op", "handlers.create.CreatePhotoAccessLink"),
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

		userUuid := r.Context().Value("user_uuid").(uuid.UUID)

		secret, err := photoService.GeneratePhotoAccessSecret(r.Context(), &metadata, photoUuid, userUuid)

		if err != nil {
			log.Info("failed to generate photo access link", slog.Any("photo_uuid", photoUuid), sl.Err(err))

			if errors.Is(err, service.ErrPhotoNotFound) {
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, response.Error("photo not found"))
				return

			} else if errors.Is(err, service.ErrAccessLinkAlreadyExists) {
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, response.Error("photo access link already exists"))
				return

			} else if errors.Is(err, service.ErrForbidden) {
				render.Status(r, http.StatusForbidden)
				render.JSON(w, r, response.Error("photo access denied"))
				return
			}

			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, response.Error("internal error"))
			return
		}

		log.Info("success create photo access link", slog.Any("photo_uuid", photoUuid))

		render.Status(r, http.StatusOK)
		render.JSON(w, r, PhotoAccessSecretResponse{
			Response:              response.OK(),
			PhotoAccessLinkSecret: *secret,
		})
	}
}
