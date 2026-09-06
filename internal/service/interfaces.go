package service

import (
	"context"
	"photo-viewer-server/internal/lib/signer"
	"photo-viewer-server/internal/storage/entity"

	"github.com/google/uuid"
)

type Healthchecker interface {
	Name() string
	Ping(ctx context.Context) error
}

type PhotoRepo interface {
	SavePhoto(ctx context.Context, photo *entity.Photo) (uuid.UUID, error)
	GetAllPhotos(ctx context.Context) ([]entity.Photo, error)
  GetPhotosByOwner(ctx context.Context, ownerUuid uuid.UUID) ([]entity.Photo, error)
  GetAllPhotosByOwner(ctx context.Context, ownerUuid uuid.UUID) ([]entity.Photo, error)
	GetPhoto(ctx context.Context, uuid uuid.UUID) (*entity.Photo, error)
	DeletePhoto(ctx context.Context, uuid uuid.UUID, ownerUuid uuid.UUID) error
  UpdatePhoto(ctx context.Context, uuid uuid.UUID, ownerUuid uuid.UUID, fields map[string]any) error
}

type TagRepo interface {
	SaveTag(ctx context.Context, tag *entity.Tag) (uuid.UUID, error)
	GetTag(ctx context.Context, uuid uuid.UUID) (*entity.Tag, error)
	GetAllTags(ctx context.Context) ([]entity.Tag, error)
	GetTagsByPhoto(ctx context.Context, photoUuid uuid.UUID) ([]entity.Tag, error)
	GetTagByName(ctx context.Context, tagName string) (*entity.Tag, error)
}

type FileRepo interface {
	SaveFile(ctx context.Context, bucketName string, objectName string, data []byte, contentType string) (string, error)
	GetFile(ctx context.Context, bucketName string, objectName string) ([]byte, string, error)
	DeleteFile(ctx context.Context, bucketName string, objectName string) error
	CreateBucket(ctx context.Context, bucketName string) error
}

type CollectionRepo interface {
	SaveCollection(ctx context.Context, collection *entity.Collection) (uuid.UUID, error)
	GetCollection(ctx context.Context, collectionUuid uuid.UUID) (*entity.Collection, error)
	GetAllCollections(ctx context.Context) ([]entity.Collection, error)
	GetCollectionsByOwner(ctx context.Context, ownerUuid uuid.UUID) ([]entity.Collection, error)
	GetAllCollectionsByOwner(ctx context.Context, ownerUuid uuid.UUID) ([]entity.Collection, error)
	DeleteCollection(ctx context.Context, collectionUuid, ownerUuid uuid.UUID) error
	AddPhotoToCollection(ctx context.Context, collectionUuid, photoUuid uuid.UUID) error
	RemovePhotoFromCollection(ctx context.Context, collectionUuid, photoUuid uuid.UUID) error
}

type UserRepo interface {
	CreateUser(ctx context.Context, user *entity.User) (uuid.UUID, error)
	GetUserByUuid(ctx context.Context, uuid uuid.UUID) (*entity.User, error)
	GetUserByEmail(ctx context.Context, email string) (*entity.User, error)
	GetUserByLogin(ctx context.Context, login string) (*entity.User, error)
	DeleteUser(ctx context.Context, uuid uuid.UUID) error
	UpdateUser(ctx context.Context, uuid uuid.UUID, fields map[string]any) error
	DecrementPhotosQuotaByUuid(ctx context.Context, uuid uuid.UUID) error
	DecrementCollectionsQuotaByUuid(ctx context.Context, uuid uuid.UUID) error
}

type AccessRepo interface {
	GetValidAccessLinkByPhotoUuid(ctx context.Context, photoUuid uuid.UUID) (*entity.PhotoAccessLink, error)
	IsUserCanAccessPhotoByUuid(ctx context.Context, photoUuid uuid.UUID, userUuid uuid.UUID) (bool, error)
	GetValidAccessLinkByCollectionUuid(ctx context.Context, collectionUuid uuid.UUID) (*entity.CollectionAccessLink, error)
	IsUserCanAccessCollectionByUuid(ctx context.Context, collectionUuid uuid.UUID, userUuid uuid.UUID) (bool, error)
}

type ImageProcessor interface {
  ResizeAndCompress(ctx context.Context, rawImage []byte, maxWidth, maxHeight int, quality int) ([]byte, error)
}

type PhotoKeySigner interface {
  Sign(photoUuid, ownerUuid uuid.UUID, photoRaw, photoMedium, photoSmall string) (string, error)
	Validate(token string, expectedPhotoUuid uuid.UUID) (*signer.FileKeyPayload, error)
}

type SessionRepo interface {
	SaveSession(ctx context.Context, refreshToken *entity.Session) (uuid.UUID, error)
	GetValidSessionByUuid(ctx context.Context, session uuid.UUID) (*entity.Session, error)
	RevokeSessionByUuid(ctx context.Context, sessionUuid uuid.UUID) error
}

type VerificationCodeRepo interface {
	SaveVerificationCode(ctx context.Context, verificationCode *entity.VerificationCode) error
	DeleteAllVerificationCodesByUserUuid(ctx context.Context, userUuid uuid.UUID) error
  GetValidVerificationCodeByUserUuid(ctx context.Context, userUuid uuid.UUID) (*entity.VerificationCode, error)
}
