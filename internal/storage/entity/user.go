package entity

import "github.com/google/uuid"

type User struct {
	UserUuid     				uuid.UUID `gorm:"column:user_uuid;primaryKey;type:uuid;default:uuid_generate_v4()"`
	Login        				string    `gorm:"column:login"`
	HashPassword				string		`gorm:"column:hash_password"`
	Email								string		`gorm:"column:email"`
	Role								string		`gorm:"column:role"`
	Description  				string    `gorm:"column:description"`
}

func (User) TableName() string {
	return "users.users"
}
