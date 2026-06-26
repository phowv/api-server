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
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
)

type PhotoMetadata struct {
	Title string `json:"title"`
	Description string `json:"description"`
	Tags string `json:"tags"`
	CreatedAt time.Time `json:"created_at"`
	TookAt time.Time `json:"took_at"`
}

type PhotoInfo struct {
	PhotoUuid uuid.UUID `json:"photo_uuid"`
	OwnerLogin string `json:"owner_login"`
	PhotoMetadata
}

type PhotoWithData struct {
	PhotoInfo
	Content []byte
}

type PhotoRepo interface {
	SavePhoto(ctx context.Context, photo *entity.Photo) (uuid.UUID, error)
	GetAllPhotos(ctx context.Context) ([]entity.Photo, error)
  GetAllPhotosByOwner(ctx context.Context, ownerUuid uuid.UUID) ([]entity.Photo, error)
	GetPhoto(ctx context.Context, uuid uuid.UUID) (*entity.Photo, error)
	DeletePhoto(ctx context.Context, uuid uuid.UUID, ownerUuid uuid.UUID) error
  UpdatePhoto(ctx context.Context, uuid uuid.UUID, ownerUuid uuid.UUID, fields map[string]any) error
}

type FileRepo interface {
	SaveFile(ctx context.Context, bucketName string, objectName string, data []byte, contentType string) (string, error)
	GetFile(ctx context.Context, bucketName string, objectName string) ([]byte, error)
	DeleteFile(ctx context.Context, bucketName string, objectName string) error
}

type ImageProcessor interface {
  ResizeAndCompress(ctx context.Context, rawImage []byte, maxWidth, maxHeight int, quality int) ([]byte, error)
}

type PhotoService struct {
	log *slog.Logger
	photoRepo PhotoRepo
	fileRepo FileRepo
	bucketName string
	userRepo UserRepo
	imageProcessor ImageProcessor
}

func NewPhotoService(log *slog.Logger, photoRepo PhotoRepo, fileRepo FileRepo, bucketName string, userRepo UserRepo, imageProcessor ImageProcessor) *PhotoService {
	return &PhotoService{
		log: log,
		photoRepo: photoRepo,
		fileRepo: fileRepo,
		bucketName: bucketName,
		userRepo: userRepo,
		imageProcessor: imageProcessor,
	}
}

type SavePhotoInput struct {
	Metadata PhotoMetadata
	Filename string
	Content []byte
	ContentType string
}

func metadataToMap(m *PhotoMetadata) map[string]interface{} {
	if m == nil {
		return nil
	}
	out := make(map[string]interface{})

	if m.Title != "" {
		out["title"] = m.Title
	}

	if m.Description != "" {
		out["description"] = m.Description
	}

	if m.Tags != "" {
		out["tags"] = m.Tags
	}
	
	if !m.CreatedAt.IsZero() {
		out["created_at"] = m.CreatedAt
	}

	if !m.TookAt.IsZero() {
		out["took_at"] = m.TookAt
	}

	return out
}

func (s *PhotoService) SavePhoto(ctx context.Context, input SavePhotoInput, ownerUuid uuid.UUID) (uuid.UUID, error) {
	log := s.log.With(
		slog.String("op", "service.SavePhoto"),
		slog.String("request_id", middleware.GetReqID(ctx)),
	)

	ext := filepath.Ext(input.Filename)
	newPhotoUuid := uuid.NewString()
	newRawFilename := newPhotoUuid + ext
	newSmallFilename := newPhotoUuid + "_small" + ext

	originalFileData := input.Content

	smallFileData, err := s.imageProcessor.ResizeAndCompress(ctx, originalFileData, 300, 300, 70)

	if err != nil {
		log.Error("failed to resize and compress image", sl.Err(err))
	}

	rawFilename, err := s.fileRepo.SaveFile(ctx, ownerUuid.String(), newRawFilename, originalFileData, input.ContentType)

	if err != nil {
		log.Error("failed to save raw photo file", sl.Err(err))

		return uuid.Nil, fmt.Errorf("failed to save photo file: %w", err)
	}

	smallFilename, err := s.fileRepo.SaveFile(ctx, ownerUuid.String(), newSmallFilename, smallFileData, input.ContentType)

	if err != nil {
		log.Error("failed to save small photo file", sl.Err(err))

		return uuid.Nil, fmt.Errorf("failed to save photo file: %w", err)
	}

	log.Info("saved photo", slog.String("filename", rawFilename), slog.String("small_filename", smallFilename))

	photoEntity := entity.Photo{
		Title: input.Metadata.Title,
		Description: input.Metadata.Description,
		Tags: input.Metadata.Tags,
		CreatedDate: input.Metadata.CreatedAt,
		TookAt: input.Metadata.TookAt,
		RawFilename: rawFilename,
		SmallFilename: smallFilename,
		OwnerUuid: ownerUuid,
	}

	photoUuid, err := s.photoRepo.SavePhoto(ctx, &photoEntity)

	if err != nil {
		log.Error("error save photo metadata", sl.Err(err))
		return uuid.Nil, fmt.Errorf("error save photo metadata: %w", err)
	}

	return photoUuid, nil
}

func (s *PhotoService) GetPhotos(ctx context.Context, ownerLogin string) ([]PhotoInfo, error) {
	log := s.log.With(
		slog.String("op", "service.GetPhotos"),
		slog.String("request_id", middleware.GetReqID(ctx)),
	)

	var photoEnities []entity.Photo
	var err error

	if ownerLogin == "" {
		photoEnities, err = s.photoRepo.GetAllPhotos(ctx)

	} else {
		user, err := s.userRepo.GetUserByLogin(ctx, ownerLogin)
		if err != nil {
			if errors.Is(err, storage.ErrUserNotFound) {
				log.Error("owner not found", sl.Err(err))

				return nil, err
			}
			log.Error("failed to get owner for photos", sl.Err(err))

			return nil, fmt.Errorf("failed to get all photos: %w", err)
		}

		photoEnities, err = s.photoRepo.GetAllPhotosByOwner(ctx, user.UserUuid)
	}

	if err != nil {
		log.Error("failed to get all photos", sl.Err(err))

		return nil, fmt.Errorf("failed to get all photos: %w", err)
	}

	photos := make([]PhotoInfo, len(photoEnities))

	for i, photoEntity := range photoEnities {
		user, err := s.userRepo.GetUserByUuid(ctx, photoEntity.OwnerUuid)
		if err != nil {
			log.Error("failed to get photo owner", slog.Any("photo_uuid", photoEntity.PhotoUuid), slog.Any("owner_uuid", photoEntity.OwnerUuid))
			continue
		}

		photos[i] = PhotoInfo{
			PhotoUuid: photoEntity.PhotoUuid,
			OwnerLogin: user.Login,
			PhotoMetadata: PhotoMetadata{
				Title: photoEntity.Title,
				Description: photoEntity.Description,
				Tags: photoEntity.Tags,
				CreatedAt: photoEntity.CreatedDate,
				TookAt: photoEntity.TookAt,
			},
		}
	}

	return photos, nil
}

func (s *PhotoService) GetPhoto(ctx context.Context, photoUuid uuid.UUID, isSmall bool) (*PhotoWithData, error) {
	log := s.log.With(
		slog.String("op", "service.GetPhoto"),
		slog.String("request_id", middleware.GetReqID(ctx)),
	)

	photoEntity, err := s.photoRepo.GetPhoto(ctx, photoUuid)
	if err != nil {
		if errors.Is(err, storage.ErrPhotoNotFound) {
			log.Error("photo not found", slog.Any("photo_uuid", photoUuid))
			return nil, err
		}

		log.Error("error get photo", sl.Err(err))
		return nil, fmt.Errorf("error get photo: %w", err)
	}

	user, err := s.userRepo.GetUserByUuid(ctx, photoEntity.OwnerUuid)
	if err != nil {
		log.Error("failed to get photo owner", slog.Any("photo_uuid", photoEntity.PhotoUuid), slog.Any("owner_uuid", photoEntity.OwnerUuid))
		return nil, err
	}

	filename := photoEntity.RawFilename
	if isSmall {
		filename = photoEntity.SmallFilename
	}

	rawPhoto, err := s.fileRepo.GetFile(ctx, photoEntity.OwnerUuid.String(), filename)
	if err != nil {
		log.Error("error get photo file", sl.Err(err), slog.String("filename", filename))
		return nil, fmt.Errorf("error get photo file: %w", err)
	}

	photoWithData := &PhotoWithData{
		Content: rawPhoto,
		PhotoInfo: PhotoInfo{
			PhotoUuid: photoEntity.PhotoUuid,
			OwnerLogin: user.Login,
			PhotoMetadata: PhotoMetadata{
				Title: photoEntity.Title,
				Description: photoEntity.Description,
				Tags: photoEntity.Tags,
				CreatedAt: photoEntity.CreatedDate,
				TookAt: photoEntity.TookAt,
			},
		},
	}

	return photoWithData, nil
}

func (s *PhotoService) GetPhotoInfo(ctx context.Context, photoUuid uuid.UUID) (*PhotoInfo, error) {
	log := s.log.With(
		slog.String("op", "service.GetPhoto"),
		slog.String("request_id", middleware.GetReqID(ctx)),
	)

	photoEntity, err := s.photoRepo.GetPhoto(ctx, photoUuid)
	if err != nil {
		if errors.Is(err, storage.ErrPhotoNotFound) {
			log.Error("photo not found", slog.Any("photo_uuid", photoUuid))
			return nil, err
		}

		log.Error("error get photo", sl.Err(err))
		return nil, fmt.Errorf("error get photo: %w", err)
	}

	user, err := s.userRepo.GetUserByUuid(ctx, photoEntity.OwnerUuid)
	if err != nil {
		log.Error("failed to get photo owner", slog.Any("photo_uuid", photoEntity.PhotoUuid), slog.Any("owner_uuid", photoEntity.OwnerUuid))
		return nil, err
	}

	photoInfo := PhotoInfo{
		PhotoUuid: photoEntity.PhotoUuid,
		OwnerLogin: user.Login,
		PhotoMetadata: PhotoMetadata{
			Title: photoEntity.Title,
			Description: photoEntity.Description,
			Tags: photoEntity.Tags,
			CreatedAt: photoEntity.CreatedDate,
			TookAt: photoEntity.TookAt,
		},
	}

	return &photoInfo, nil
}

func (s *PhotoService) DeletePhoto(ctx context.Context, photoUuid uuid.UUID, ownerUuid uuid.UUID) error {
	log := s.log.With(
		slog.String("op", "service.DeletePhoto"),
		slog.String("request_id", middleware.GetReqID(ctx)),
	)

	photoEntity, err := s.photoRepo.GetPhoto(ctx, photoUuid)
	if err != nil {
		if errors.Is(err, storage.ErrPhotoNotFound) {
			log.Error("photo not found", slog.Any("photo_uuid", photoUuid))
			return err
		}

		log.Error("error get photo", sl.Err(err))
		return fmt.Errorf("error get photo: %w", err)
	}

	if photoEntity.OwnerUuid != ownerUuid {	
		return ErrUserInvalidAuthorization
	}

	err = s.photoRepo.DeletePhoto(ctx, photoUuid, ownerUuid)
	if err != nil {
		log.Error("failed to delete photo", sl.Err(err))
		return err
	}

	err = s.fileRepo.DeleteFile(ctx, photoEntity.OwnerUuid.String(), photoEntity.RawFilename)
	if err != nil {
		log.Error("failed to delete photo file", sl.Err(err))
		return err
	}

	err = s.fileRepo.DeleteFile(ctx, photoEntity.OwnerUuid.String(), photoEntity.SmallFilename)
	if err != nil {
		log.Error("failed to delete photo file", sl.Err(err))
		return err
	}

	return nil
}

func (s *PhotoService) UpdatePhotoInfo(ctx context.Context, photoUuid uuid.UUID, metadata PhotoMetadata, userUuid uuid.UUID) error {
	log := s.log.With(
		slog.String("op", "service.UpdatePhotoInfo"),
		slog.String("request_id", middleware.GetReqID(ctx)),
	)

	err := s.photoRepo.UpdatePhoto(ctx, photoUuid, userUuid, metadataToMap(&metadata))

	if err != nil {
		if errors.Is(err, storage.ErrPhotoNotFound) {
			log.Error("photo not found", slog.Any("photo_uuid", photoUuid))
			return err
		}

		log.Error("failed to update photo", sl.Err(err))
		return err
	}

	return nil
}
