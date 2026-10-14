package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"photo-viewer-server/internal/lib/logger/sl"
	"photo-viewer-server/internal/storage"
	"photo-viewer-server/internal/storage/entity"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
)

type TagSmallInfo struct {
	TagUuid uuid.UUID `json:"tag_uuid"`
	TagName string `json:"tag_name"`
}

type TagInfo struct {
	TagSmallInfo
	TagDescription string `json:"tag_description"`
}

type SaveTagInput struct {
	TagName string `json:"tag_name" validate:"required,min=2,max=50"`
	TagDescription string `json:"tag_description,omitempty" validate:"max=200"`
}

type TagService struct {
	log *slog.Logger
	tagRepo TagRepo
	txManager storage.TxManager
}

func NewTagService(log *slog.Logger, tagRepo TagRepo, txManager storage.TxManager) *TagService {
	return &TagService{
		log: log,
		tagRepo: tagRepo,
		txManager: txManager,
	}
}

func (s *TagService) SaveTag(ctx context.Context, tagInput SaveTagInput) (uuid.UUID, error) {
	lg := s.log.With(
		slog.String("op", "service.SaveTag"),
		slog.String("request_id", middleware.GetReqID(ctx)),
	)

	var tagUuid uuid.UUID

	err := s.txManager.WithTransaction(ctx, func(txCtx context.Context) error {
		newTag := entity.Tag{
			Name: tagInput.TagName,
			Description: tagInput.TagDescription,
		}

		var err error

		tagUuid, err = s.tagRepo.SaveTag(txCtx, &newTag)

		if err != nil {
			if errors.Is(err, storage.ErrTagAlreadyExists) {
				return ErrTagAlreadyExists
			}

			return fmt.Errorf("failed to save tag: %w", err)
		}

		return nil
	})

	if err != nil {
		lg.Error("error save tag", sl.Err(err))

		if errors.Is(err, ErrTagAlreadyExists) {
			return uuid.Nil, ErrTagAlreadyExists
		}

		return uuid.Nil, fmt.Errorf("error save tag: %w", err)
	}

	return tagUuid, nil
}

func (s *TagService) GetTags(ctx context.Context, photoUuid uuid.UUID) ([]TagInfo, error) {
	lg := s.log.With(
		slog.String("op", "service.GetTags"),
		slog.String("request_id", middleware.GetReqID(ctx)),
	)

	var tagsEntities []entity.Tag
	var err error

	if photoUuid == uuid.Nil {
		tagsEntities, err = s.tagRepo.GetAllTags(ctx)
	} else {
		tagsEntities, err = s.tagRepo.GetTagsByPhoto(ctx, photoUuid)
	}

	if err != nil {
		lg.Error("failed to get tags", sl.Err(err))

		if errors.Is(err, storage.ErrPhotoNotFound) {
			return nil, ErrPhotoNotFound
		}

		return nil, fmt.Errorf("failed to get all tags: %w", err)
	}

	tagsInfo := make([]TagInfo, len(tagsEntities))

	for i, tagEntity := range tagsEntities {
		tagsInfo[i] = TagInfo{
			TagSmallInfo: TagSmallInfo{
				TagUuid: tagEntity.TagUuid,
				TagName: tagEntity.Name,
			},
			TagDescription: tagEntity.Description,
		}
	}

	return tagsInfo, nil
}
