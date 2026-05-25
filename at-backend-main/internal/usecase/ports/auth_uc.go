package ucports

import (
	"activeTver/internal/domain/entity"
	"context"
)

type AuthUseCase interface {
	RequestLogin(ctx context.Context, email string) error
	Login(ctx context.Context, email string, mailCode uint64) (jwt string, rt entity.RefreshToken, err error)
	RefreshPairTokens(ctx context.Context, oldRT entity.RefreshToken) (jwt string, rt entity.RefreshToken, err error)
	Logout(ctx context.Context, rt entity.RefreshToken) error
	LogoutAllUserDevices(ctx context.Context, uid uint64) error
}