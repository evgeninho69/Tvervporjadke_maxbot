package repository

import (
	"activeTver/internal/domain/entity"
	"activeTver/internal/domain/types"
	"context"
	"time"
)

type ClaimsRepository interface {
	GetByID(ctx context.Context, id uint64) (entity.Claim, error)
	GetByIDWithUserEmail(ctx context.Context, id uint64) (entity.Claim, string, error)
	GetFirstPage(ctx context.Context, pageSize uint64) ([]entity.Claim, error)
	GetPage(ctx context.Context, ptr uint64, pageSize uint64) ([]entity.Claim, error)
	GetFirstUserPage(ctx context.Context, pageSize uint64, uid uint64) ([]entity.Claim, error)
	GetUserPage(ctx context.Context, ptr uint64, pageSize uint64, uid uint64) ([]entity.Claim, error)
	Create(ctx context.Context, data entity.Claim) (uint64, error)
	Delete(ctx context.Context, id uint64) error
	Update(ctx context.Context, data entity.Claim) error
	ChangeStatus(ctx context.Context, id uint64, status types.ClaimStatus) error
	AddFeedback(ctx context.Context, id uint64, feedback string) error
	GetByArea(ctx context.Context, lat1, long1, lat2, long2 float64, acceptedStatus, completedStatus string, startingFrom time.Time) ([]entity.Claim, error)
}
