package entity

import "github.com/google/uuid"

type CollectionPhotoEntity struct {
	CollectionUuid uuid.UUID `gorm:"column:collection_uuid"`
	PhotoUuid uuid.UUID `gorm:"column:photo_uuid"`
}

func (CollectionPhotoEntity) TableName() string {
	return "collection.collection_photos"
}
