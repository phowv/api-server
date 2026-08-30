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

type AuthRepository struct {
	*Storage
}

func NewAuthReposotory(st *Storage) *AuthRepository {
	return &AuthRepository{Storage: st}
}

func (s *AuthRepository) SaveSession(ctx context.Context, refreshToken *entity.Session) (uuid.UUID, error) {
	err := s.db.WithContext(ctx).Create(refreshToken).Error

	if err != nil {
		return uuid.Nil, fmt.Errorf("error persist session entity: %w", err)
	}

	return refreshToken.SessionUuid, nil
}

func (s *AuthRepository) GetValidSessionByUuid(ctx context.Context, sessionUuid uuid.UUID) (*entity.Session, error) {
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

func (s *AuthRepository) RevokeSessionByUuid(ctx context.Context, sessionUuid uuid.UUID) error {
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

func (s *AuthRepository) SaveVerificationCode(ctx context.Context, verificationCode *entity.VerificationCode) error {
	err := s.db.WithContext(ctx).Create(verificationCode).Error

	if err != nil {
		return fmt.Errorf("error persist verification code entity: %w", err)
	}

	return nil
}

func (s *AuthRepository) DeleteAllVerificationCodesByUserUuid(ctx context.Context, userUuid uuid.UUID) error {
	err := s.db.Where("user_uuid = ?", userUuid).Delete(entity.VerificationCode{}).Error

	if err != nil {
		return fmt.Errorf("error delete verification codes: %w", err)
	}

	return nil
}

func (s *AuthRepository) GetValidVerificationCodeByUserUuid(ctx context.Context, userUuid uuid.UUID) (*entity.VerificationCode, error) {
	var verificationCode entity.VerificationCode

	now := time.Now()
	res := s.db.WithContext(ctx).Where("user_uuid = ?", userUuid).Where("expires_at > ?", now).First(&verificationCode)

	if res.Error != nil {
		if errors.Is(res.Error, gorm.ErrRecordNotFound) {
			return nil, storage.ErrVerificationCodeNotFound
		}

		return nil, fmt.Errorf("error get sessions: %w", res.Error)
	}

	return &verificationCode, nil
}

