package ucports

import (
	"activeTver/internal/domain/entity"
	"activeTver/internal/domain/types"
	"context"
)

type ClaimsUseCase interface {
	GetByID(ctx context.Context, id uint64) (entity.Claim, error)
	GetByIDWithUserEmail(ctx context.Context, id uint64) (entity.Claim, string, error)
	GetFirstPage(ctx context.Context, pageSize uint64) ([]entity.Claim, error)
	GetPage(ctx context.Context, cursor, pageSize uint64) ([]entity.Claim, error)
	GetFirstUserPage(ctx context.Context, uid, pageSize uint64) ([]entity.Claim, error)
	GetUserPage(ctx context.Context, cursor, uid, pageSize uint64) ([]entity.Claim, error)
	CreateProblem(ctx context.Context, claim entity.Claim, files [][]byte) (uint64, error)
	CreateProposal(ctx context.Context, claim entity.Claim) (uint64, error)
	Delete(ctx context.Context, id, uid uint64, isAdmin bool) error
	Update(ctx context.Context, updatedClaim entity.Claim) error
	AddFeedback(ctx context.Context, id uint64, feedback string) error
	ChangeStatus(ctx context.Context, id uint64, status types.ClaimStatus) error
	GetByArea(ctx context.Context, lat1, long1, lat2, long2 float64) ([]entity.Claim, error)
}