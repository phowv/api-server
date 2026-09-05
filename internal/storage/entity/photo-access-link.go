package entity

import (
	"time"

	"github.com/google/uuid"
)

type PhotoAccessLink struct {
	PhotoUuid 	uuid.UUID 	`gorm:"column:photo_uuid;primaryKey;type:uuid;default:uuid_generate_v4()"`
	hash_code 	string			`gorm:"column:hash_code"`
	created_at 	time.Time 	`gorm:"column:created_at"`
	expires_at 	time.Time 	`gorm:"column:expires_at"`
	is_revoked	bool				`gorm:"is_revoked"`
}

func (PhotoAccessLink) TableName() string {
	return "photo.access_link"
}
