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

var (
	ErrCollectionAlreadyExists = errors.New("collection already exists")
)

type CollectionMetadata struct {
	Title string `json:"title" validate:"max=50"`
	Description string `json:"description" validate:"max=200"`
	AccessLevel entity.AccessModifier `json:"access_level" validate:"required,access_modifier"`
}

type CollectionInfo struct {
	CollectionMetadata
	CollectionUuid uuid.UUID `json:"collection_uuid"`
	OwnerLogin string `json:"owner_login"`
	Photos []PhotoSmallInfo `json:"photos"`
	AccessKey string `json:"access_key"`
}

type SaveCollectionInputMetadata struct {
	CollectionMetadata
	PhotoUuids []uuid.UUID `json:"photo_uuids"`
}

type CollectionRepo interface {
	SaveCollection(ctx context.Context, collection *entity.Collection) (uuid.UUID, error)
	GetCollection(ctx context.Context, collectionUuid uuid.UUID) (*entity.Collection, error)
	GetAllCollections(ctx context.Context) ([]entity.Collection, error)
	GetCollectionsByOwner(ctx context.Context, ownerUuid uuid.UUID) ([]entity.Collection, error)
	GetAllCollectionsByOwner(ctx context.Context, ownerUuid uuid.UUID) ([]entity.Collection, error)
	DeleteCollection(ctx context.Context, collectionUuid, ownerUuid uuid.UUID) error
	AddPhotoToCollection(ctx context.Context, collectionUuid, photoUuid uuid.UUID) error
	RemovePhotoFromCollection(ctx context.Context, collectionUuid, photoUuid uuid.UUID) error
}

type CollectionService struct {
	log *slog.Logger
	collectionRepo CollectionRepo
	userRepo UserRepo
	txManager storage.TxManager
}

func NewCollectionService(
	log *slog.Logger,
	collectionRepo CollectionRepo,
	userRepo UserRepo,
	txManager storage.TxManager,
) *CollectionService {
	return &CollectionService{
		log: log,
		collectionRepo: collectionRepo,
		userRepo: userRepo,
		txManager: txManager,
	}
}

func (s *CollectionService) SaveCollection(ctx context.Context, input *SaveCollectionInputMetadata, ownerUuid uuid.UUID) (uuid.UUID, error) {
	log := s.log.With(
		slog.String("op", "service.SavePhoto"),
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
