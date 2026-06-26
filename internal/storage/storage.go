package storage

import "errors"

var (
	ErrPhotoNotFound = errors.New("photo not found")
	ErrUserNotFound = errors.New("user not found")
	ErrSessionNotFound = errors.New("session not found")
	ErrVerificationCodeNotFound = errors.New("verification code not found")
)
