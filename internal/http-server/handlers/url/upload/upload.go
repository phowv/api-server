package upload

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"photo-viewer-server/internal/lib/api/response"
	"photo-viewer-server/internal/lib/logger/sl"
	"photo-viewer-server/internal/service"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

const (
	MaxBodySize = 10 * 1024 * 1024
	MaxPhotoSize = MaxBodySize - 1024
)

type Response struct {
	response.Response
  PhotoUuid uuid.UUID `json:"photo_uuid"`
}

func UploadPhoto(lg *slog.Logger, photoService *service.PhotoService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := lg.With(
			slog.String("op", "handlers.upload.UploadPhoto"),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		r.Body = http.MaxBytesReader(w, r.Body, MaxBodySize)

		jsonMetadata := r.FormValue("metadata")

		var metadata service.PhotoMetadata
		if err := json.Unmarshal([]byte(jsonMetadata), &metadata); err != nil {
			log.Error("failed to decode metadata", sl.Err(err))

			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, response.Error("invalid metadata"))
			return
		}

		log.Info("request metadata decoded", slog.Any("metadata", metadata))

		if err := validator.New().Struct(metadata); err != nil {
			validateErr := err.(validator.ValidationErrors)

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
			Metadata: metadata,
			Filename: header.Filename,
			Content: fileBytes,
			ContentType: header.Header.Get("Content-Type"),
		}

		userUuid := r.Context().Value("user_uuid").(uuid.UUID)

		photoUuid, err := photoService.SavePhoto(r.Context(), input, userUuid)
		if err != nil {
			log.Error("failed to save photo", sl.Err(err))

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
		Response: response.OK(),
		PhotoUuid: photoUuid,
	})
}
