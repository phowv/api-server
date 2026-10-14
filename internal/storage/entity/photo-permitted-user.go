package entity

import "github.com/google/uuid"

type PhotoPermittedUser struct {
	PhotoUuid 	uuid.UUID 	`gorm:"column:photo_uuid;primaryKey;type:uuid"`
	UserUuid 		uuid.UUID 	`gorm:"column:user_uuid;primaryKey;type:uuid"`
}

func (PhotoPermittedUser) TableName() string {
	return "photo.permitted_users"
}
