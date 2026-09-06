package entity

import "github.com/google/uuid"

type CollectionPermittedUser struct {
	CollectionUuid uuid.UUID 	`gorm:"column:collection_uuid;primaryKey;type:uuid"`
	UserUuid 		uuid.UUID 		`gorm:"column:user_uuid;primaryKey;type:uuid"`
}

func (CollectionPermittedUser) TableName() string {
	return "collection.permitted_users"
}
