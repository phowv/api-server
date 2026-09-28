package upload

import (
	"encoding/json"
	"errors"
	"io"
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

const (
	MaxBodySize  = 10 * 1024 * 1024
	MaxPhotoSize = MaxBodySize - 1024
)

type Response struct {
	response.Response
	PhotoUuid uuid.UUID `json:"photo_uuid"`
}

// UploadPhoto stores a photo with metadata and tags; generates medium/small renditions.
//
//	@Summary		Upload a photo
//	@Description	Total body size limit is 10 MB.
//	@Tags				photos
//	@Accept			multipart/form-data
//	@Produce		json
//	@Security		BearerAuth
//	@Param			metadata	formData	string	true	"JSON: title (≤50), description (≤200), created_at, took_at, access_level (private|protected|public), tag_uuids []"
//	@Param			photo		formData	file	true	"Photo file"
//	@Success		201			{object}	upload.Response
//	@Failure		400			{object}	response.Response	"invalid metadata / validation / invalid photo file / file too big / tag does not exist"
//	@Failure		401			{object}	response.Response	"token is empty / invalid token"
//	@Failure		403			{object}	response.Response	"quota is not enough"
//	@Failure		500			{object}	response.Response	"internal error"
//	@Router			/photos [post]
func UploadPhoto(lg *slog.Logger, validator *vlpkg.Validate, photoService *service.PhotoService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := lg.With(
			slog.String("op", "handlers.upload.UploadPhoto"),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		r.Body = http.MaxBytesReader(w, r.Body, MaxBodySize)

		jsonMetadata := r.FormValue("metadata")

		var metadata service.SavePhotoInputMetadata
		if err := json.Unmarshal([]byte(jsonMetadata), &metadata); err != nil {
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

		file, header, err := r.FormFile("photo")
		if err != nil {
			log.Error("failed to get photo", sl.Err(err))

			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, response.Error("invalid photo file"))
			return
		}
		defer file.Close()

		if header.Size > MaxPhotoSize {
			log.Error("photo is too big", slog.Int64("size", header.Size))

			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, response.Error("photo file is too big"))
			return
		}

		fileBytes, err := io.ReadAll(file)
		if err != nil {
			log.Error("failed to read photo", sl.Err(err))

			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, response.Error("internal error"))
			return
		}

		log.Info("receive photo file", slog.Int64("size", header.Size))

		input := service.SavePhotoInput{
			Metadata:    metadata,
			Filename:    header.Filename,
			Content:     fileBytes,
			ContentType: header.Header.Get("Content-Type"),
		}

		userUuid := r.Context().Value("user_uuid").(uuid.UUID)

		photoUuid, err := photoService.SavePhoto(r.Context(), input, userUuid)
		if err != nil {
			log.Error("failed to save photo", sl.Err(err))

			if errors.Is(err, service.ErrUserQuotaIsNotEnough) {
				render.Status(r, http.StatusForbidden)
				render.JSON(w, r, response.Error("quota is not enough"))
				return

			} else if errors.Is(err, service.ErrTagDoesNotExists) {
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, response.Error("tag does not exists"))
				return
			}

			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, response.Error("internal error"))
			return
		}

		log.Info("saved photo", slog.Any("photo_uuid", photoUuid))

		render.Status(r, http.StatusCreated)
		responseOk(w, r, photoUuid)
	}
}

func responseOk(w http.ResponseWriter, r *http.Request, photoUuid uuid.UUID) {
	render.JSON(w, r, Response{
		Response:  response.OK(),
		PhotoUuid: photoUuid,
	})
}
