package postrgesql

import (
	"context"
	"errors"
	"fmt"
	"photo-viewer-server/internal/storage"
	"photo-viewer-server/internal/storage/entity"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type CollectionRepository struct {
	*Storage
}

func NewCollectionRepository(st *Storage) *CollectionRepository {
	return &CollectionRepository{Storage: st}
}

func (s *CollectionRepository) SaveCollection(ctx context.Context, collection *entity.Collection) (uuid.UUID, error) {
	err := s.getDB(ctx).Create(collection).Error

	if err != nil {
		return uuid.Nil, fmt.Errorf("error persist collection entity: %w", err)
	}

	if len(collection.Photos) > 0 {
		collectionPhotoEntities := make([]entity.CollectionPhotoEntity, len(collection.Photos))

		for i, photo := range collection.Photos {
			collectionPhotoEntities[i] = entity.CollectionPhotoEntity{
				CollectionUuid: collection.CollectionUuid,
				PhotoUuid: photo.PhotoUuid,
			}
		}

		err = s.getDB(ctx).Table(entity.CollectionPhotoEntity{}.TableName()).Create(&collectionPhotoEntities).Error

		if err != nil {
			if errors.Is(err, gorm.ErrForeignKeyViolated) {
				return uuid.Nil, storage.ErrPhotoNotFound
			}

			return uuid.Nil, fmt.Errorf("error add relation collection with photo: %w", err)
		}
	}

	return collection.CollectionUuid, nil
}

func (s *CollectionRepository) GetCollection(ctx context.Context, collectionUuid uuid.UUID) (*entity.Collection, error) {
	var collection entity.Collection

	err := s.getDB(ctx).Preload("Photos").First(&collection, collectionUuid).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, storage.ErrCollectionNotFound
		}

		return nil, fmt.Errorf("error get collection: %w", err)
	}

	return &collection, nil
}

func (s *CollectionRepository) GetAllCollections(ctx context.Context) ([]entity.Collection, error) {
	var collections []entity.Collection

	err := s.db.Preload("Photos").Where("access_level = ?", "public").Find(&collections).Error

	if err != nil {
		return nil, fmt.Errorf("error get collections: %w", err)
	}

	return collections, nil
}

func (s *CollectionRepository) GetCollectionsByOwner(ctx context.Context, ownerUuid uuid.UUID) ([]entity.Collection, error) {
	var collections []entity.Collection

	err := s.db.Preload("Photos").Where("access_level = ?", "public").Where("owner_uuid = ?", ownerUuid).Find(&collections).Error

	if err != nil {
		return nil, fmt.Errorf("error get collections by owner: %w", err)
	}

	return collections, nil
}

func (s *CollectionRepository) GetAllCollectionsByOwner(ctx context.Context, ownerUuid uuid.UUID) ([]entity.Collection, error) {
	var collections []entity.Collection

	err := s.db.Preload("Photos").Where("owner_uuid = ?", ownerUuid).Find(&collections).Error

	if err != nil {
		return nil, fmt.Errorf("error get photos by owner: %w", err)
	}

	return collections, nil
}

func (s *CollectionRepository) DeleteCollection(ctx context.Context, collectionUuid, ownerUuid uuid.UUID) error {
	err := s.db.Where("owner_uuid = ?", ownerUuid).Delete(entity.Collection{}, collectionUuid).Error

	if err != nil {
		return fmt.Errorf("error delete collection: %w", err)
	}

	return nil
}

func (s *CollectionRepository) AddPhotoToCollection(ctx context.Context, collectionUuid, photoUuid uuid.UUID) error {
	collectionPhotoEntity := entity.CollectionPhotoEntity{
		CollectionUuid: collectionUuid,
		PhotoUuid: photoUuid,
	}

	err := s.getDB(ctx).Create(&collectionPhotoEntity).Error

	if err != nil {
		return fmt.Errorf("error persist collection photo entity: %w", err)
	}

	return nil
}

func (s *CollectionRepository) RemovePhotoFromCollection(ctx context.Context, collectionUuid, photoUuid uuid.UUID) error {
	err := s.getDB(ctx).Where("collection_uuid = ?", collectionUuid).Where("photo_uuid = ?", photoUuid).Delete(entity.CollectionPhotoEntity{}).Error

	if err != nil {
		return fmt.Errorf("error remove collection photo entity: %w", err)
	}

	return nil
}
