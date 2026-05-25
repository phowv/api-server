package entity

import (
	"time"
)

type Photo struct {
	PhotoId      int       `gorm:"column:photo_id;primaryKey"`
	Title        string    `gorm:"column:title"`
	Description  string    `gorm:"column:description"`
	Filename         string    `gorm:"column:filename"`
	ModifiedDate time.Time `gorm:"column:modified_date;autoUpdateTime"`
}

func (Photo) TableName() string {
	return "photo.photo"
}
