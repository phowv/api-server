package entity

import (
	"time"

	"github.com/google/uuid"
)

type CollectionAccessLink struct {
	CollectionUuid uuid.UUID 	`gorm:"column:collection_uuid;primaryKey;type:uuid"`
	HashCode 	string					`gorm:"column:hash_code"`
	CreatedAt 	time.Time 		`gorm:"column:created_at"`
	EpiresAt 	time.Time 			`gorm:"column:expires_at"`
	IsRevoked	bool						`gorm:"is_revoked"`
}

func (CollectionAccessLink) TableName() string {
	return "collection.access_link"
}
