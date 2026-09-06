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

type UserRepository struct {
	*Storage
}

func NewUserRepository(st *Storage) *UserRepository {
	return &UserRepository{Storage: st}
}

func (s *UserRepository) CreateUser(ctx context.Context, user *entity.User) (uuid.UUID, error) {
	err := s.getDB(ctx).Create(user).Error

	if err != nil {
		return uuid.Nil, fmt.Errorf("error persist user entity: %w", err)
	}

	return user.UserUuid, nil
}

func (s *UserRepository) GetUserByUuid(ctx context.Context, uuid uuid.UUID) (*entity.User, error) {
	var user entity.User

	err := s.getDB(ctx).First(&user, uuid).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, storage.ErrUserNotFound
		}

		return nil, fmt.Errorf("error get user by id: %w", err)
	}

	return &user, nil
}


func (s *UserRepository) GetUserByEmail(ctx context.Context, email string) (*entity.User, error) {
	var user entity.User

	err := s.getDB(ctx).Where("email = ?", email).First(&user).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, storage.ErrUserNotFound
		}

		return nil, fmt.Errorf("error get user by email: %w", err)
	}

	return &user, nil
}

func (s *UserRepository) GetUserByLogin(ctx context.Context, login string) (*entity.User, error) {
	var user entity.User

	err := s.getDB(ctx).Where("login = ?", login).First(&user).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, storage.ErrUserNotFound
		}

		return nil, fmt.Errorf("error get user by email: %w", err)
	}

	return &user, nil
}

func (s *UserRepository) DeleteUser(ctx context.Context, uuid uuid.UUID)error {
	err := s.getDB(ctx).Delete(entity.User{}, uuid).Error

	if err != nil {
		return fmt.Errorf("error delete user: %w", err)
	}

	return nil
}

func (s *UserRepository) UpdateUser(ctx context.Context, uuid uuid.UUID, fields map[string]any) error {
  res := s.getDB(ctx).Model(&entity.User{}).Where("user_uuid = ?", uuid).Updates(fields)

	if res.Error != nil {
		if errors.Is(res.Error, gorm.ErrRecordNotFound) {
			return storage.ErrUserNotFound
		}

		return fmt.Errorf("error update user: %w", res.Error)
	}

	if res.RowsAffected == 0 {
		return storage.ErrUserNotFound
	}

	return nil
}

func (s* UserRepository) DecrementPhotosQuotaByUuid(ctx context.Context, uuid uuid.UUID) error {
	res := s.getDB(ctx).Model(&entity.User{}).Where("user_uuid = ? AND photos_quota > 0", uuid).UpdateColumn("photos_quota", gorm.Expr("photos_quota - 1"))

	if res.Error != nil {
		if errors.Is(res.Error, gorm.ErrRecordNotFound) {
			return storage.ErrUserNotFound
		}

		return fmt.Errorf("error update user: %w", res.Error)
	}

	if res.RowsAffected == 0 {
		return storage.ErrUserQuotaIsNotEnough
	}

	return nil
}

func (s* UserRepository) DecrementCollectionsQuotaByUuid(ctx context.Context, uuid uuid.UUID) error {
	res := s.getDB(ctx).Model(&entity.User{}).Where("user_uuid = ? AND collections_quota > 0", uuid).UpdateColumn("collections_quota", gorm.Expr("collections_quota - 1"))

	if res.Error != nil {
		if errors.Is(res.Error, gorm.ErrRecordNotFound) {
			return storage.ErrUserNotFound
		}

		return fmt.Errorf("error update user: %w", res.Error)
	}

	if res.RowsAffected == 0 {
		return storage.ErrUserQuotaIsNotEnough
	}

	return nil
}
