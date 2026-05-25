package repository

import (
	"activeTver/internal/domain/entity"
	"context"
)

type AnnouncementRepository interface {
	Create(ctx context.Context, announcement entity.Announcement) (uint64, error)
	GetByID(ctx context.Context, id uint64) (entity.Announcement, error)
	Delete(ctx context.Context, id uint64) error
	GetFirstPage(ctx context.Context, pageSize uint64) ([]entity.Announcement, error)
	GetPage(ctx context.Context, cursor, pageSize uint64) ([]entity.Announcement, error)
	Update(ctx context.Context, announcement entity.Announcement) error
}
