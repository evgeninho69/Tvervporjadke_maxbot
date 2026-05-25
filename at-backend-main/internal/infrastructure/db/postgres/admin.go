package postgres

import (
	"activeTver/internal/domain/entity"
	"activeTver/internal/infrastructure/db/postgres/models"
	"activeTver/internal/pkg/apperror"
	"activeTver/internal/pkg/common"
	"activeTver/internal/pkg/logs"
	"context"
	"errors"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type adminRepository struct {
	baseRepository
}

func NewAdminRepository(pool *pgxpool.Pool) *adminRepository {
	return &adminRepository{
		baseRepository{
			pool: pool,
		},
	}
}

var dummyAdmin = entity.Admin{}

func (ar adminRepository) setBy(ctx context.Context, query, op string, value any) error {
	_, err := ar.conn(ctx).Exec(ctx, query, value)
	if err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(ctx),
			op,
		)

		if pgErr, ok := err.(*pgconn.PgError); ok {
			switch pgErr.Code {
			case pgerrcode.UniqueViolation:
				return apperror.ErrUserAlreadyAdmin
			case pgerrcode.NotNullViolation:
				return apperror.ErrUserNotExists
			}
		}

		return apperror.ErrRepository
	}

	return nil
}

const opAdminSetByEmail = "infrastructure.db.postgres.Admin.SetByEmail"

const queryAdminSetByEmail = `
	INSERT INTO public.admin
		(uid)
	SELECT 
		id 
	FROM public.user 
	WHERE email = $1
`

func (ar adminRepository) SetByEmail(ctx context.Context, email string) error {
	return ar.setBy(ctx, queryAdminSetByEmail, opAdminSetByEmail, email)
}

const opAdminSetByUID = "infrastructure.db.postgres.Admin.SetByUID"

const queryAdminSetByUID = `
	INSERT INTO public.admin
		(uid)
	SELECT
		id
	FROM public.user
	WHERE id = $1
`

func (ar adminRepository) SetByUID(ctx context.Context, uid uint64) error {
	return ar.setBy(ctx, queryAdminSetByUID, opAdminSetByUID, uid)
}

func (ar adminRepository) deleteBy(ctx context.Context, query, op string, value any) error {
	resp, err := ar.conn(ctx).Exec(ctx, query, value)
	if err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(ctx),
			op,
		)

		if pgErr, ok := err.(*pgconn.PgError); ok {
			switch pgErr.Code {
			case pgerrcode.NotNullViolation:
				return apperror.ErrUserNotExists
			}
		}

		return apperror.ErrRepository
	}

	if resp.RowsAffected() < 1 {
		logs.Warn(
			apperror.ErrUserNotAdmin.Error(),
			common.GetCorrelationID(ctx),
			op,
		)

		return apperror.ErrUserNotAdmin
	}

	return nil
}

const opAdminDeleteByEmail = "infrastructure.db.postgres.Admin.DeleteByEmail"

const queryAdminDeleteByEmail = `
	DELETE FROM public.admin
	WHERE uid = (select id from public.user where email = $1)
`

func (ar adminRepository) DeleteByEmail(ctx context.Context, email string) error {
	return ar.deleteBy(ctx, queryAdminDeleteByEmail, opAdminDeleteByEmail, email)
}

const opAdminDeleteByUID = "infrastructure.db.postgres.Admin.DeleteByUID"

const queryAdminDeleteByUID = `
	DELETE FROM public.admin
	WHERE uid = $1
`

func (ar adminRepository) DeleteByUID(ctx context.Context, uid uint64) error {
	return ar.deleteBy(ctx, queryAdminDeleteByUID, opAdminDeleteByUID, uid)
}

func (ar adminRepository) getDataBy(ctx context.Context, query, op string, value any) (entity.Admin, error) {
	rows, err := ar.conn(ctx).Query(ctx, query, value)
	if err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(ctx),
			op,
		)

		return dummyAdmin, apperror.ErrRepository
	}

	adminModel, err := pgx.CollectOneRow(rows, pgx.RowToStructByNameLax[models.AdminData])
	if err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(ctx),
			op,
		)

		if errors.Is(err, pgx.ErrNoRows) {
			return dummyAdmin, apperror.ErrUserNotAdmin
		}

		return dummyAdmin, apperror.ErrRepository
	}

	admin := entity.Admin{
		UID:          adminModel.UID,
		Email:        adminModel.Email,
		IsSuperAdmin: adminModel.IsSuperAdmin,
	}

	return admin, nil
}

const opAdminGetDataByUID = "infrastructure.db.postgres.Admin.GetDataByUID"

const queryAdminGetDataByUID = `
	SELECT
		uid, email, is_super_admin 
	FROM public.admin
	INNER JOIN public.user
		ON public.user.id = public.admin.uid
	WHERE uid = $1
`

func (ar adminRepository) GetDataByUID(ctx context.Context, uid uint64) (entity.Admin, error) {
	admin, err := ar.getDataBy(ctx, queryAdminGetDataByUID, opAdminGetDataByUID, uid)
	if err != nil {
		return dummyAdmin, err
	}

	admin.UID = uid

	return admin, nil
}

const opAdminGetDataByEmail = "infrastructure.db.postgres.Admin.GetDataByEmail"

const queryAdminGetDataByEmail = `
	SELECT
		uid, email, is_super_admin 
	FROM public.admin
	INNER JOIN public.user
		ON public.user.id = public.admin.uid
	WHERE uid = (select id from public.user where email = $1)
`

func (ar adminRepository) GetDataByEmail(ctx context.Context, email string) (entity.Admin, error) {
	admin, err := ar.getDataBy(ctx, queryAdminGetDataByEmail, opAdminGetDataByEmail, email)
	if err != nil {
		return dummyAdmin, err
	}

	return admin, nil
}

const opAdminGetListAdmins = "infrastructure.db.postgres.Admin.GetListAdmins"

const queryAdminGetListAdmins = `
	SELECT uid, email, is_super_admin 
	FROM public.admin
	INNER JOIN public.user
		ON public.user.id = public.admin.uid
`

func (ar adminRepository) GetListAdmins(ctx context.Context) ([]entity.Admin, error) {
	rows, err := ar.conn(ctx).Query(ctx, queryAdminGetListAdmins)
	if err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(ctx),
			opAdminGetListAdmins,
		)

		return nil, apperror.ErrRepository
	}

	adminsModels, err := pgx.CollectRows(rows, pgx.RowToStructByNameLax[models.AdminData])
	if err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(ctx),
			opAdminGetListAdmins,
		)

		return nil, apperror.ErrRepository
	}

	admins := make([]entity.Admin, 0, len(adminsModels))

	for _, adminModel := range adminsModels {
		admin := entity.Admin{
			UID:          adminModel.UID,
			Email:        adminModel.Email,
			IsSuperAdmin: adminModel.IsSuperAdmin,
		}

		admins = append(admins, admin)
	}

	return admins, nil
}
