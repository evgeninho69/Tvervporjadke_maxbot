package repository

import (
	"activeTver/internal/domain/entity"
	"context"
)

type AdminRepository interface {
	SetByEmail(ctx context.Context, email string) error
	SetByUID(ctx context.Context, uid uint64) error
	DeleteByEmail(ctx context.Context, email string) error
	DeleteByUID(ctx context.Context, uid uint64) error
	GetDataByUID(ctx context.Context, uid uint64) (entity.Admin, error)
	GetDataByEmail(ctx context.Context, email string) (entity.Admin, error)
	GetListAdmins(ctx context.Context) ([]entity.Admin, error)
}
