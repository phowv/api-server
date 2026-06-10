package postrgesql

import (
	"context"
	"errors"
	"fmt"
	"photo-viewer-server/internal/storage"
	"photo-viewer-server/internal/storage/entity"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Storage struct {
	db *gorm.DB
}

func New(host string, port int, dbname string, user string, password string) (*Storage, error) {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=disable",
		host, user, password, dbname, port)

	db, err := gorm.Open(postgres.Open(dsn))

	if err != nil {
		return nil, fmt.Errorf("failed to connect database: %w", err)
	}

	return &Storage{db: db}, nil
}

func (s *Storage) SavePhoto(ctx context.Context, photo *entity.Photo) (int, error) {
	err := s.db.WithContext(ctx).Create(photo).Error

	if err != nil {
		return 0, fmt.Errorf("error persist photo entity: %w", err)
	}

	return photo.PhotoId, nil
}

func (s *Storage) GetPhoto(ctx context.Context, id int) (*entity.Photo, error) {
	var photo entity.Photo

	err := s.db.WithContext(ctx).First(&photo, id).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, storage.ErrPhotoNotFound
		}

		return nil, fmt.Errorf("error get photo: %w", err)
	}

	return &photo, nil
}

func (s *Storage) GetAllPhotos(ctx context.Context) ([]entity.Photo, error) {
	var photos []entity.Photo

	err := s.db.Find(&photos).Error

	if err != nil {
		return nil, fmt.Errorf("error get photos: %w", err)
	}

	return photos, nil
}

func (s *Storage) DeletePhoto(ctx context.Context, id int, ownerId int) error {
	err := s.db.Where("owner_id = ?", ownerId).Delete(entity.Photo{}, id).Error

	if err != nil {
		return fmt.Errorf("error delete photo: %w", err)
	}

	return nil
}

func (s *Storage) UpdatePhoto(ctx context.Context, id int, ownerId int, fields map[string]any) error {
  res := s.db.WithContext(ctx).Model(&entity.Photo{}).Where("photo_id = ?", id).Where("owner_id = ?", ownerId).Updates(fields)

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

func (s *Storage) CreateUser(ctx context.Context, user *entity.User) (int, error) {
	err := s.db.WithContext(ctx).Create(user).Error

	if err != nil {
		return 0, fmt.Errorf("error persist user entity: %w", err)
	}

	return user.UserId, nil
}

func (s *Storage) GetUserById(ctx context.Context, id int) (*entity.User, error) {
	var user entity.User

	err := s.db.WithContext(ctx).First(&user, id).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, storage.ErrUserNotFound
		}

		return nil, fmt.Errorf("error get user by id: %w", err)
	}

	return &user, nil
}


func (s *Storage) GetUserByEmail(ctx context.Context, email string) (*entity.User, error) {
	var user entity.User

	err := s.db.WithContext(ctx).Where("email = ?", email).First(&user).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, storage.ErrUserNotFound
		}

		return nil, fmt.Errorf("error get user by email: %w", err)
	}

	return &user, nil
}

func (s *Storage) GetUserByLogin(ctx context.Context, login string) (*entity.User, error) {
	var user entity.User

	err := s.db.WithContext(ctx).Where("login = ?", login).First(&user).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, storage.ErrUserNotFound
		}

		return nil, fmt.Errorf("error get user by email: %w", err)
	}

	return &user, nil
}

func (s *Storage) DeleteUser(ctx context.Context, id int) error {
	err := s.db.Delete(entity.User{}, id).Error

	if err != nil {
		return fmt.Errorf("error delete user: %w", err)
	}

	return nil
}

func (s *Storage) UpdateUser(ctx context.Context, id int, fields map[string]any) error {
  res := s.db.WithContext(ctx).Model(&entity.User{}).Where("user_id = ?", id).Updates(fields)

	if res.Error != nil {
		if errors.Is(res.Error, gorm.ErrRecordNotFound) {
			return storage.ErrPhotoNotFound
		}

		return fmt.Errorf("error update user: %w", res.Error)
	}

	if res.RowsAffected == 0 {
		return storage.ErrUserNotFound
	}

	return nil
}
