package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"photo-viewer-server/internal/lib/api/response"
	"photo-viewer-server/internal/lib/auth"
	"photo-viewer-server/internal/lib/logger/sl"
	"photo-viewer-server/internal/service"
	"strings"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const accessTokenExpirationTime = 15 * time.Minute
const refreshTokenExpirationTime = 30 * 24 * time.Hour

type userInfoResponse struct {
	Login string `json:"user_login"`
	Email string `json:"user_email"`
}

func RegisterUser(lg *slog.Logger, userService *service.UserService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := lg.With(
			slog.String("op", "handlers.auth.RegisterUser"),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		var userData service.UserData
		if err := json.NewDecoder(r.Body).Decode(&userData); err != nil {
			log.Error("failed to decode metadata", sl.Err(err))

			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, response.Error("invalid metadata"))
			return
		}

		log.Debug("request metadata decoded", slog.String("login", userData.Login), slog.String("email", userData.Email))

		_, err := userService.CreateUser(r.Context(), userData)

		if err != nil {
			log.Error("error create user", sl.Err(err))

			if errors.Is(err, service.ErrUserExists) {
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, response.Error("error user already exists"))
				return

			} else if errors.Is(err, service.ErrUserPasswordTooShort) {
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, response.Error("error user password too short, must be longer than 8 symbols"))
				return
			}

			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, response.Error("error register new user"))
			return
		}

		render.Status(r, http.StatusCreated)
		render.JSON(w, r, response.OK())
	}
}

func LoginUser(lg *slog.Logger, jwtAccessSecret string, jwtRefreshSecret string, userService *service.UserService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := lg.With(
			slog.String("op", "handlers.auth.LoginUser"),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		var userCredentials service.UserAuthCredentials
		if err := json.NewDecoder(r.Body).Decode(&userCredentials); err != nil {
			log.Error("failed to decode metadata", sl.Err(err))

			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, response.Error("invalid metadata"))
			return
		}

		log.Debug("request metadata decoded", slog.String("login", userCredentials.Login))

		user, err := userService.AuthenticateUser(r.Context(), userCredentials)

		if err != nil {
			log.Error("failed to authenticate user", sl.Err(err))
			http.Error(w, "invalid credentials", http.StatusForbidden)	
			return
		}

	  tokens, err := createJwtTokens(r.Context(), userService, user, jwtAccessSecret, jwtRefreshSecret)
		if err != nil {
			log.Error("failed to create jwt token pair", sl.Err(err))
			
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, response.Error("failed to create tokens"))
			return
		}

		sendJwtTokens(w, tokens)
	}
}

func GetMe(lg *slog.Logger, userService *service.UserService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := lg.With(
			slog.String("op", "handlers.auth.GetMe"),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		userUuid := r.Context().Value("user_uuid").(uuid.UUID)

		log.Info("get user info", slog.Any("user_uuid", userUuid))

		user, err := userService.GetUserInfo(r.Context(), userUuid)

		if err != nil {
			log.Error("failed to get user info", sl.Err(err))

			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, response.Error("failed to get user info"))
			return
		}

		render.Status(r, http.StatusOK)
		render.JSON(w, r, userInfoResponse{
			Login: user.Login,
			Email: user.Email,
		})
	}
}

func RefreshUser(lg *slog.Logger, jwtAccessSecret string, jwtRefreshSecret string, userService *service.UserService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := lg.With(
			slog.String("op", "handlers.auth.RefreshUser"),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)
	
		authHeader := r.Header.Get("Authorization")

		if authHeader == "" {
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, response.Error("token is empty"))
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, response.Error("invalid header format"))
			return
		}

		tokenString := parts[1]
		claims := &auth.Claims{}

		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return []byte(jwtAccessSecret), nil
		})

		if err != nil {
			log.Debug("failed to parse accsess token", sl.Err(err))

	    render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, response.Error("invalid token"))
			return 
		}

		if token.Valid {
			log.Debug("token is valid yet")

		   render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, response.Error("token is valid yet"))
			return
		}

		userUuid := claims.UserUuid

		log.Debug("parsed user uuid", slog.Any("user_uuid", userUuid))

		cookie, err := r.Cookie("refresh_token")
		if err != nil {
			log.Debug("missing refresh token cookie")

	    render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, response.Error("missing authentication"))
			return 
		}

		log.Info("parsed cookie refresh token")

		refreshTokenString := cookie.Value
		refreshClaims := &auth.RefreshClaims{}

		refreshToken, err := jwt.ParseWithClaims(refreshTokenString, refreshClaims, func(token *jwt.Token) (interface{}, error) {
			return []byte(jwtRefreshSecret), nil
		})

		if err != nil || !refreshToken.Valid {
			log.Debug("invalid refresh token")

	    render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, response.Error("invalid token"))
			return 
		}
		
		user, err := userService.AuthenticateSession(r.Context(), userUuid, refreshTokenString)
		if err != nil {
			log.Error("failed to authenticate session", sl.Err(err))

	    render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, response.Error("invalid token"))
			return 
		}

		tokens, err := createJwtTokens(r.Context(), userService, user, jwtAccessSecret, jwtRefreshSecret)	
		if err != nil {
			log.Error("failed to create jwt token pair", sl.Err(err))
			
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, response.Error("failed to create tokens"))
			return
		}

		sendJwtTokens(w, tokens)
	}
}

type createJwtTokensResult struct {
	tokenString string
	refreshCookie *http.Cookie
}

func createJwtTokens(ctx context.Context, userService *service.UserService, user *service.User, jwtAccessSecret, jwtRefreshSecret string) (*createJwtTokensResult, error) {
	expirationTime := time.Now().Add(accessTokenExpirationTime)
	claims := &auth.Claims{
		UserUuid: user.UserUuid,
		Role: user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(jwtAccessSecret))
	if err != nil {
		return nil, fmt.Errorf("failed to create string access token: %w", err)
	}

	refreshExpirarionTime := time.Now().Add(refreshTokenExpirationTime)
	refreshClaims := &auth.RefreshClaims{
		UserUuid: user.UserUuid,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(refreshExpirarionTime),
		},
	}
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS512, refreshClaims)
	refreshTokenString, err := refreshToken.SignedString([]byte(jwtRefreshSecret))
	if err != nil {
		return nil, fmt.Errorf("failed to create string refresh token: %w", err)
	}

	_, err = userService.CreateSession(ctx, user.UserUuid, refreshTokenString, refreshExpirarionTime)

	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	cookie := &http.Cookie{
		Name: "refresh_token",
		Value: refreshTokenString,
		Path: "/auth/refresh",
		HttpOnly: true,
		Secure: true,
		Expires: refreshExpirarionTime,
		SameSite: http.SameSiteStrictMode,
	}

	return &createJwtTokensResult{
		tokenString: tokenString,
		refreshCookie: cookie,
	}, nil
}

func sendJwtTokens(w http.ResponseWriter, tokens *createJwtTokensResult) {
	http.SetCookie(w, tokens.refreshCookie)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"access_token": tokens.tokenString,
	})
}
