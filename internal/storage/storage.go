package storage

import (
	"context"
	"errors"
)

type TxManager interface {
	WithTransaction(ctx context.Context, tFunc func(txCtx context.Context) error) error
}

var (
	ErrPhotoNotFound = errors.New("photo not found")
	ErrUserNotFound = errors.New("user not found")
	ErrSessionNotFound = errors.New("session not found")
	ErrVerificationCodeNotFound = errors.New("verification code not found")
	ErrUserQuotaIsNotEnough = errors.New("user quota is not enough")
	ErrTagNotFound = errors.New("tag not found")
	ErrTagAlreadyExists = errors.New("tag already exists")
)
