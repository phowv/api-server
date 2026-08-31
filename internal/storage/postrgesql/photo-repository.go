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

type PhotoRepository struct {
	*Storage
}

func NewPhotoRepository(st *Storage) *PhotoRepository {
	return &PhotoRepository{Storage: st}
}

func (s *PhotoRepository) SavePhoto(ctx context.Context, photo *entity.Photo) (uuid.UUID, error) {
	err := s.getDB(ctx).Create(photo).Error

	if err != nil {
		return uuid.Nil, fmt.Errorf("error persist photo entity: %w", err)
	}

	return photo.PhotoUuid, nil
}

func (s *PhotoRepository) GetPhoto(ctx context.Context, uuid uuid.UUID) (*entity.Photo, error) {
	var photo entity.Photo

	err := s.getDB(ctx).First(&photo, uuid).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, storage.ErrPhotoNotFound
		}

		return nil, fmt.Errorf("error get photo: %w", err)
	}

	return &photo, nil
}

func (s *PhotoRepository) GetAllPhotos(ctx context.Context) ([]entity.Photo, error) {
	var photos []entity.Photo

	err := s.db.Find(&photos).Error

	if err != nil {
		return nil, fmt.Errorf("error get photos: %w", err)
	}

	return photos, nil
}

func (s *PhotoRepository) GetAllPhotosByOwner(ctx context.Context, ownerUuid uuid.UUID) ([]entity.Photo, error) {
	var photos []entity.Photo

	err := s.db.Where("owner_uuid = ?", ownerUuid).Find(&photos).Error

	if err != nil {
		return nil, fmt.Errorf("error get photos by owner: %w", err)
	}

	return photos, nil
}

func (s *PhotoRepository) DeletePhoto(ctx context.Context, uuid uuid.UUID, ownerUuid uuid.UUID) error {
	err := s.db.Where("owner_uuid = ?", ownerUuid).Delete(entity.Photo{}, uuid).Error

	if err != nil {
		return fmt.Errorf("error delete photo: %w", err)
	}

	return nil
}

func (s *PhotoRepository) UpdatePhoto(ctx context.Context, uuid uuid.UUID, ownerUuid uuid.UUID, fields map[string]any) error {
  res := s.getDB(ctx).Model(&entity.Photo{}).Where("photo_uuid = ?", uuid).Where("owner_uuid = ?", ownerUuid).Updates(fields)

	if res.Error != nil {
		if errors.Is(res.Error, gorm.ErrRecordNotFound) {
			return storage.ErrPhotoNotFound
		}

		return fmt.Errorf("error update photo: %w", res.Error)
	}

	if res.RowsAffected == 0 {
		return storage.ErrPhotoNotFound
	}

	return nil
}
