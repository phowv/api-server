package entity

import (
	"time"

	"github.com/google/uuid"
)

type Collection struct {
	CollectionUuid      uuid.UUID				`gorm:"column:collection_uuid;primaryKey;type:uuid;default:uuid_generate_v4()"`
	OwnerUuid 					uuid.UUID 			`gorm:"column:owner_uuid;type:uuid"`
	Title        				string    			`gorm:"column:title"`
	Description  				string    			`gorm:"column:description"`
	CreatedDate					time.Time 			`gorm:"column:created_at;autoUpdateTime"`
	Photos							[]Photo 				`gorm:"many2many:collection.collection_photos;joinForeignKey:CollectionUuid;joinReferences:PhotoUuid;foreignKey:CollectionUuid;references:PhotoUuid"`
	AccessLevel					AccessModifier 	`gorm:"column:access_level;type:access_modifier;not null"`
}

func (Collection) TableName() string {
	return "collection.collection"
}
