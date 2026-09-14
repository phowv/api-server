package entity

import (
	"time"

	"github.com/google/uuid"
)

type AccessModifier string

const (
	AccessModifierPrivate 	AccessModifier = "private"
	AccessModifierProtected AccessModifier = "protected"
	AccessModifierPublic 		AccessModifier = "public"
)

type Photo struct {
	PhotoUuid      			uuid.UUID				`gorm:"column:photo_uuid;primaryKey;type:uuid;default:uuid_generate_v4()"`
	OwnerUuid 					uuid.UUID 			`gorm:"column:owner_uuid;type:uuid"`
	Title        				string    			`gorm:"column:title"`
	Description  				string    			`gorm:"column:description"`
	RawFilename      		string    			`gorm:"column:filename"`
	MediumFilename			string    			`gorm:"column:medium_filename"`
	SmallFilename				string    			`gorm:"column:small_filename"`
	ModifiedDate 				time.Time 			`gorm:"column:modified_date;autoUpdateTime"`
	CreatedDate					time.Time 			`gorm:"column:created_at"`
	TookAt 							time.Time 			`gorm:"column:took_at"`
	Tags								[]Tag						`gorm:"many2many:photo.photo_tags;joinForeignKey:PhotoUuid;joinReferences:TagUuid;foreignKey:PhotoUuid;references:TagUuid"`
	AccessLevel					AccessModifier 	`gorm:"column:access_level;type:access_modifier;not null"`
}

func (Photo) TableName() string {
	return "photo.photo"
}

func CompareAccessLevels(a AccessModifier, b AccessModifier) int {
	return a.Int() - b.Int()
}

func (a AccessModifier) Int() int {
	switch a {
		case AccessModifierPrivate: return 3
		case AccessModifierProtected: return 2
		case AccessModifierPublic: return 1
		default: return 3
	}
}

func (a AccessModifier) Valid() bool {
	switch a {
	case AccessModifierPrivate, AccessModifierProtected, AccessModifierPublic:
		return true
	default:
		return false
	}
}
