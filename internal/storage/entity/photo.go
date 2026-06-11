package entity

import (
	"time"

	"github.com/google/uuid"
)

type Photo struct {
	PhotoUuid      			uuid.UUID	`gorm:"column:photo_uuid;primaryKey;type:uuid;default:uuid_generate_v4()"`
	OwnerUuid 					uuid.UUID `gorm:"column:owner_uuid;type:uuid"`
	Title        				string    `gorm:"column:title"`
	Description  				string    `gorm:"column:description"`
	Tags								string		`gorm:"column:tags"`
	Filename         		string    `gorm:"column:filename"`
	ModifiedDate 				time.Time `gorm:"column:modified_date;autoUpdateTime"`
	CreatedDate					time.Time `gorm:"column:created_at"`
	TookAt 							time.Time `gorm:"column:took_at"`
}

func (Photo) TableName() string {
	return "photo.photo"
}
