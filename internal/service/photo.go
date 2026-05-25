package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"photo-viewer-server/internal/lib/logger/sl"
	"photo-viewer-server/internal/storage"
	"photo-viewer-server/internal/storage/entity"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
)

type PhotoMetadata struct {
	Title string `json:"title" validate:"required"`
	Description string `json:"description"`
}

type PhotoInfo struct {
	PhotoId int `json:"photo_id"`
	PhotoMetadata
}

type PhotoWithData struct {
	PhotoInfo
	Content []byte
}

type PhotoRepo interface {
	SavePhoto(ctx context.Context, photo *entity.Photo) (int, error)
	GetAllPhotos(ctx context.Context) ([]entity.Photo, error)
	GetPhoto(ctx context.Context, id int) (*entity.Photo, error)
	DeletePhoto(ctx context.Context, id int) error
}

type FileRepo interface {
	SaveFile(ctx context.Context, bucketName string, objectName string, data []byte, contentType string) (string, error)
	GetFile(ctx context.Context, bucketName string, objectName string) ([]byte, error)
	DeleteFile(ctx context.Context, bucketName string, objectName string) error
}

type PhotoService struct {
	log *slog.Logger
	photoRepo PhotoRepo
	fileRepo FileRepo
	bucketName string
}

func NewPhotoService(log *slog.Logger, photoRepo PhotoRepo, fileRepo FileRepo, bucketName string) *PhotoService {
	return &PhotoService{
		log: log,
		photoRepo: photoRepo,
		fileRepo: fileRepo,
		bucketName: bucketName,
	}
}

type SavePhotoInput struct {
	Metadata PhotoMetadata
	Filename string
	Content []byte
	ContentType string
}

func (s *PhotoService) SavePhoto(ctx context.Context, input SavePhotoInput) (int, error) {
	log := s.log.With(
		slog.String("op", "service.SavePhoto"),
		slog.String("request_id", middleware.GetReqID(ctx)),
	)

	ext := filepath.Ext(input.Filename)
	newFilename := uuid.NewString() + ext

	filename, err := s.fileRepo.SaveFile(ctx, s.bucketName, newFilename, input.Content, input.ContentType)

	if err != nil {
		log.Error("failed to save photo file", sl.Err(err))

		return 0, fmt.Errorf("failed to save photo file: %w", err)
	}

	log.Info("saved photo", slog.String("filename", filename))

	photoEntity := entity.Photo{
		Title: input.Metadata.Title,
		Description: input.Metadata.Description,
		Filename: filename,
	}

	photoId, err := s.photoRepo.SavePhoto(ctx, &photoEntity)

	if err != nil {
		log.Error("error save photo metadata", sl.Err(err))
		return 0, fmt.Errorf("error save photo metadata: %w", err)
	}

	return photoId, nil
}

func (s *PhotoService) GetPhotos(ctx context.Context) ([]PhotoInfo, error) {
	log := s.log.With(
		slog.String("op", "service.GetPhotos"),
		slog.String("request_id", middleware.GetReqID(ctx)),
	)

	photoEnities, err := s.photoRepo.GetAllPhotos(ctx)
	if err != nil {
		log.Error("failed to get all photos", sl.Err(err))

		return nil, fmt.Errorf("failed to get all photos: %w", err)
	}

	photos := make([]PhotoInfo, len(photoEnities))

	for i, photoEntity := range photoEnities {
		photos[i] = PhotoInfo{
			PhotoId: photoEntity.PhotoId,
			PhotoMetadata: PhotoMetadata{
				Title: photoEntity.Title,
				Description: photoEntity.Description,
			},
		}
	}

	return photos, nil
}

func (s *PhotoService) GetPhoto(ctx context.Context, photoId int) (*PhotoWithData, error) {
	log := s.log.With(
		slog.String("op", "service.GetPhoto"),
		slog.String("request_id", middleware.GetReqID(ctx)),
	)

	photoEntity, err := s.photoRepo.GetPhoto(ctx, photoId)
	if err != nil {
		if errors.Is(err, storage.ErrPhotoNotFound) {
			log.Error("photo not found", slog.Int("photo_id", photoId))
			return nil, err
		}

		log.Error("error get photo", sl.Err(err))
		return nil, fmt.Errorf("error get photo: %w", err)
	}

	rawPhoto, err := s.fileRepo.GetFile(ctx, s.bucketName, photoEntity.Filename)
	if err != nil {
		log.Error("error get photo file", sl.Err(err), slog.String("filename", photoEntity.Filename))
		return nil, fmt.Errorf("error get photo file: %w", err)
	}

	photoWithData := &PhotoWithData{
		Content: rawPhoto,
		PhotoInfo: PhotoInfo{
			PhotoId: photoEntity.PhotoId,
			PhotoMetadata: PhotoMetadata{
				Title: photoEntity.Title,
				Description: photoEntity.Description,
			},
		},
	}

	return photoWithData, nil
}

func (s *PhotoService) DeletePhoto(ctx context.Context, photoId int) error {
	log := s.log.With(
		slog.String("op", "service.DeletePhoto"),
		slog.String("request_id", middleware.GetReqID(ctx)),
	)

	photoEntity, err := s.photoRepo.GetPhoto(ctx, photoId)
	if err != nil {
		if errors.Is(err, storage.ErrPhotoNotFound) {
			log.Error("photo not found", slog.Int("photo_id", photoId))
			return err
		}

		log.Error("error get photo", sl.Err(err))
		return fmt.Errorf("error get photo: %w", err)
	}

	err = s.photoRepo.DeletePhoto(ctx, photoId)
	if err != nil {
		log.Error("failed to delete photo", sl.Err(err))
		return err
	}

	err = s.fileRepo.DeleteFile(ctx, s.bucketName, photoEntity.Filename)
	if err != nil {
		log.Error("failed to delete photo file", sl.Err(err))
		return err
	}

	return nil
}
