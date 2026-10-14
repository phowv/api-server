package entity

import (
	"time"

	"github.com/google/uuid"
)

type Session struct {
	SessionUuid					uuid.UUID `gorm:"column:session_uuid;primaryKey;type:uuid"`
	UserUuid						uuid.UUID `gorm:"column:user_uuid;type:uuid"`
	HashToken						string 		`gorm:"column:hash_token"`
	ExpiresAt						time.Time `gorm:"column:expires_at"`
	IsRevoked						bool 			`gorm:"column:is_revoked"`
}

func (Session) TableName() string {
	return "sessions.sessions"	
}
