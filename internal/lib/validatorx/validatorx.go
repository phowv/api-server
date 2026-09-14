package validatorx

import (
	"photo-viewer-server/internal/storage/entity"

	"github.com/go-playground/validator/v10"
)

func NewValidator() *validator.Validate {
	v := validator.New()

	v.RegisterValidation("access_modifier", func(fl validator.FieldLevel) bool {
		val := fl.Field().String()
		return val == string(entity.AccessModifierPrivate) ||
			val == string(entity.AccessModifierProtected) ||
			val == string(entity.AccessModifierPublic)
	})

	return v
}
