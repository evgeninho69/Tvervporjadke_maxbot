package ucports

import (
	"activeTver/internal/domain/entity"
	"context"
)

type UserUseCase interface {
	Create(ctx context.Context, user entity.User) (uint64, error)
	GetByID(ctx context.Context, uid uint64) (entity.User, error)
	GetByEmail(ctx context.Context, email string) (entity.User, error)
	Update(ctx context.Context, user entity.User) error
	Delete(ctx context.Context, uid uint64) error

	IsBlockedByUID(ctx context.Context, uid uint64) (bool, error)
	IsBlockedByEmail(ctx context.Context, email string) (bool, error)

	BlockByEmail(ctx context.Context, adminUID uint64, email, reason string) error
	BlockByUID(ctx context.Context, adminUID, uid uint64, reason string) error

	UnblockByEmail(ctx context.Context, email string) error
	UnblockByUID(ctx context.Context,uid uint64) error

	BlockedUsersPage(ctx context.Context, cursor uint64, pageSize uint64) ([]entity.BlockedUser, error)
	BlockedUsersFirstPage(ctx context.Context, pageSize uint64) ([]entity.BlockedUser, error)
	GetBlockedUser(ctx context.Context, uid uint64) (entity.BlockedUser, error)
}
