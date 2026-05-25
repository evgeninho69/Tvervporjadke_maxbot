package postgres

import (
	"activeTver/internal/domain/entity"
	"activeTver/internal/pkg/apperror"
	"activeTver/internal/pkg/common"
	"activeTver/internal/pkg/logs"
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type refreshTokensRepository struct {
	baseRepository
}

func NewRefreshTokensRepository(pool *pgxpool.Pool) *refreshTokensRepository {
	return &refreshTokensRepository{
		baseRepository{
			pool: pool,
		},
	}
}

var (
	dummyRefreshToken = entity.RefreshToken{}
)

const opTokensCreate = "infrastructure.db.postgres.Tokens.Create"

const queryTokensCreate = `
	INSERT INTO public.refresh_tokens
		(user_id, hash, salt, expire)
	VALUES
		($1, $2, $3, $4)
	RETURNING id
`

func (rtr refreshTokensRepository) Create(ctx context.Context, rt entity.RefreshToken) (uuid.UUID, error) {
	row := rtr.conn(ctx).QueryRow(
		ctx,
		queryTokensCreate,
		rt.UserID,
		rt.Hash,
		rt.Salt,
		rt.Expire,
	)

	var id uuid.UUID

	if err := row.Scan(&id); err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(ctx),
			opTokensCreate,
		)

		if pgErr, ok := err.(*pgconn.PgError); ok {
			switch pgErr.Code {
			case pgerrcode.UniqueViolation:
				return id, apperror.ErrTokenExists
			case pgerrcode.NotNullViolation:
				return id, apperror.ErrUserNotExists
			}
		}

		return id, apperror.ErrRepository
	}

	return id, nil
}

func (rtr refreshTokensRepository) deleteBy(ctx context.Context, query, op string, value any) error {
	resp, err := rtr.conn(ctx).Exec(ctx, query, value)
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
			apperror.ErrTokenNotExists.Error(),
			common.GetCorrelationID(ctx),
			op,
		)

		return apperror.ErrTokenNotExists
	}

	return nil
}

const opTokensDeleteByID = "infrastructure.db.postgres.Tokens.DeleteByID"

const queryTokensDeleteByID = `
	DELETE FROM public.refresh_tokens
	WHERE id = $1
`

func (rtr refreshTokensRepository) DeleteByTokenID(ctx context.Context, tokenID uuid.UUID) error {
	return rtr.deleteBy(ctx, queryTokensDeleteByID, opTokensDeleteByID, tokenID)
}

const opTokensDeleteByHash = "infrastructure.db.postgres.Tokens.DeleteByHash"

const queryTokensDeleteByHash = `
	DELETE FROM public.refresh_tokens
	WHERE hash = $1
`

func (rtr refreshTokensRepository) DeleteByHash(ctx context.Context, hash string) error {
	return rtr.deleteBy(ctx, queryTokensDeleteByHash, opTokensDeleteByHash, hash)
}

const opTokensDeleteAllByUserID = "infrastructure.db.postgres.Tokens.DeleteAllByUserID"

const queryTokensDeleteAllByUserID = `
	DELETE FROM public.refresh_tokens
	WHERE user_id = $1
`

func (rtr refreshTokensRepository) DeleteAllByUserID(ctx context.Context, userID uint64) error {
	if _, err := rtr.conn(ctx).Exec(ctx, queryTokensDeleteAllByUserID, userID); err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(ctx),
			opTokensDeleteAllByUserID,
		)

		return apperror.ErrRepository
	}

	return nil
}

func (rtr refreshTokensRepository) getDataBy(ctx context.Context, query, op string, value any) (entity.RefreshToken, error) {
	rows, err := rtr.conn(ctx).Query(ctx, query, value)
	if err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(ctx),
			op,
		)

		return dummyRefreshToken, apperror.ErrRepository
	}

	rt, err := pgx.CollectOneRow(rows, pgx.RowToStructByNameLax[entity.RefreshToken])
	if err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(ctx),
			op,
		)

		if errors.Is(err, pgx.ErrNoRows) {
			return dummyRefreshToken, apperror.ErrTokenNotExists
		}

		return dummyRefreshToken, apperror.ErrRepository
	}

	return rt, nil
}

const opTokensGetDataByHash = "infrastructure.db.postgres.Tokens.GetRefreshTokenDataByHash"

const queryTokensGetDataByHash = `
	SELECT
		id, user_id, salt, expire, revoked
	FROM public.refresh_tokens
	WHERE hash = $1
	FOR UPDATE
`

func (rtr refreshTokensRepository) GetDataByHash(ctx context.Context, hash string) (entity.RefreshToken, error) {
	rt, err := rtr.getDataBy(ctx, queryTokensGetDataByHash, opTokensGetDataByHash, hash)
	if err != nil {
		return rt, err
	}

	rt.Hash = hash

	return rt, nil
}

const opTokensGetDataByTokenID = "infrastructure.db.postgres.Tokens.GetRefreshTokenDataByTokenID"

const queryTokensGetDataByTokenID = `
	SELECT
		user_id, hash, salt, expire, revoked
	FROM public.refresh_tokens
	WHERE id = $1
	FOR UPDATE
`

func (rtr refreshTokensRepository) GetDataByTokenID(ctx context.Context, tokenID uuid.UUID) (entity.RefreshToken, error) {
	rt, err := rtr.getDataBy(ctx, queryTokensGetDataByTokenID, opTokensGetDataByTokenID, tokenID)
	if err != nil {
		return rt, err
	}

	rt.ID = tokenID

	return rt, nil
}

const opTokensRevoke = "infrastructure.db.postgres.Tokens.Revoke"

const queryTokensRevoke = `
	UPDATE public.refresh_tokens
	SET
		revoked = true
	WHERE
		id = $1
`

func (rtr refreshTokensRepository) Revoke(ctx context.Context, tokenID uuid.UUID) error {
	resp, err := rtr.conn(ctx).Exec(ctx, queryTokensRevoke, tokenID)
	if err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(ctx),
			opTokensRevoke,
		)

		return apperror.ErrRepository
	}

	if resp.RowsAffected() < 1 {
		logs.Warn(
			apperror.ErrTokenNotExists.Error(),
			common.GetCorrelationID(ctx),
			opTokensRevoke,
		)

		return apperror.ErrTokenNotExists
	}

	return nil
}
