package usecase

import (
	"activeTver/internal/domain/entity"
	"activeTver/internal/domain/ports/repository"
	"context"
)

type UserService struct {
	Repository  repository.UserRepository
	MaxPageSize uint64
}

func (us UserService) Create(ctx context.Context, user entity.User) (uint64, error) {
	uid, err := us.Repository.Create(ctx, user)
	if err != nil {
		return uid, err
	}

	return uid, nil
}

func (us UserService) Update(ctx context.Context, user entity.User) error {
	err := us.Repository.Update(ctx, user)
	if err != nil {
		return err
	}

	return nil
}

func (us UserService) GetByID(ctx context.Context, uid uint64) (entity.User, error) {
	user, err := us.Repository.GetByID(ctx, uid)
	if err != nil {
		return user, err
	}

	return user, nil
}

func (us UserService) GetByEmail(ctx context.Context, email string) (entity.User, error) {
	user, err := us.Repository.GetByEmail(ctx, email)
	if err != nil {
		return user, err
	}

	return user, nil
}

func (us UserService) Delete(ctx context.Context, uid uint64) error {
	err := us.Repository.Delete(ctx, uid)
	if err != nil {
		return err
	}

	return nil
}

func (us UserService) IsBlockedByUID(ctx context.Context, uid uint64) (bool, error) {
	blocked, err := us.Repository.IsBlockedByUID(ctx, uid)
	if err != nil {
		return true, err
	}

	return blocked, nil
}

func (us UserService) IsBlockedByEmail(ctx context.Context, email string) (bool, error) {
	blocked, err := us.Repository.IsBlockedByEmail(ctx, email)
	if err != nil {
		return true, err
	}

	return blocked, nil
}

func (us UserService) UnblockByEmail(ctx context.Context, email string) error {
	if err := us.Repository.UnblockByEmail(ctx, email); err != nil {
		return err
	}

	return nil
}

func (us UserService) UnblockByUID(ctx context.Context, uid uint64) error {
	if err := us.Repository.UnblockByUID(ctx, uid); err != nil {
		return err
	}

	return nil
}

func (us UserService) BlockByEmail(ctx context.Context, adminUID uint64, email, reason string) error {
	if err := us.Repository.BlockByEmail(ctx, adminUID, email, reason); err != nil {
		return err
	}

	return nil
}

func (us UserService) BlockByUID(ctx context.Context, adminUID, uid uint64, reason string) error {
	if err := us.Repository.BlockByUID(ctx, adminUID, uid, reason); err != nil {
		return err
	}

	return nil
}

func (us UserService) BlockedUsersPage(ctx context.Context, cursor uint64, pageSize uint64) ([]entity.BlockedUser, error) {
	if pageSize > us.MaxPageSize {
		pageSize = us.MaxPageSize
	}

	blockedUsers, err := us.Repository.BlockedUsersPage(ctx, cursor, pageSize)
	if err != nil {
		return nil, err
	}

	return blockedUsers, nil
}

func (us UserService) BlockedUsersFirstPage(ctx context.Context, pageSize uint64) ([]entity.BlockedUser, error) {
	if pageSize > us.MaxPageSize {
		pageSize = us.MaxPageSize
	}

	blockedUsers, err := us.Repository.BlockedUsersFirstPage(ctx, pageSize)
	if err != nil {
		return nil, err
	}

	return blockedUsers, nil
}

func (us UserService) GetBlockedUser(ctx context.Context, uid uint64) (entity.BlockedUser, error) {
	blockedUser, err := us.Repository.GetBlockedUser(ctx, uid)
	if err != nil {
		return blockedUser, err
	}

	return blockedUser, nil
}
