package postrgesql

import (
	"context"
	"errors"
	"fmt"
	"photo-viewer-server/internal/storage"
	"photo-viewer-server/internal/storage/entity"
	"time"

	"github.com/google/uuid"
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

func (s *Storage) SavePhoto(ctx context.Context, photo *entity.Photo) (uuid.UUID, error) {
	err := s.db.WithContext(ctx).Create(photo).Error

	if err != nil {
		return uuid.Nil, fmt.Errorf("error persist photo entity: %w", err)
	}

	return photo.PhotoUuid, nil
}

func (s *Storage) GetPhoto(ctx context.Context, uuid uuid.UUID) (*entity.Photo, error) {
	var photo entity.Photo

	err := s.db.WithContext(ctx).First(&photo, uuid).Error

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

func (s *Storage) GetAllPhotosByOwner(ctx context.Context, ownerUuid uuid.UUID) ([]entity.Photo, error) {
	var photos []entity.Photo

	err := s.db.Where("owner_uuid = ?", ownerUuid).Find(&photos).Error

	if err != nil {
		return nil, fmt.Errorf("error get photos by owner: %w", err)
	}

	return photos, nil
}

func (s *Storage) DeletePhoto(ctx context.Context, uuid uuid.UUID, ownerUuid uuid.UUID) error {
	err := s.db.Where("owner_uuid = ?", ownerUuid).Delete(entity.Photo{}, uuid).Error

	if err != nil {
		return fmt.Errorf("error delete photo: %w", err)
	}

	return nil
}

func (s *Storage) UpdatePhoto(ctx context.Context, uuid uuid.UUID, ownerUuid uuid.UUID, fields map[string]any) error {
  res := s.db.WithContext(ctx).Model(&entity.Photo{}).Where("photo_uuid = ?", uuid).Where("owner_uuid = ?", ownerUuid).Updates(fields)

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

func (s *Storage) CreateUser(ctx context.Context, user *entity.User) (uuid.UUID, error) {
	err := s.db.WithContext(ctx).Create(user).Error

	if err != nil {
		return uuid.Nil, fmt.Errorf("error persist user entity: %w", err)
	}

	return user.UserUuid, nil
}

func (s *Storage) GetUserByUuid(ctx context.Context, uuid uuid.UUID) (*entity.User, error) {
	var user entity.User

	err := s.db.WithContext(ctx).First(&user, uuid).Error

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

func (s *Storage) DeleteUser(ctx context.Context, uuid uuid.UUID)error {
	err := s.db.Delete(entity.User{}, uuid).Error

	if err != nil {
		return fmt.Errorf("error delete user: %w", err)
	}

	return nil
}

func (s *Storage) UpdateUser(ctx context.Context, uuid uuid.UUID, fields map[string]any) error {
  res := s.db.WithContext(ctx).Model(&entity.User{}).Where("user_uuid = ?", uuid).Updates(fields)

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

func (s *Storage) SaveSession(ctx context.Context, refreshToken *entity.Session) (uuid.UUID, error) {
	err := s.db.WithContext(ctx).Create(refreshToken).Error

	if err != nil {
		return uuid.Nil, fmt.Errorf("error persist session entity: %w", err)
	}

	return refreshToken.SessionUuid, nil
}

func (s *Storage) GetValidSessionByUuid(ctx context.Context, sessionUuid uuid.UUID) (*entity.Session, error) {
	var session entity.Session

	now := time.Now()
	res := s.db.WithContext(ctx).Where("session_uuid = ?", sessionUuid).Where("is_revoked = FALSE").Where("expires_at > ?", now).First(&session)

	if res.Error != nil {
		if errors.Is(res.Error, gorm.ErrRecordNotFound) {
			return nil, storage.ErrSessionNotFound
		}

		return nil, fmt.Errorf("error get sessions: %w", res.Error)
	}

	return &session, nil
}

func (s *Storage) RevokeSessionByUuid(ctx context.Context, sessionUuid uuid.UUID) error {
	res := s.db.WithContext(ctx).Model(&entity.Session{}).Where("session_uuid = ?", sessionUuid).Update("is_revoked", true)

	if res.Error != nil {
		if errors.Is(res.Error, gorm.ErrRecordNotFound) {
			return storage.ErrSessionNotFound
		}

		return fmt.Errorf("error get session: %w", res.Error)
	}

	if res.RowsAffected == 0 {
		return storage.ErrSessionNotFound
	}

	return nil
}

func (s *Storage) Ping(ctx context.Context) error {
	db, err := s.db.DB()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, 2 * time.Second)
	defer cancel()
	return db.PingContext(ctx)
}

func (s *Storage) Name() string {
	return "postgres"
}
