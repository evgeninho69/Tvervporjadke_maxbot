package ucports

import (
	"activeTver/internal/domain/entity"
	"context"
)

type AdminUseCase interface {
	SetAdminByUID(ctx context.Context, uid uint64) error
	SetAdminByEmail(ctx context.Context, email string) error
	DeleteAdminByEmail(ctx context.Context, email string) error
	DeleteAdminByUID(ctx context.Context, uid uint64) error
	GetAdminDataByUID(ctx context.Context, uid uint64) (entity.Admin, error)
	GetListAdmins(ctx context.Context) ([]entity.Admin, error)
}