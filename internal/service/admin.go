package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"photo-viewer-server/internal/lib/logger/sl"
	"photo-viewer-server/internal/storage"
	"photo-viewer-server/internal/storage/entity"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
)

type PatchUserRequest struct {
	Role *entity.UserRole `json:"user_role,omitempty" validate:"omitempty,oneof=user moderator"`
	PhotosQuota *int `json:"photos_quota,omitempty" validate:"omitempty,min=0"`
	CollectionsQuota *int `json:"collections_quota,omitempty" validate:"omitempty,min=0"`
}

type UserFullInformation struct {
	User
	PhotosQuota int
	CollectionsQuota int
	IsActive bool
	CreateDate time.Time
}

type AdminService struct {
	log *slog.Logger
	adminUserRepo AdminUserRepo
	txManager storage.TxManager
}

func NewAdminService(
	log *slog.Logger,
	adminUserRepo AdminUserRepo,
	txManager storage.TxManager,
) *AdminService {
	return &AdminService{
		log: log,
		adminUserRepo: adminUserRepo,
		txManager: txManager,
	}
}

func (s *AdminService) GetUsers(ctx context.Context) ([]UserFullInformation, error) {
	log := s.log.With(
		slog.String("op", "service.GetUsers"),
		slog.String("request_id", middleware.GetReqID(ctx)),
	)

	userEntities, err := s.adminUserRepo.GetAllUsers(ctx)

	if err != nil {
		log.Error("failed to get all users list", sl.Err(err))
		return nil, fmt.Errorf("failed to get users")
	}

  users := make([]UserFullInformation, len(userEntities))

	for i, userEntity := range userEntities {
		users[i] = UserFullInformation{
			User: User{
				UserUuid: userEntity.UserUuid,
				Login: userEntity.Login,
				Email: userEntity.Email,
				Role: string(userEntity.Role),
			},
			IsActive: userEntity.IsActive,
			PhotosQuota: userEntity.PhotosQuota,
			CollectionsQuota: userEntity.CollectionsQuota,
			CreateDate: userEntity.CreateDate,
		}
	}

	return users, nil
}

func (s *AdminService) PatchUser(ctx context.Context, userUuid uuid.UUID, request *PatchUserRequest) error {
	log := s.log.With(
		slog.String("op", "service.PatchUser"),
		slog.String("request_id", middleware.GetReqID(ctx)),
	)

	fields := make(map[string]any)
	
	if request.Role != nil {
		fields["role"] = *request.Role
	}
	if request.PhotosQuota != nil {
		fields["photos_quota"] = *request.PhotosQuota
	}
	if request.CollectionsQuota != nil {
		fields["collections_quota"] = *request.CollectionsQuota
	}

	err := s.txManager.WithTransaction(ctx, func(txCtx context.Context) error {
		return s.adminUserRepo.UpdateUser(txCtx, userUuid, fields)
	})

	if err != nil {
		log.Error("failed to update user", sl.Err(err))

		if errors.Is(err, storage.ErrUserNotFound) {
			return ErrUserNotFound
		}

		return fmt.Errorf("failed to update user")
	}

	return nil
}
