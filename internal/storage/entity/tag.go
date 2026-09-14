package entity

import (
	"time"

	"github.com/google/uuid"
)

type Tag struct {
	TagUuid 			uuid.UUID `gorm:"column:tag_uuid;primaryKey;type:uuid;default:uuid_generate_v4()"`
	Name					string		`gorm:"column:tag_name"`
	Description 	string		`gorm:"column:tag_description"`
	ModifiedDate 	time.Time `gorm:"column:modified_date;autoUpdateTime"`
}

func (Tag) TableName() string {
	return "photo.tag"
}
