package entity

import (
	"time"

	"github.com/google/uuid"
)

type UserRole string

const (
	UserDefaultRole UserRole = "user"
	UserModeratorRole UserRole = "moderator"
	UserAdminRole UserRole = "admin"
)

type User struct {
	UserUuid     				uuid.UUID `gorm:"column:user_uuid;primaryKey;type:uuid;default:uuid_generate_v4()"`
	Login        				string    `gorm:"column:login"`
	HashPassword				string		`gorm:"column:hash_password"`
	Email								string		`gorm:"column:email"`
	Role								UserRole	`gorm:"column:role;type:user_role;not null"`
	Description  				string    `gorm:"column:description"`
	CreateDate					time.Time `gorm:"column:created_at"`
	IsActive						bool 			`gorm:"column:is_active"`
	PhotosQuota					int				`gorm:"column:photos_quota"`
	CollectionsQuota		int				`gorm:"column:collections_quota"`
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
