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
	err := s.getDB(ctx).Omit("Tags").Create(photo).Error

	if err != nil {
		return uuid.Nil, fmt.Errorf("error persist photo entity: %w", err)
	}

	if len(photo.Tags) > 0 {
		photoTagEntities := make([]entity.PhotoTagEntity, len(photo.Tags))

		for i, tag := range photo.Tags {
			photoTagEntities[i] = entity.PhotoTagEntity{
				PhotoUuid: photo.PhotoUuid,
				TagUuid: tag.TagUuid,
			}
		}

		err = s.getDB(ctx).Table(entity.PhotoTagEntity{}.TableName()).Create(&photoTagEntities).Error

		if err != nil {
			if errors.Is(err, gorm.ErrForeignKeyViolated) {
				return uuid.Nil, storage.ErrTagNotFound
			}

			return uuid.Nil, fmt.Errorf("error add relation photo with tag: %w", err)
		}
	}

	return photo.PhotoUuid, nil
}

func (s *PhotoRepository) GetPhoto(ctx context.Context, uuid uuid.UUID) (*entity.Photo, error) {
	var photo entity.Photo

	err := s.getDB(ctx).Preload("Tags").First(&photo, uuid).Error

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

	err := s.db.Preload("Tags").Where("access_level = ?", "public").Find(&photos).Error

	if err != nil {
		return nil, fmt.Errorf("error get photos: %w", err)
	}

	return photos, nil
}

func (s *PhotoRepository) GetAllPhotosByOwner(ctx context.Context, ownerUuid uuid.UUID) ([]entity.Photo, error) {
	var photos []entity.Photo

	err := s.db.Preload("Tags").Where("access_level = ?", "public").Where("owner_uuid = ?", ownerUuid).Find(&photos).Error

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

func (s *PhotoRepository) SaveTag(ctx context.Context, tag *entity.Tag) (uuid.UUID, error) {
	err := s.getDB(ctx).Create(tag).Error

	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return uuid.Nil, storage.ErrTagAlreadyExists
		}

		return uuid.Nil, fmt.Errorf("error persist tag entity: %w", err)
	}

	return tag.TagUuid, nil
}

func (s *PhotoRepository) GetTag(ctx context.Context, tagUuid uuid.UUID) (*entity.Tag, error) {
	var tag entity.Tag

	err := s.getDB(ctx).First(&tag, tagUuid).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, storage.ErrPhotoNotFound
		}

		return nil, fmt.Errorf("error get tag: %w", err)
	}

	return &tag, nil
}

func (s *PhotoRepository) DeleteTag(ctx context.Context, tagUuid uuid.UUID) error {
	err := s.db.Delete(entity.Tag{}, tagUuid).Error

	if err != nil {
		return fmt.Errorf("error delete tag: %w", err)
	}

	return nil
}

func (s *PhotoRepository) GetAllTags(ctx context.Context) ([]entity.Tag, error) {
	var tags []entity.Tag

	err := s.getDB(ctx).Find(&tags).Error

	if err != nil {
		return nil, fmt.Errorf("error get tags: %w", err)
	}

	return tags, nil
}

func (s *PhotoRepository) GetTagsByPhoto(ctx context.Context, photoUuid uuid.UUID) ([]entity.Tag, error) {
	var tags []entity.Tag

	err := s.getDB(ctx).Joins("JOIN photo.photo_tags ON photo_tags.tag_uuid = tag.tag_uuid").Where("photo_tags.photo_uuid = ?", photoUuid).Find(&tags).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, storage.ErrPhotoNotFound
		}

		return nil, fmt.Errorf("error get tags by photo uuid: %w", err)
	}

	return tags, nil
}

func (s *PhotoRepository) GetTagByName(ctx context.Context, tagName string) (*entity.Tag, error) {
	var tag entity.Tag

	err := s.getDB(ctx).Where("tag_name = ?", tagName).First(&tag).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, storage.ErrPhotoNotFound
		}

		return nil, fmt.Errorf("error get tag by name: %w", err)
	}

	return &tag, nil
}
