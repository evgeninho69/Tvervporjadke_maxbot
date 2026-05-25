package postgres

import (
	"activeTver/internal/domain/entity"
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

type userRepository struct {
	baseRepository
}

func NewUserRepository(pool *pgxpool.Pool) *userRepository {
	return &userRepository{
		baseRepository{
			pool: pool,
		},
	}
}

var (
	dummyUser        = entity.User{}
	dummyBlockedUser = entity.BlockedUser{}
)

const opUserCreate = "infrastructure.db.postgres.User.Create"

const queryUserCreate = `
	INSERT INTO public.user
		(email)
	VALUES 
		($1)
	RETURNING id
`

func (ur userRepository) Create(ctx context.Context, user entity.User) (uint64, error) {
	var uid uint64

	err := ur.conn(ctx).QueryRow(ctx, queryUserCreate, user.Email).Scan(&uid)
	if err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(ctx),
			opUserCreate,
		)

		if pgErr, ok := err.(*pgconn.PgError); ok {
			if pgErr.Code == pgerrcode.UniqueViolation {
				return 0, apperror.ErrUserExists
			}

			return 0, apperror.ErrRepository
		}

		return 0, apperror.ErrRepository
	}

	return uid, nil
}

func (ur userRepository) getBy(ctx context.Context, query, op string, value any) (entity.User, error) {
	rows, err := ur.conn(ctx).Query(ctx, query, value)
	if err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(ctx),
			op,
		)

		return dummyUser, apperror.ErrRepository
	}

	user, err := pgx.CollectOneRow(rows, pgx.RowToStructByNameLax[entity.User])
	if err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(ctx),
			op,
		)

		if errors.Is(err, pgx.ErrNoRows) {
			return user, apperror.ErrUserNotExists
		}

		return user, apperror.ErrRepository
	}

	return user, nil
}

const opUserGetByID = "infrastructure.db.postgres.User.GetByID"

const queryUserGetByID = `
	SELECT 
		email, registered_at
	FROM public.user
	WHERE id = $1
`

func (ur userRepository) GetByID(ctx context.Context, uid uint64) (entity.User, error) {
	user, err := ur.getBy(ctx, queryUserGetByID, opUserGetByID, uid)
	if err != nil {
		return user, err
	}

	user.ID = uid

	return user, nil
}

const opUserGetByEmail = "infrastructure.db.postgres.User.GetByEmail"

const queryUserGetByEmail = `
	SELECT 
		id, registered_at
	FROM public.user
	WHERE email = $1
`

func (ur userRepository) GetByEmail(ctx context.Context, email string) (entity.User, error) {
	user, err := ur.getBy(ctx, queryUserGetByEmail, opUserGetByEmail, email)
	if err != nil {
		return user, err
	}

	user.Email = email

	return user, nil
}

const opUserUpdate = "infrastructure.db.postgres.User.Update"

const queryUserUpdate = `
	UPDATE public.user
	SET 
		email = $1
	WHERE
		id = $2
`

func (ur userRepository) Update(ctx context.Context, user entity.User) error {
	resp, err := ur.conn(ctx).Exec(ctx, queryUserUpdate, user.Email, user.ID)
	if err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(ctx),
			opUserUpdate,
		)

		return apperror.ErrRepository
	}

	if resp.RowsAffected() < 1 {
		logs.Warn(
			apperror.ErrUserNotExists.Error(),
			common.GetCorrelationID(ctx),
			opUserUpdate,
		)

		return apperror.ErrUserNotExists
	}

	return nil
}

const opUserDelete = "infrastructure.db.postgres.User.Delete"

const queryUserDelete = `
	DELETE FROM public.user
	WHERE id = $1
`

func (ur userRepository) Delete(ctx context.Context, uid uint64) error {
	resp, err := ur.conn(ctx).Exec(ctx, queryUserDelete, uid)
	if err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(ctx),
			opUserDelete,
		)

		return apperror.ErrRepository
	}

	if resp.RowsAffected() < 1 {
		logs.Warn(
			apperror.ErrUserNotExists.Error(),
			common.GetCorrelationID(ctx),
			opUserDelete,
		)

		return apperror.ErrUserNotExists
	}

	return nil
}

func (ur userRepository) isBlockedBy(ctx context.Context, value any, query, op string) (bool, error) {
	blocked := false

	if err := ur.conn(ctx).QueryRow(ctx, query, value).Scan(&blocked); err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(ctx),
			op,
		)

		if errors.Is(err, pgx.ErrNoRows) {
			return true, apperror.ErrUserNotExists
		}

		return true, apperror.ErrRepository
	}

	return blocked, nil
}

const opUserIsBlockedByUID = "infrastructure.db.postgres.User.IsBlockedByUID"

const queryUserIsBlockedByUID = `
	SELECT EXISTS (
		SELECT id
		FROM public.blocked_user
		WHERE uid = $1
	)
`

func (ur userRepository) IsBlockedByUID(ctx context.Context, uid uint64) (bool, error) {
	return ur.isBlockedBy(ctx, uid, queryUserIsBlockedByUID, opUserIsBlockedByUID)
}

const opUserIsBlockedByEmail = "infrastructure.db.postgres.User.IsBlockedByEmail"

const queryUserIsBlockedByEmail = `
	SELECT EXISTS (
		SELECT id
		FROM public.blocked_user
		WHERE uid = (select id from public.user WHERE email = $1)
	)
`

func (ur userRepository) IsBlockedByEmail(ctx context.Context, email string) (bool, error) {
	return ur.isBlockedBy(ctx, email, queryUserIsBlockedByEmail, opUserIsBlockedByEmail)
}

func (ur userRepository) blockBy(ctx context.Context, query, op, reason string, adminUID uint64, value any) error {
	if _, err := ur.conn(ctx).Exec(ctx, query, value, reason, adminUID); err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(ctx),
			op,
		)

		if pgErr, ok := err.(*pgconn.PgError); ok {
			switch pgErr.Code {
			case pgerrcode.UniqueViolation:
				return apperror.ErrUserAlreadyBlocked
			case pgerrcode.NotNullViolation:
				return apperror.ErrUserNotExists
			}
		}

		return apperror.ErrRepository
	}

	return nil
}

const opUserBlockByEmail = "infrastructure.db.postgres.User.BlockByEmail"

const queryUserBlockByEmail = `
	INSERT INTO public.blocked_user
		(uid, reason, blocked_by)
	VALUES (
		(SELECT id FROM public.user WHERE email = $1),
		$2,
		$3
	)
`

func (ur userRepository) BlockByEmail(ctx context.Context, adminUID uint64, email, reason string) error {
	return ur.blockBy(ctx, queryUserBlockByEmail, opUserBlockByEmail, reason, adminUID, email)
}

const opUserBlockByUID = "infrastructure.db.postgres.User.BlockByUID"

const queryUserBlockByUID = `
	INSERT INTO public.blocked_user
		(uid, reason, blocked_by)
	VALUES (
		(SELECT id FROM public.user WHERE id = $1),
		$2,
		$3
	)
`

func (ur userRepository) BlockByUID(ctx context.Context, adminUID, uid uint64, reason string) error {
	return ur.blockBy(ctx, queryUserBlockByUID, opUserBlockByUID, reason, adminUID, uid)
}

func (ur userRepository) unblockBy(ctx context.Context, query, op string, value any) error {
	resp, err := ur.conn(ctx).Exec(ctx, query, value)
	if err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(ctx),
			op,
		)

		return apperror.ErrRepository
	}

	if resp.RowsAffected() < 1 {
		logs.Warn(
			apperror.ErrUserNotBlocked.Error(),
			common.GetCorrelationID(ctx),
			op,
		)

		return apperror.ErrUserNotBlocked
	}

	return nil
}

const opUserUnblockByEmail = "infrastructure.db.postgres.User.UnblockByEmail"

const queryUserUnblockByEmail = `
	DELETE FROM public.blocked_user
	WHERE uid = (select id from public.user where email = $1)
`

func (ur userRepository) UnblockByEmail(ctx context.Context, email string) error {
	return ur.unblockBy(ctx, queryUserUnblockByEmail, opUserUnblockByEmail, email)
}

const opUserUnblockByUID = "infrastructure.db.postgres.User.UnblockByUID"

const queryUserUnblockByUID = `
	DELETE FROM public.blocked_user
	WHERE uid = (select id from public.user where id = $1)
`

func (ur userRepository) UnblockByUID(ctx context.Context, uid uint64) error {
	return ur.unblockBy(ctx, queryUserUnblockByUID, opUserUnblockByUID, uid)
}

func (ur userRepository) getBlockedUsersFromRows(ctx context.Context, rows pgx.Rows, op string) ([]entity.BlockedUser, error) {
	blockedUsers, err := pgx.CollectRows(rows, pgx.RowToStructByNameLax[entity.BlockedUser])
	if err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(ctx),
			op,
		)

		return nil, apperror.ErrRepository
	}

	return blockedUsers, nil
}

const opBlockedUserPage = "infrastructure.db.postgres.User.BlockedPage"

const queryBlockedUserPage = `
	SELECT 
		bu.id, bu.uid, u.email, bu.blocked_at
	FROM blocked_user AS bu
	INNER JOIN public.user AS u
		ON u.id = bu.uid
	WHERE bu.id < $1
	ORDER BY bu.id DESC
	LIMIT $2
`

func (ur userRepository) BlockedUsersPage(ctx context.Context, cursor uint64, pageSize uint64) ([]entity.BlockedUser, error) {
	rows, err := ur.conn(ctx).Query(ctx, queryBlockedUserPage, cursor, pageSize)
	if err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(ctx),
			opBlockedUserPage,
		)

		return nil, apperror.ErrRepository
	}

	return ur.getBlockedUsersFromRows(ctx, rows, opBlockedUserPage)
}

const opBlockedUserFirstPage = "infrastructure.db.postgres.User.BlockedFirstPage"

const queryBlockedUserFirstPage = `
	SELECT 
		bu.id, bu.uid, u.email, bu.blocked_at
	FROM blocked_user AS bu
	INNER JOIN public.user AS u
		ON u.id = bu.uid
	ORDER BY bu.id DESC
	LIMIT $1
`

func (ur userRepository) BlockedUsersFirstPage(ctx context.Context, pageSize uint64) ([]entity.BlockedUser, error) {
	rows, err := ur.conn(ctx).Query(ctx, queryBlockedUserFirstPage, pageSize)
	if err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(ctx),
			opBlockedUserFirstPage,
		)

		return nil, apperror.ErrRepository
	}

	return ur.getBlockedUsersFromRows(ctx, rows, opBlockedUserFirstPage)
}

const opBlockedUserGetBlockedUser = "infrastructure.db.postgres.User.GetBlockedUser"

const queryBlockedUserGetBlockedUser = `
	SELECT
		public.blocked_user.id,
		public.user.email,
		public.blocked_user.uid,
		public.blocked_user.reason,
		public.blocked_user.blocked_at,
		public.blocked_user.blocked_by
	FROM public.blocked_user
	INNER JOIN public.user
		ON public.user.id = public.blocked_user.uid
	WHERE public.blocked_user.uid = $1
`

func (ur userRepository) GetBlockedUser(ctx context.Context, uid uint64) (entity.BlockedUser, error) {
	rows, err := ur.conn(ctx).Query(ctx, queryBlockedUserGetBlockedUser, uid)
	if err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(ctx),
			opBlockedUserGetBlockedUser,
		)

		return dummyBlockedUser, apperror.ErrRepository
	}

	blockedUser, err := pgx.CollectOneRow(rows, pgx.RowToStructByNameLax[entity.BlockedUser])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return dummyBlockedUser, apperror.ErrUserNotBlocked
		}

		logs.Error(
			err.Error(),
			common.GetCorrelationID(ctx),
			opBlockedUserGetBlockedUser,
		)

		return dummyBlockedUser, apperror.ErrRepository
	}

	return blockedUser, nil
}
