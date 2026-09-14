package entity

import "github.com/google/uuid"

type PhotoTagEntity struct {
	PhotoUuid uuid.UUID `gorm:"column:photo_uuid"`
	TagUuid uuid.UUID `gorm:"column:tag_uuid"`
}

func (PhotoTagEntity) TableName() string {
	return "photo.photo_tags"
}
