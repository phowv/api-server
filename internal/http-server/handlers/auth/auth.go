package auth

import (
	"encoding/json"
	"errors"
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
)

const tokenExpirationTime = 24 * time.Hour

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

func LoginUser(lg *slog.Logger, jwtSecret string, userService *service.UserService) http.HandlerFunc {
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

		expirationTime := time.Now().Add(tokenExpirationTime)
		claims := &auth.Claims{
			UserUuid: user.UserUuid,
			Role: user.Role,
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(expirationTime),
			},
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, err := token.SignedString([]byte(jwtSecret))
		if err != nil {
			log.Error("failed to create string token", sl.Err(err))
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"token": tokenString,
		})
	}
}
