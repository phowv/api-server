package auth

import (
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type Claims struct {
	UserUuid uuid.UUID `json:"uid"`
	Role string `json:"ur"`
	SessionUuid uuid.UUID `json:"sid"`
	jwt.RegisteredClaims
}

type RefreshClaims struct {
	UserUuid uuid.UUID `json:"uid"`
	SessionUuid uuid.UUID `json:"sid"`
	jwt.RegisteredClaims
}
