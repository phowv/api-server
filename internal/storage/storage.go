package storage

import "errors"

var (
	ErrPhotoNotFound = errors.New("photo not found")
	ErrUserNotFound = errors.New("user not found")
)
