package postrgesql

import (
	"context"
	"errors"
	"fmt"
	"photo-viewer-server/internal/storage"
	"photo-viewer-server/internal/storage/entity"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AccessRepository struct {
	*Storage
}

func NewAccessReposotory(st *Storage) *AccessRepository {
	return &AccessRepository{Storage: st}
}

func (s *AccessRepository) SavePhotoAccessLink(ctx context.Context, accessLink *entity.PhotoAccessLink) error {
	err := s.getDB(ctx).Create(accessLink).Error

	if err != nil {
		return fmt.Errorf("error persist photo access link entity: %w", err)
	}

	return nil
}

func (s *AccessRepository) GetValidAccessLinkByPhotoUuid(ctx context.Context, photoUuid uuid.UUID) (*entity.PhotoAccessLink, error) {
	var accessLink entity.PhotoAccessLink

	now := time.Now()
	res := s.getDB(ctx).Model(entity.PhotoAccessLink{}).Where("photo_uuid = ?", photoUuid).Where("expires_at > ?", now).First(&accessLink)

	if res.Error != nil {
		if errors.Is(res.Error, gorm.ErrRecordNotFound) {
			return nil, storage.ErrAccessLinkNotFound
		}

		return nil, fmt.Errorf("error get access link: %w", res.Error)
	}

	return &accessLink, nil
}

func (s *AccessRepository) GrantAccessPhotoToUser(ctx context.Context, photoUuid uuid.UUID, userUuid uuid.UUID) error {
	permittedUserEntity := entity.PhotoPermittedUser{
		PhotoUuid: photoUuid,
		UserUuid: userUuid,
	}
	
	err := s.getDB(ctx).Create(&permittedUserEntity).Error

	if err != nil {
		return  fmt.Errorf("error persist photo permitted user entity: %w", err)
	}

	return nil
}

func (s *AccessRepository) IsUserCanAccessPhotoByUuid(ctx context.Context, photoUuid uuid.UUID, userUuid uuid.UUID) (bool, error) {
	var count int64
	err := s.db.Model(&entity.PhotoPermittedUser{}).Where("photo_uuid = ? AND user_uuid = ?", photoUuid, userUuid).Count(&count).Error

	if err != nil {
		return false, fmt.Errorf("error check user access: %w", err)
	}

	return count > 0, nil
}
