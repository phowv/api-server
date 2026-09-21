package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const (
	roleAdmin = 3
	roleModerator = 2
	roleUser = 1
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

type AuthConfig struct {
	IsDevEnv bool
	JwtAccessExpires time.Duration
	JwtRefreshExpires time.Duration
}
