package entity

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	UserUuid     				uuid.UUID `gorm:"column:user_uuid;primaryKey;type:uuid;default:uuid_generate_v4()"`
	Login        				string    `gorm:"column:login"`
	HashPassword				string		`gorm:"column:hash_password"`
	Email								string		`gorm:"column:email"`
	Role								string		`gorm:"column:role"`
	Description  				string    `gorm:"column:description"`
	CreateDate					time.Time `gorm:"column:created_at"`
	IsActive						bool 			`gorm:"column:is_active"`
}

func (User) TableName() string {
	return "users.users"
}

type VerificationCode struct {
	UserUuid						uuid.UUID `gorm:"column:user_uuid;primaryKey;type:uuid"`
	HashCode					  string 		`gorm:"column:hash_code"`
	CreatedDate					time.Time `gorm:"column:created_at"`
	ExpiresAt						time.Time `gorm:"column:expires_at"`
}

func (VerificationCode) TableName() string {
	return "users.codes"
}
