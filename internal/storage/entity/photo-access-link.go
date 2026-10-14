package entity

import (
	"time"

	"github.com/google/uuid"
)

type PhotoAccessLink struct {
	PhotoUuid 	uuid.UUID `gorm:"column:photo_uuid;primaryKey;type:uuid"`
	HashCode 	string			`gorm:"column:hash_code"`
	CreatedAt 	time.Time `gorm:"column:created_at"`
	EpiresAt 	time.Time 	`gorm:"column:expires_at"`
	IsRevoked	bool				`gorm:"is_revoked"`
}

func (PhotoAccessLink) TableName() string {
	return "photo.access_link"
}
