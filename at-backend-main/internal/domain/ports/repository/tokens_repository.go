package repository

import (
	"activeTver/internal/domain/entity"
	"context"

	"github.com/google/uuid"
)

type RefreshTokensRepository interface {
	Create(ctx context.Context, rt entity.RefreshToken) (uuid.UUID, error)
	DeleteByHash(ctx context.Context, hash string) error
	DeleteByTokenID(ctx context.Context, tokenID uuid.UUID) error
	DeleteAllByUserID(ctx context.Context, userID uint64) error
	GetDataByTokenID(ctx context.Context, tokenID uuid.UUID) (entity.RefreshToken, error)
	Revoke(ctx context.Context, id uuid.UUID) error
}
