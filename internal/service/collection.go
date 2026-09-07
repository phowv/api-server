package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"photo-viewer-server/internal/lib/logger/sl"
	"photo-viewer-server/internal/storage"
	"photo-viewer-server/internal/storage/entity"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
)

type CollectionMetadata struct {
	Title string `json:"title" validate:"max=50"`
	Description string `json:"description" validate:"max=200"`
	AccessLevel entity.AccessModifier `json:"access_level" validate:"required,access_modifier"`
}

type SimpleCollectionInfo struct {
	CollectionMetadata
	CollectionUuid uuid.UUID `json:"collection_uuid"`
	OwnerLogin string `json:"owner_login"`
}

type CollectionInfo struct {
	SimpleCollectionInfo
	Photos []PhotoSmallInfo `json:"photos"`
}

type SaveCollectionInputMetadata struct {
	CollectionMetadata
	PhotoUuids []uuid.UUID `json:"photo_uuids"`
}

type CollectionService struct {
	log *slog.Logger
	collectionRepo CollectionRepo
	userRepo UserRepo
	txManager storage.TxManager
	accessRepo AccessRepo
	keySigner PhotoKeySigner
	photoRepo PhotoRepo
}

func NewCollectionService(
	log *slog.Logger,
	collectionRepo CollectionRepo,
	userRepo UserRepo,
	txManager storage.TxManager,
	accessRepo AccessRepo,
	keySigner PhotoKeySigner,
	photoRepo PhotoRepo,
) *CollectionService {
	return &CollectionService{
		log: log,
		collectionRepo: collectionRepo,
		userRepo: userRepo,
		txManager: txManager,
		accessRepo: accessRepo,
		keySigner: keySigner,
		photoRepo: photoRepo,
	}
}

func (s *CollectionService) SaveCollection(ctx context.Context, input *SaveCollectionInputMetadata, ownerUuid uuid.UUID) (uuid.UUID, error) {
	log := s.log.With(
		slog.String("op", "service.SaveCollection"),
		slog.String("request_id", middleware.GetReqID(ctx)),
	)

	var collectionUuid uuid.UUID
	var err error

	err = s.txManager.WithTransaction(ctx, func(txCtx context.Context) error {
		err = s.userRepo.DecrementCollectionsQuotaByUuid(txCtx, ownerUuid)

		if err != nil {
			if errors.Is(err, storage.ErrUserQuotaIsNotEnough) {
				return ErrUserQuotaIsNotEnough
			}

			return fmt.Errorf("failed to decrement user quota: %w", err)
		}

		photos := make([]entity.Photo, len(input.PhotoUuids))

		for i, photoUuid := range input.PhotoUuids {
			photos[i] = entity.Photo{
				PhotoUuid: photoUuid,
			}
		}

		collection := entity.Collection{
			OwnerUuid: ownerUuid,
			Title: input.Title,
			Description: input.Description,
			AccessLevel: input.AccessLevel,
			Photos: photos,
		}

		collectionUuid, err = s.collectionRepo.SaveCollection(txCtx, &collection)
		if err != nil {
			if errors.Is(err, storage.ErrCollectionAlreadyExists) {
				return ErrCollectionAlreadyExists
			}

			return fmt.Errorf("failed to save collection: %w", err)
		}

		return nil
	})

	if err != nil {
		log.Error("failed to transact collection", sl.Err(err))

		if errors.Is(err, ErrUserQuotaIsNotEnough) {
			return uuid.Nil, ErrUserQuotaIsNotEnough

		} else if errors.Is(err, ErrCollectionAlreadyExists) {
			return uuid.Nil, ErrCollectionAlreadyExists
		}

		return uuid.Nil, fmt.Errorf("failed to transact collection")
	}

	return collectionUuid, nil
}

func (s *CollectionService) GetCollection(ctx context.Context, collectionUuid uuid.UUID) (*CollectionInfo, error) {
	log := s.log.With(
		slog.String("op", "service.GetCollection"),
		slog.String("request_id", middleware.GetReqID(ctx)),
	)

	collectionEntity, err := s.collectionRepo.GetCollection(ctx, collectionUuid)
	if err != nil {
		if errors.Is(err, storage.ErrPhotoNotFound) {
			log.Error("photo not found", slog.Any("collection_uuid", collectionUuid))
			return nil, ErrCollectionNotFound
		}

		log.Error("failed to get collection info", slog.Any("collection_uuid", collectionUuid))
		return nil, fmt.Errorf("failed to get collection")
	}

	isPermit, err := s.isCollectionPermit(ctx, collectionEntity)

	if err != nil {
		log.Error("failed to check photo permissions", sl.Err(err))
		return nil, ErrCollectionIsNotPermitted
	}

	if !isPermit {
		log.Info("photo is not permitted", slog.Any("collection_uuid", collectionUuid))
		return nil, ErrCollectionIsNotPermitted
	}

	user, err := s.userRepo.GetUserByUuid(ctx, collectionEntity.OwnerUuid)
	if err != nil {
		log.Error("failed to get collection's owner", sl.Err(err), slog.Any("collection_uuid", collectionEntity.CollectionUuid), slog.Any("owner_uuid", collectionEntity.OwnerUuid))
		return nil, fmt.Errorf("failed to get owner")
	}

	photosInfo := make([]PhotoSmallInfo, 0)

	for _, photoEntity := range collectionEntity.Photos {
		isPhotoPermit, err := s.isPhotoPermitInCollection(ctx, collectionEntity, &photoEntity)

		if err != nil {
			log.Error("faild to check photo permissions in collection", sl.Err(err))
			continue
		}
		if !isPhotoPermit {
			log.Debug("photo is not permitted in coillection")
			continue
		}
	
		accessKey, err := s.keySigner.Sign(
			photoEntity.PhotoUuid,
			photoEntity.OwnerUuid,
			photoEntity.RawFilename,
			photoEntity.MediumFilename,
			photoEntity.SmallFilename,
		)

		photosInfo = append(photosInfo, PhotoSmallInfo{
			PhotoUuid: photoEntity.PhotoUuid,
			AccessKey: accessKey,
		})
	}

	return &CollectionInfo{
		Photos: photosInfo,
		SimpleCollectionInfo: SimpleCollectionInfo{
			CollectionUuid: collectionEntity.CollectionUuid,
			OwnerLogin: user.Login,
			CollectionMetadata: CollectionMetadata{
				Title: collectionEntity.Title,
				Description: collectionEntity.Description,
				AccessLevel: collectionEntity.AccessLevel,
			},
		},
	}, nil
}

func (s *CollectionService) GetCollections(ctx context.Context, ownerLogin string) ([]SimpleCollectionInfo, error) {
	log := s.log.With(
		slog.String("op", "service.GetCollections"),
		slog.String("request_id", middleware.GetReqID(ctx)),
	)

	var collectionEntities []entity.Collection
	var err error

	requestUserUuidStr := ctx.Value("user_uuid")

	if requestUserUuidStr != nil {
		requestUserUuid, ok := requestUserUuidStr.(uuid.UUID)
		if !ok {
			log.Error("request user uuid has invalid type ")
			return nil, errors.ErrUnsupported
		}

		if ownerLogin == "" {
			collectionEntities, err = s.collectionRepo.GetAllPermittedCollections(ctx, requestUserUuid)

			if err != nil {
				log.Error("failed to get permitted collections", sl.Err(err), slog.Any("user_uuid", requestUserUuid))
				return nil, fmt.Errorf("failed to get permitted collections")
			}
		} else {
			user, err := s.userRepo.GetUserByLogin(ctx, ownerLogin)
			if err != nil {
				log.Error("failed to get collection's owner", sl.Err(err))
				return nil, ErrUserNotFound
			}

			collectionEntities, err = s.collectionRepo.GetAllPermittedCollectionsByOwner(ctx, requestUserUuid, user.UserUuid)
			if err != nil {
				log.Error("failed to get permitted collections with specified owner", sl.Err(err), slog.Any("owner_uuid", user.UserUuid), slog.Any("user_uuid", requestUserUuid))
				return nil, fmt.Errorf("failed to get permitted collections with specified owner")
			}
		}

	} else {
		if ownerLogin == "" {
			collectionEntities, err = s.collectionRepo.GetAllCollections(ctx)

			if err != nil {
				log.Error("failed to get all public collections", sl.Err(err))
				return nil, fmt.Errorf("failed to get all public collections")
			}
		} else {
			user, err := s.userRepo.GetUserByLogin(ctx, ownerLogin)
			if err != nil {
				log.Error("failed to get collection's owner", sl.Err(err))
				return nil, ErrUserNotFound
			}
			collectionEntities, err = s.collectionRepo.GetCollectionsByOwner(ctx, user.UserUuid)

			if err != nil {
				log.Error("failed to get public collections with specified owner", sl.Err(err), slog.Any("owner_uuid", user.UserUuid))
				return nil, fmt.Errorf("failed to get public collections with specified owner")
			}
		}
	}

	collections := make([]SimpleCollectionInfo, len(collectionEntities))

	for i, collectionEntity := range collectionEntities {
		user, err := s.userRepo.GetUserByUuid(ctx, collectionEntity.OwnerUuid)
		if err != nil {
			log.Error("failed to get collection's owner", slog.Any("collection_uuid", collectionEntity.CollectionUuid), slog.Any("owner_uuid", collectionEntity.OwnerUuid))
			return nil, err
		}

		collections[i] = SimpleCollectionInfo{
			CollectionUuid: collectionEntity.CollectionUuid,
			OwnerLogin: user.Login,
			CollectionMetadata: CollectionMetadata{
				Title: collectionEntity.Title,
				Description: collectionEntity.Description,
				AccessLevel: collectionEntity.AccessLevel,
			},
		}
	}

	return collections, nil
}

func (s *CollectionService) AddPhotoToCollection(ctx context.Context, collectionUuid, photoUuid uuid.UUID) error {
	log := s.log.With(
		slog.String("op", "service.AddPhotoToCollection"),
		slog.String("request_id", middleware.GetReqID(ctx)),
	)

	err := s.txManager.WithTransaction(ctx, func(txCtx context.Context) error {
		collectionEntity, err := s.collectionRepo.GetCollection(txCtx, collectionUuid)
		if err != nil {
			log.Error("failed get collection for add photo", sl.Err(err))

			if errors.Is(err, storage.ErrCollectionNotFound) {
				return ErrCollectionNotFound
			}

			return fmt.Errorf("failed get collection: %w", err)
		}

		photoEntity, err := s.photoRepo.GetPhoto(txCtx, photoUuid)
		if err != nil {
			log.Error("failed get photo for add to collection", sl.Err(err))

			if errors.Is(err, storage.ErrPhotoNotFound) {
				return ErrPhotoNotFound
			}

			return fmt.Errorf("failed get photo: %w", err)
		}

		isPermit, err := s.isCollectionPermitToAdd(txCtx, collectionEntity, photoEntity)

		if err != nil {
			log.Error("add photo to collection is not permitted", sl.Err(err))
			return fmt.Errorf("add photo to collection is not permitted")
		}

		if !isPermit {
			return ErrCollectionActionIsNotPermitted
		}

		err = s.collectionRepo.AddPhotoToCollection(txCtx, collectionUuid, photoUuid)
		if err != nil {
			log.Error("failed to add photo to collection", sl.Err(err))

			if errors.Is(err, storage.ErrPhotoInCollectionAlreadyExists) {
				return ErrPhotoInCollectionAlreadyExists
			}
		}

		return nil
	})

	if err != nil {
		log.Error("failed to add photo to collection", sl.Err(err))

		switch err {
		case ErrCollectionNotFound: return ErrCollectionNotFound
		case ErrPhotoNotFound: return ErrPhotoNotFound
		case ErrCollectionActionIsNotPermitted: return ErrCollectionActionIsNotPermitted
		case ErrPhotoInCollectionAlreadyExists: return ErrPhotoInCollectionAlreadyExists
		}

		return fmt.Errorf("failed to add photo to collection")
	}

	log.Info("add photo to collection", slog.Any("collection_uuid", collectionUuid), slog.Any("photo_uuid", photoUuid))
	return nil
}

func (s *CollectionService) RemovePhotoFromCollection(ctx context.Context, collectionUuid, photoUuid uuid.UUID) error {
	log := s.log.With(
		slog.String("op", "service.RemovePhotoFromCollection"),
		slog.String("request_id", middleware.GetReqID(ctx)),
	)

	err := s.txManager.WithTransaction(ctx, func(txCtx context.Context) error {
		collectionEntity, err := s.collectionRepo.GetCollection(txCtx, collectionUuid)
		if err != nil {
			log.Error("failed get collection for add photo", sl.Err(err))

			if errors.Is(err, storage.ErrCollectionNotFound) {
				return ErrCollectionNotFound
			}

			return fmt.Errorf("failed get collection: %w", err)
		}

		isPermit, err := s.isCollectionPermitToRemove(txCtx, collectionEntity)

		if err != nil {
			log.Error("remove photo from collection is not permitted", sl.Err(err))
			return fmt.Errorf("remove photo from collection is not permitted")
		}

		if !isPermit {
			return ErrCollectionActionIsNotPermitted
		}

		err = s.collectionRepo.RemovePhotoFromCollection(txCtx, collectionUuid, photoUuid)
		if err != nil {
			log.Error("failed to remove photo from collection", sl.Err(err))

			if errors.Is(err, storage.ErrPhotoNotFound) {
				return ErrPhotoNotFound
			}
		}

		return nil
	})

	if err != nil {
		log.Error("failed to remove photo from collection", sl.Err(err))

		switch err {
		case ErrCollectionNotFound: return ErrCollectionNotFound
		case ErrPhotoNotFound: return ErrPhotoNotFound
		case ErrCollectionActionIsNotPermitted: return ErrCollectionActionIsNotPermitted
		}

		return fmt.Errorf("failed to remove photo from collection")
	}

	log.Info("remove photo from collection", slog.Any("collection_uuid", collectionUuid), slog.Any("photo_uuid", photoUuid))
	return nil
}

func (s *CollectionService) isPhotoPermitInCollection(ctx context.Context, collection *entity.Collection, photo *entity.Photo) (bool, error) {
	log := s.log.With(
		slog.String("op", "service.isPhotoPermitInCollection"),
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

	if entity.CompareAccessLevels(photo.AccessLevel, entity.AccessModifierProtected)== 0 &&
	   entity.CompareAccessLevels(collection.AccessLevel, entity.AccessModifierProtected) == 0 {
		return true, nil
	}		

	log.Error("photo in collection access denied")
	return false, nil
}

func (s *CollectionService) isCollectionPermit(ctx context.Context, collection *entity.Collection) (bool, error) {
	log := s.log.With(
		slog.String("op", "service.isCollectionPermit"),
		slog.String("request_id", middleware.GetReqID(ctx)),
	)

	if entity.CompareAccessLevels(collection.AccessLevel, entity.AccessModifierPublic) == 0 {
		return true, nil
	}

	requestUserUuid := ctx.Value("user_uuid")

	if requestUserUuid != nil {
		userUuid, ok := requestUserUuid.(uuid.UUID)
		if !ok {
			log.Error("request user uuid has invalid type")
			return false, errors.ErrUnsupported
		}

		if collection.OwnerUuid == userUuid {
			return true, nil
		}

		if entity.CompareAccessLevels(collection.AccessLevel, entity.AccessModifierPrivate) >= 0 {
			return false, nil
		}

		canAccess, err := s.accessRepo.IsUserCanAccessCollectionByUuid(ctx, collection.CollectionUuid, userUuid)

		if err != nil {
			log.Error("failed to check user acces to photo", sl.Err(err))
			return false, fmt.Errorf("failed to check user acces to photo: %w", err)
		}

		if canAccess {
			return true, nil
		}
	}

	if entity.CompareAccessLevels(collection.AccessLevel, entity.AccessModifierPrivate) >= 0 {
		return false, nil
	}

	requestAccessKey := ctx.Value("collection_access_secret")

	if requestAccessKey != nil {
		accessKey, ok := requestAccessKey.(string)
		if !ok {
			return false, errors.ErrUnsupported
		}

		accessLink, err := s.accessRepo.GetValidAccessLinkByCollectionUuid(ctx, collection.CollectionUuid)
		if err != nil {
			log.Error("failed to get access link", sl.Err(err))
			return false, fmt.Errorf("failed to get access link: %w", err)
		}
		if comparePasswords(accessKey, accessLink.HashCode) {
			return true, nil
		}
	}

	log.Error("collection access denied", slog.Any("collection_uuid", collection.CollectionUuid))

	return false, nil
}

func (s *CollectionService) isCollectionPermitToAdd(ctx context.Context, collection *entity.Collection, photo *entity.Photo) (bool, error) {
	log := s.log.With(
		slog.String("op", "service.isCollectionPermitToAdd"),
		slog.String("request_id", middleware.GetReqID(ctx)),
	)

	requestUserUuid := ctx.Value("user_uuid")

	if requestUserUuid != nil {
		userUuid, ok := requestUserUuid.(uuid.UUID)
		if !ok {
			log.Error("request user uuid has invalid type")
			return false, errors.ErrUnsupported
		}

		if collection.OwnerUuid == userUuid {
			return true, nil
		}

		if entity.CompareAccessLevels(photo.AccessLevel, entity.AccessModifierPrivate) >= 0 {
			return false, nil
		}

		if entity.CompareAccessLevels(collection.AccessLevel, entity.AccessModifierPrivate) >= 0 {
			return false, nil
		}

		canAccess, err := s.accessRepo.IsUserCanAccessCollectionByUuid(ctx, collection.CollectionUuid, userUuid)

		if err != nil {
			log.Error("failed to check user acces to photo", sl.Err(err))
			return false, fmt.Errorf("failed to check user acces to photo: %w", err)
		}

		if canAccess {
			return true, nil
		}
	}

	log.Error("collection access denied", slog.Any("collection_uuid", collection.CollectionUuid))
	return false, nil
}

func (s *CollectionService) isCollectionPermitToRemove(ctx context.Context, collection *entity.Collection) (bool, error) {
	log := s.log.With(
		slog.String("op", "service.isCollectionPermitToAdd"),
		slog.String("request_id", middleware.GetReqID(ctx)),
	)

	requestUserUuid := ctx.Value("user_uuid")

	if requestUserUuid != nil {
		userUuid, ok := requestUserUuid.(uuid.UUID)
		if !ok {
			log.Error("request user uuid has invalid type")
			return false, errors.ErrUnsupported
		}

		if collection.OwnerUuid == userUuid {
			return true, nil
		}
	}

	log.Error("collection access denied", slog.Any("collection_uuid", collection.CollectionUuid))
	return false, nil
}

