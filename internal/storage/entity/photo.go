package entity

import (
	"time"
)

type Photo struct {
	PhotoId      				int       `gorm:"column:photo_id;primaryKey"`
	OwnerId 						int				`gorm:"column:owner_id"`
	Title        				string    `gorm:"column:title"`
	Description  				string    `gorm:"column:description"`
	Tags								string		`gorm:"column:tags"`
	Filename         		string    `gorm:"column:filename"`
	ModifiedDate 				time.Time `gorm:"column:modified_date;autoUpdateTime"`
	CreatedAt 					time.Time `gorm:"column:created_at"`
	TookAt 							time.Time `gorm:"column:took_at"`
}

func (Photo) TableName() string {
	return "photo.photo"
}
