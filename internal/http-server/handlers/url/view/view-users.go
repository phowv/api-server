package view

import (
	"log/slog"
	"net/http"
	"photo-viewer-server/internal/lib/api/response"
	"photo-viewer-server/internal/lib/logger/sl"
	"photo-viewer-server/internal/service"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/google/uuid"
)

type UserInfo struct {
	UserUuid uuid.UUID `json:"user_uuid"`
	Role string `json:"role"`
	Login string `json:"user_login"`
	Email string `json:"user_email"`
	PhotosQuota int `json:"photos_quota"`
	CollectionsQuota int `json:"collections_quota"`
	IsActive bool `json:"is_active"`
	CreateDate time.Time `json:"created_at"`
}

type UsersInfoResponse struct {
	response.Response
	Users []UserInfo `json:"users"`
}

func ViewUsers(lg *slog.Logger, adminService *service.AdminService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := lg.With(
			slog.String("op", "handlers.view.ViewUsers"),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		users, err := adminService.GetUsers(r.Context())

		if err != nil {
			log.Error("failed to get all users", sl.Err(err))

			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, response.Error("internal error"))
			return
		}

		usersInfo := make([]UserInfo, len(users))

		for i, user := range users {
			usersInfo[i] = UserInfo{
				UserUuid: user.UserUuid,
				Role: user.Role,
				Login: user.Login,
				Email: user.Email,
				PhotosQuota: user.PhotosQuota,
				CollectionsQuota: user.CollectionsQuota,
				IsActive: user.IsActive,
				CreateDate: user.CreateDate,
			}
		}

		render.Status(r, http.StatusOK)
		render.JSON(w, r, &UsersInfoResponse{
			Response: response.OK(),
			Users: usersInfo,
		})
	}
}
