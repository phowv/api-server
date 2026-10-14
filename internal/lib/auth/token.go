package auth

import (
	"context"
	"errors"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidToken = errors.New("token is invalid")
)

func ParseAccessToken(tokenString string, secret []byte) (*Claims, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected singing method: %v", token.Header["alg"])
		}

		return secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Name}))

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

func ApplyAccessTokenClaims(ctx context.Context, claims *Claims) context.Context {
	ctx = context.WithValue(ctx, "user_uuid", claims.UserUuid)
	ctx = context.WithValue(ctx, "user_role", claims.Role)
	ctx = context.WithValue(ctx, "session_uuid", claims.SessionUuid)

	return ctx
}
