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

type StoredPhotoType string

var (
	PhotoSizeSmall StoredPhotoType = "small"
	PhotoSizeMedium StoredPhotoType = "medium"
	PhotoSizeRaw StoredPhotoType = "raw"

	ErrInvalidPhotoSize = errors.New("invalid photo size")
	ErrPhotoNotFound = errors.New("photo not found")
	ErrPhotoIsNotPermitted = errors.New("photo is not permitted")
)

const (
	mediumImageSize = 800
	mediumImagePostfix = "_medium"

	smallImageSize = 300
	smallImagePostfix = "_small"
)

type PhotoMetadata struct {
	Title string `json:"title" validate:"max=50"`
	Description string `json:"description" validate:"max=200"`
	CreatedAt time.Time `json:"created_at"`
	TookAt time.Time `json:"took_at"`
	AccessLevel entity.AccessModifier `json:"access_level" validate:"required,access_modifier"`
}

type PhotoInfo struct {
	PhotoUuid uuid.UUID `json:"photo_uuid"`
	OwnerLogin string `json:"owner_login"`
	Tags []TagSmallInfo `json:"tags"`
	AccessKey string `json:"access_key"`
	PhotoMetadata
}

type PhotoSmallInfo struct {
	PhotoUuid uuid.UUID `json:"photo_uuid"`
	AccessKey string `json:"access_key"`
}

type SavePhotoInputMetadata struct {
	PhotoMetadata
	TagUuids []uuid.UUID `json:"tag_uuids"`
}

type SavePhotoInput struct {
	Metadata SavePhotoInputMetadata
	Filename string
	Content []byte
	ContentType string
}

type imageWithType struct {
	name string
	content []byte
	contentType string
}

type PhotoService struct {
	log *slog.Logger
	photoRepo PhotoRepo
	fileRepo FileRepo
	bucketName string
	userRepo UserRepo
	imageProcessor ImageProcessor
	txManager storage.TxManager
	accessRepo AccessRepo
	keySigner PhotoKeySigner
}

func NewPhotoService(
	log *slog.Logger,
	photoRepo PhotoRepo,
	fileRepo FileRepo,
	bucketName string,
	userRepo UserRepo,
	imageProcessor ImageProcessor,
	txManager storage.TxManager,
	accessReoo AccessRepo,
	keySigner PhotoKeySigner,
) *PhotoService {
	return &PhotoService{
		log: log,
		photoRepo: photoRepo,
		fileRepo: fileRepo,
		bucketName: bucketName,
		userRepo: userRepo,
		imageProcessor: imageProcessor,
		txManager: txManager,
		accessRepo: accessReoo,
		keySigner: keySigner,
	}
}

func metadataToMap(m *PhotoMetadata) map[string]any {
	if m == nil {
		return nil
	}
	out := make(map[string]any)

	if m.Title != "" {
		out["title"] = m.Title
	}

	if m.Description != "" {
		out["description"] = m.Description
	}
	
	if !m.CreatedAt.IsZero() {
		out["created_at"] = m.CreatedAt
	}

	if !m.TookAt.IsZero() {
		out["took_at"] = m.TookAt
	}

	out["access_level"] = m.AccessLevel

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
	newMediumFilename := newPhotoUuid + mediumImagePostfix + ext
	newSmallFilename := newPhotoUuid + smallImagePostfix + ext

	originalFileData := input.Content

	mediumFileData, err := s.imageProcessor.ResizeAndCompress(ctx, originalFileData, mediumImageSize, mediumImageSize, 70)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to resize and compress image to medium size: %w", err)
	}

	smallFileData, err := s.imageProcessor.ResizeAndCompress(ctx, originalFileData, smallImageSize, smallImageSize, 70)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to resize and compress image to small size: %w", err)
	}

	var photoUuid uuid.UUID
	err = s.txManager.WithTransaction(ctx, func(txCtx context.Context) error {
		err := s.userRepo.DecrementPhotosQuotaByUuid(txCtx, ownerUuid)

		if err != nil {
			if (errors.Is(err, storage.ErrUserQuotaIsNotEnough)) {
				return ErrUserQuotaIsNotEnough
			}

			return fmt.Errorf("failed to decrement user's quota: %w", err)
		}

		var savedImages []string
		cleanup := func() error {
			log.Info("cleanup saved images")
			for _, image := range savedImages {
			  err = s.fileRepo.DeleteFile(txCtx, ownerUuid.String(), image)
				if err != nil {
					return fmt.Errorf("failed to cleanup image %s: %w", image, err)
				}
			}
			return nil
		}

		rawFilename, err := s.fileRepo.SaveFile(ctx, ownerUuid.String(), newRawFilename, originalFileData, input.ContentType)
		if err != nil {
			log.Error("failed to save raw photo file", sl.Err(err))

			cleanupErr := cleanup()
			if cleanupErr != nil {
				return fmt.Errorf("failed to save photo file: %w: %w", err, cleanupErr)
			}
			return fmt.Errorf("failed to save photo file: %w", err)
		}
		savedImages = append(savedImages, newRawFilename)

		mediumFilename, err := s.fileRepo.SaveFile(ctx, ownerUuid.String(), newMediumFilename, mediumFileData, input.ContentType)
		if err != nil {
			log.Error("failed to save medium photo file", sl.Err(err))

			cleanupErr := cleanup()
			if cleanupErr != nil {
				return fmt.Errorf("failed to save photo file: %w: %w", err, cleanupErr)
			}
			return fmt.Errorf("failed to save photo file: %w", err)
		}
		savedImages = append(savedImages, newMediumFilename)

		smallFilename, err := s.fileRepo.SaveFile(ctx, ownerUuid.String(), newSmallFilename, smallFileData, input.ContentType)
		if err != nil {
			log.Error("failed to save small photo file", sl.Err(err))

			cleanupErr := cleanup()
			if cleanupErr != nil {
				return fmt.Errorf("failed to save photo file: %w: %w", err, cleanupErr)
			}
			return fmt.Errorf("failed to save photo file: %w", err)
		}
		savedImages = append(savedImages, newSmallFilename)

		log.Info("saved photo to file storage", slog.String("filename", rawFilename), slog.String("medium_filename", mediumFilename), slog.String("small_filename", smallFilename))

		tags := make([]entity.Tag, len(input.Metadata.TagUuids))

		for i, tagUuid := range input.Metadata.TagUuids {
			tags[i] = entity.Tag{
				TagUuid: tagUuid,
			}
		}

		photoEntity := entity.Photo{
			Title: input.Metadata.Title,
			Description: input.Metadata.Description,
			Tags: tags,
			CreatedDate: input.Metadata.CreatedAt,
			TookAt: input.Metadata.TookAt,
			RawFilename: rawFilename,
			MediumFilename: mediumFilename,
			SmallFilename: smallFilename,
			OwnerUuid: ownerUuid,
			AccessLevel: input.Metadata.AccessLevel,
		}

		photoUuid, err = s.photoRepo.SavePhoto(ctx, &photoEntity)

		if err != nil {
			log.Error("error save photo metadata", sl.Err(err))

			if errors.Is(err, storage.ErrTagNotFound) {
			  cleanup()
				return ErrTagDoesNotExists
			}

			cleanup()
			return fmt.Errorf("error save photo metadata: %w", err)
		}

		return nil
	})

	if err != nil {
		if errors.Is(err, ErrUserQuotaIsNotEnough) {
			return uuid.Nil, ErrUserQuotaIsNotEnough
		} else if errors.Is(err, ErrTagDoesNotExists) {
			return uuid.Nil, ErrTagDoesNotExists
		}

		return uuid.Nil, fmt.Errorf("failed to transact photo data: %w", err)
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

		requestUserUuid := ctx.Value("user_uuid")
		userUuid, ok := requestUserUuid.(uuid.UUID)

		log.Debug("getting photos list", slog.Any("request_user_uuid", userUuid), slog.Any("user_uuid", user.UserUuid))

		if ok {
			if userUuid == user.UserUuid {
				log.Debug("get all photos for user", slog.Any("user_uuid", userUuid))
		  	photoEnities, err = s.photoRepo.GetAllPhotosByOwner(ctx, user.UserUuid)
			}
		} else {
			photoEnities, err = s.photoRepo.GetPhotosByOwner(ctx, user.UserUuid)
		}
	}

	if err != nil {
		log.Error("failed to get all photos", sl.Err(err))

		return nil, fmt.Errorf("failed to get all photos: %w", err)
	}

	photos := make([]PhotoInfo, len(photoEnities))

	for i, photoEntity := range photoEnities {
		user, err := s.userRepo.GetUserByUuid(ctx, photoEntity.OwnerUuid)
		if err != nil {
			log.Error("failed to get photo owner", sl.Err(err), slog.Any("photo_uuid", photoEntity.PhotoUuid), slog.Any("owner_uuid", photoEntity.OwnerUuid))
			continue
		}

		photoTags := make([]TagSmallInfo, len(photoEntity.Tags))

		for i, photoEntityTag := range photoEntity.Tags {
			photoTags[i] = TagSmallInfo{
				TagUuid: photoEntityTag.TagUuid,
				TagName: photoEntityTag.Name,
			}
		}

		accessKey, err := s.keySigner.Sign(
			photoEntity.PhotoUuid,
			photoEntity.OwnerUuid,
			photoEntity.RawFilename,
			photoEntity.MediumFilename,
			photoEntity.SmallFilename,
		)

		photos[i] = PhotoInfo{
			PhotoUuid: photoEntity.PhotoUuid,
			OwnerLogin: user.Login,
			Tags: photoTags,
			AccessKey: accessKey,
			PhotoMetadata: PhotoMetadata{
				Title: photoEntity.Title,
				Description: photoEntity.Description,
				CreatedAt: photoEntity.CreatedDate,
				TookAt: photoEntity.TookAt,
				AccessLevel: photoEntity.AccessLevel,
			},
		}
	}

	return photos, nil
}

func (s *PhotoService) GetPhotoFile(ctx context.Context, photoUuid uuid.UUID, accessKey string, photoSize StoredPhotoType) ([]byte, error) {
	log := s.log.With(
		slog.String("op", "service.GetPhotoFile"),
		slog.String("request_id", middleware.GetReqID(ctx)),
	)

	photoKeyPayload, err := s.keySigner.Validate(accessKey, photoUuid)
	if err != nil {
		log.Error("failed to validate access key", sl.Err(err))
		return nil, fmt.Errorf("failed to validate access key")
	}

	filename := photoKeyPayload.PhotoRawFile
	switch photoSize {
		case PhotoSizeSmall: filename = photoKeyPayload.PhotoSmallFile
		case PhotoSizeMedium: filename = photoKeyPayload.PhotoMediumFile
		case PhotoSizeRaw: filename = photoKeyPayload.PhotoRawFile
	}
	
	rawPhoto, _, err := s.fileRepo.GetFile(ctx, photoKeyPayload.OwnerUuid.String(), filename)
	if err != nil {
		log.Error("error get photo file", sl.Err(err), slog.String("filename", filename))
		return nil, fmt.Errorf("error get photo file: %w", err)
	}

	return rawPhoto, nil
}

func (s *PhotoService) GetPhotoInfo(ctx context.Context, photoUuid uuid.UUID) (*PhotoInfo, error) {
	log := s.log.With(
		slog.String("op", "service.GetPhotoInfo"),
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

	isPermit, err := s.isPhotoPermit(ctx, photoEntity)

	if err != nil {
		log.Error("failed to check photo permissions", sl.Err(err))
		return nil, err
	}

	if !isPermit {
		log.Info("photo is not permitted", slog.Any("photo_uuid", photoUuid))
		return nil, ErrPhotoIsNotPermitted
	}

	user, err := s.userRepo.GetUserByUuid(ctx, photoEntity.OwnerUuid)
	if err != nil {
		log.Error("failed to get photo owner", slog.Any("photo_uuid", photoEntity.PhotoUuid), slog.Any("owner_uuid", photoEntity.OwnerUuid))
		return nil, err
	}

	accessKey, err := s.keySigner.Sign(
		photoEntity.PhotoUuid,
		photoEntity.OwnerUuid,
		photoEntity.RawFilename,
		photoEntity.MediumFilename,
		photoEntity.SmallFilename,
	)

	photoTags := make([]TagSmallInfo, len(photoEntity.Tags))

	for i, photoEntityTag := range photoEntity.Tags {
		photoTags[i] = TagSmallInfo{
			TagUuid: photoEntityTag.TagUuid,
			TagName: photoEntityTag.Name,
		}
	}

	photoInfo := PhotoInfo{
		PhotoUuid: photoEntity.PhotoUuid,
		OwnerLogin: user.Login,
		Tags: photoTags,
		AccessKey: accessKey,
		PhotoMetadata: PhotoMetadata{
			Title: photoEntity.Title,
			Description: photoEntity.Description,
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

	return s.txManager.WithTransaction(ctx, func(txCtx context.Context) error {
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

		rawPhotoData, rawContentType, err := s.fileRepo.GetFile(ctx, photoEntity.OwnerUuid.String(), photoEntity.RawFilename)
		if err != nil {
			return fmt.Errorf("failed to get raw photo file")
		}
		mediumPhotoData, mediumContentType, err := s.fileRepo.GetFile(ctx, photoEntity.OwnerUuid.String(), photoEntity.MediumFilename)
		if err != nil {
			return fmt.Errorf("failed to get medium photo file")
		}
		smallPhotoData, smallContentType, err := s.fileRepo.GetFile(ctx, photoEntity.OwnerUuid.String(), photoEntity.SmallFilename)
		if err != nil {
			return fmt.Errorf("failed to get small photo file")
		}

		var deleted []imageWithType
		restore := func() error {
			for _, image := range deleted {
				_, err := s.fileRepo.SaveFile(txCtx, photoEntity.OwnerUuid.String(), image.name, image.content, image.contentType)
				if err != nil {
					return fmt.Errorf("failed to restore image file: %w", err)
				}
			}
			return nil
		}

		err = s.photoRepo.DeletePhoto(ctx, photoUuid, ownerUuid)
		if err != nil {
			return fmt.Errorf("failed to delete photo: %w", err)
		}

		deleted = append(deleted, imageWithType{name: photoEntity.RawFilename, content: rawPhotoData, contentType: rawContentType})
		err = s.fileRepo.DeleteFile(ctx, photoEntity.OwnerUuid.String(), photoEntity.RawFilename)
		if err != nil {
			restoreErr := restore()
			if restoreErr != nil {
				return fmt.Errorf("failed to delete raw photo file: %w: %w", err, restoreErr)
			}
			return fmt.Errorf("failed to delete raw photo file: %w", err)
		}

		deleted = append(deleted, imageWithType{name: photoEntity.MediumFilename, content: mediumPhotoData, contentType: mediumContentType})
		err = s.fileRepo.DeleteFile(ctx, photoEntity.OwnerUuid.String(), photoEntity.MediumFilename)
		if err != nil {
			restoreErr := restore()
			if restoreErr != nil {
				return fmt.Errorf("failed to delete medium photo file: %w: %w", err, restoreErr)
			}
			return fmt.Errorf("failed to delete medium photo file: %w", err)
		}

		deleted = append(deleted, imageWithType{name: photoEntity.SmallFilename, content: smallPhotoData, contentType: smallContentType})
		err = s.fileRepo.DeleteFile(ctx, photoEntity.OwnerUuid.String(), photoEntity.SmallFilename)
		if err != nil {
			restoreErr := restore()
			if restoreErr != nil {
				return fmt.Errorf("failed to delete small photo file: %w: %w", err, restoreErr)
			}
			return fmt.Errorf("failed to delete small photo file: %w", err)
		}
		return nil
	})
}

func (s *PhotoService) UpdatePhotoInfo(ctx context.Context, photoUuid uuid.UUID, metadata PhotoMetadata, userUuid uuid.UUID) error {
	log := s.log.With(
		slog.String("op", "service.UpdatePhotoInfo"),
		slog.String("request_id", middleware.GetReqID(ctx)),
	)

	err := s.txManager.WithTransaction(ctx, func(txCtx context.Context) error {
	  return s.photoRepo.UpdatePhoto(ctx, photoUuid, userUuid, metadataToMap(&metadata))
	})

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

func (s *PhotoService) isPhotoPermit(ctx context.Context, photo *entity.Photo) (bool, error) {
	log := s.log.With(
		slog.String("op", "service.isPhotoPermit"),
		slog.String("request_id", middleware.GetReqID(ctx)),
	)

	if entity.CompareAccessLevels(photo.AccessLevel, entity.AccessModifierPublic) == 0 {
		return true, nil
	}

	requestUserUuid := ctx.Value("user_uuid")

	if requestUserUuid != nil {
		userUuid, ok := requestUserUuid.(uuid.UUID)
		if !ok {
			log.Error("request user uuid has invalid type ")
			return false, errors.ErrUnsupported
		}

		if photo.OwnerUuid == userUuid {
			return true, nil
		}

		if entity.CompareAccessLevels(photo.AccessLevel, entity.AccessModifierPrivate) >= 0 {
			return false, nil
		}

		canAccess, err := s.accessRepo.IsUserCanAccessPhotoByUuid(ctx, photo.PhotoUuid, userUuid)

		if err != nil {
			log.Error("failed to check user acces to photo", sl.Err(err))
			return false, fmt.Errorf("failed to check user acces to photo: %w", err)
		}

		if canAccess {
			return true, nil
		}
	}

	if entity.CompareAccessLevels(photo.AccessLevel, entity.AccessModifierPrivate) >= 0 {
		return false, nil
	}

	requestAccessKey := ctx.Value("photo_access_secret")

	if requestAccessKey != nil {
		accessKey, ok := requestAccessKey.(string)
		if !ok {
			return false, errors.ErrUnsupported
		}

		accessLink, err := s.accessRepo.GetValidAccessLinkByPhotoUuid(ctx, photo.PhotoUuid)
		if err != nil {
			log.Error("failed to get access link", sl.Err(err))
			return false, fmt.Errorf("failed to get access link: %w", err)
		}
		if comparePasswords(accessKey, accessLink.HashCode) {
			return true, nil
		}
	}

	log.Error("photo access denied")

	return false, nil
}

func StringToStoredPhotoType(value string) (StoredPhotoType, error) {
	switch value {
		case string(PhotoSizeRaw): return PhotoSizeRaw, nil
		case string(PhotoSizeMedium): return PhotoSizeMedium, nil
		case string(PhotoSizeSmall): return PhotoSizeSmall, nil
	}
	return "", ErrInvalidPhotoSize
}
