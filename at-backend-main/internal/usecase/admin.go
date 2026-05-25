package usecase

import (
	"activeTver/internal/domain/entity"
	"activeTver/internal/domain/ports/repository"
	"activeTver/internal/pkg/apperror"
	"activeTver/internal/pkg/common"
	"activeTver/internal/pkg/logs"
	"context"
)

var dummyAdmin = entity.Admin{}

type AdminService struct {
	Repository repository.AdminRepository
}

func (as AdminService) SetAdminByUID(ctx context.Context, uid uint64) error {
	if err := as.Repository.SetByUID(ctx, uid); err != nil {
		return err
	}

	return nil
}

func (as AdminService) SetAdminByEmail(ctx context.Context, email string) error {
	if err := as.Repository.SetByEmail(ctx, email); err != nil {
		return err
	}

	return nil
}

const opAdminDeleteAdminByEmail = "usecase.Admin.DeleteAdminByEmail"

func (as AdminService) DeleteAdminByEmail(ctx context.Context, email string) error {
	data, err := as.Repository.GetDataByEmail(ctx, email)
	if err != nil {
		return err
	}

	if data.IsSuperAdmin {
		logs.Error(
			apperror.ErrUserSuperAdmin.Error(),
			common.GetCorrelationID(ctx),
			opAdminDeleteAdminByEmail,
		)

		return apperror.ErrUserSuperAdmin
	}

	if err := as.Repository.DeleteByEmail(ctx, email); err != nil {
		return err
	}

	return nil
}

const opAdminDeleteAdminByUID = "usecase.Admin.DeleteAdminByUID"

func (as AdminService) DeleteAdminByUID(ctx context.Context, uid uint64) error {
	data, err := as.Repository.GetDataByUID(ctx, uid)
	if err != nil {
		return err
	}

	if data.IsSuperAdmin {
		logs.Error(
			apperror.ErrUserSuperAdmin.Error(),
			common.GetCorrelationID(ctx),
			opAdminDeleteAdminByUID,
		)

		return apperror.ErrUserSuperAdmin
	}

	if err := as.Repository.DeleteByUID(ctx, uid); err != nil {
		return err
	}

	return nil
}

func (as AdminService) GetAdminDataByUID(ctx context.Context, uid uint64) (entity.Admin, error) {
	data, err := as.Repository.GetDataByUID(ctx, uid)
	if err != nil {
		return dummyAdmin, err
	}

	return data, nil
}

func (as AdminService) GetAdminDataByEmail(ctx context.Context, email string) (entity.Admin, error) {
	data, err := as.Repository.GetDataByEmail(ctx, email)
	if err != nil {
		return dummyAdmin, err
	}

	return data, nil
}

func (as AdminService) GetListAdmins(ctx context.Context) ([]entity.Admin, error) {
	admins, err := as.Repository.GetListAdmins(ctx)
	if err != nil {
		return nil, err
	}

	return admins, nil
}