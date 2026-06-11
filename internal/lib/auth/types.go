package auth

import (
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type Claims struct {
	UserUuid uuid.UUID `json:"user_uuid"`
	Role string `json:"user_role"`
	jwt.RegisteredClaims
}

type RefreshClaims struct {
	UserUuid uuid.UUID `json:"user_uuid"`
	jwt.RegisteredClaims
}
