package auth

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"photo-viewer-server/internal/lib/api/response"
	"photo-viewer-server/internal/lib/auth"
	"photo-viewer-server/internal/lib/logger/sl"
	"photo-viewer-server/internal/service"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type UserInfoResponse struct {
	Login string `json:"user_login"`
	Email string `json:"user_email"`
	Role  string `json:"user_role"`
}

type AccessTokenResponse struct {
	response.Response
	AccessToken string `json:"access_token"`
}

type createJwtTokensResult struct {
	tokenString   string
	refreshCookie *http.Cookie
}

// GetMe returns the authenticated user's profile.
//
//	@Summary		Get current user
//	@Tags			  auth
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200			{object}	auth.UserInfoResponse	"user_login, user_email, user_role"
//	@Failure		401			{object}	response.Response	"token is empty / invalid token"
//	@Failure		500			{object}	response.Response	"failed to get user info (incl. inactive user)"
//	@Router			/auth/me [get]
func GetMe(lg *slog.Logger, userService *service.UserService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := lg.With(
			slog.String("op", "handlers.auth.GetMe"),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		userUuid, ok := r.Context().Value("user_uuid").(uuid.UUID)
		if !ok {
			log.Error("failed to get user_uuid from context")

			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, response.Error("invalid user_uuid"))
			return
		}

		log.Info("get user info", slog.Any("user_uuid", userUuid))

		user, err := userService.GetUserInfo(r.Context(), userUuid)

		if err != nil {
			log.Error("failed to get user info", sl.Err(err))

			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, response.Error("failed to get user info"))
			return
		}

		render.Status(r, http.StatusOK)
		render.JSON(w, r, UserInfoResponse{
			Login: user.Login,
			Email: user.Email,
			Role:  user.Role,
		})
	}
}

func createJwtTokens(
	ctx context.Context, userService *service.UserService, user *service.User, apiPrefix string, jwtAccessSecret, jwtRefreshSecret string, authConfig *auth.AuthConfig,
) (*createJwtTokensResult, error) {
	expirationTime := time.Now().Add(authConfig.JwtAccessExpires)
	sessionUuid := uuid.New()

	claims := &auth.Claims{
		UserUuid:    user.UserUuid,
		Role:        user.Role,
		SessionUuid: sessionUuid,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(jwtAccessSecret))
	if err != nil {
		return nil, fmt.Errorf("failed to create string access token: %w", err)
	}

	refreshExpirarionTime := time.Now().Add(authConfig.JwtRefreshExpires)
	refreshClaims := &auth.RefreshClaims{
		UserUuid:    user.UserUuid,
		SessionUuid: sessionUuid,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(refreshExpirarionTime),
		},
	}
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS512, refreshClaims)
	refreshTokenString, err := refreshToken.SignedString([]byte(jwtRefreshSecret))
	if err != nil {
		return nil, fmt.Errorf("failed to create string refresh token: %w", err)
	}

	err = userService.CreateSession(ctx, sessionUuid, user.UserUuid, refreshTokenString, refreshExpirarionTime)

	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	sameSite := http.SameSiteStrictMode
	if authConfig.IsDevEnv {
		sameSite = http.SameSiteNoneMode
	}

	cookie := &http.Cookie{
		Name:     "refresh_token",
		Value:    refreshTokenString,
		Path:     apiPrefix + "/auth/refresh",
		HttpOnly: true,
		Secure:   true,
		Expires:  refreshExpirarionTime,
		SameSite: sameSite,
	}

	return &createJwtTokensResult{
		tokenString:   tokenString,
		refreshCookie: cookie,
	}, nil
}

func sendJwtTokens(w http.ResponseWriter, r *http.Request, tokens *createJwtTokensResult) {
	http.SetCookie(w, tokens.refreshCookie)

	render.JSON(w, r, AccessTokenResponse{
		Response:    response.OK(),
		AccessToken: tokens.tokenString,
	})
}
