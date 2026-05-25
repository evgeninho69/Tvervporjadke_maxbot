package postgres

import (
	"activeTver/internal/domain/entity"
	"activeTver/internal/domain/types"
	"activeTver/internal/infrastructure/db/postgres/models"
	"activeTver/internal/pkg/apperror"
	"activeTver/internal/pkg/common"
	"activeTver/internal/pkg/logs"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"context"
)

type claimsRepository struct {
	baseRepository
}

func NewClaimsRepository(pool *pgxpool.Pool) *claimsRepository {
	return &claimsRepository{
		baseRepository{
			pool: pool,
		},
	}
}

var dummyClaim = entity.Claim{}

const opClaimsGetByID = "infrastructure.db.postgres.Claims.GetByID"

const queryClaimsGetByID = `
	SELECT 
		title,
		created_by,
		description, 
		category, 
		status,
		photos, 
		latitude, 
		longitude, 
		feedback, 
		created_at, 
		updated_at, 
		status_updated_at, 
		feedback_updated_at
	FROM public.claim
	WHERE id = $1
`

func (cr claimsRepository) GetByID(ctx context.Context, id uint64) (entity.Claim, error) {
	rows, err := cr.conn(ctx).Query(ctx, queryClaimsGetByID, id)
	if err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(ctx),
			opClaimsGetByID,
		)

		return dummyClaim, apperror.ErrRepository
	}

	claim, err := pgx.CollectOneRow(rows, pgx.RowToStructByNameLax[entity.Claim])
	if err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(ctx),
			opClaimsGetByID,
		)

		if errors.Is(err, pgx.ErrNoRows) {
			return dummyClaim, apperror.ErrClaimNotExists
		}

		return dummyClaim, apperror.ErrRepository
	}

	claim.ID = id

	return claim, nil
}

const opClaimsGetByIDWithUserEmail = "infrastructure.db.postgres.Claims.GetByIDWithUserEmail"

const queryClaimsGetByIDWithUserEmail = `
	SELECT 
		public.claim.title,
		public.claim.created_by,
		public.user.email, 
		public.claim.description, 
		public.claim.category, 
		public.claim.status, 
		public.claim.photos, 
		public.claim.latitude, 
		public.claim.longitude, 
		public.claim.feedback,
		public.claim.created_at, 
		public.claim.updated_at, 
		public.claim.status_updated_at, 
		public.claim.feedback_updated_at
	FROM public.claim
	INNER JOIN public.user
		ON public.user.id = public.claim.created_by
	WHERE public.claim.id = $1
`

func (cr claimsRepository) GetByIDWithUserEmail(ctx context.Context, id uint64) (entity.Claim, string, error) {
	rows, err := cr.conn(ctx).Query(ctx, queryClaimsGetByIDWithUserEmail, id)
	if err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(ctx),
			opClaimsGetByIDWithUserEmail,
		)

		return dummyClaim, "", apperror.ErrRepository
	}

	claim, err := pgx.CollectOneRow(rows, pgx.RowToStructByNameLax[models.ClaimWithUserEmail])
	if err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(ctx),
			opClaimsGetByIDWithUserEmail,
		)

		if errors.Is(err, pgx.ErrNoRows) {
			return dummyClaim, "", apperror.ErrClaimNotExists
		}

		return dummyClaim, "", apperror.ErrRepository
	}

	claimEntity := entity.Claim{
		ID:                id,
		CreatedBy:         claim.CreatedBy,
		Title:             claim.Title,
		Description:       claim.Description,
		Status:            claim.Status,
		Category:          claim.Category,
		Feedback:          claim.Feedback,
		Photos:            claim.Photos,
		Latitude:          claim.Latitude,
		Longitude:         claim.Longitude,
		CreatedAt:         claim.CreatedAt,
		UpdatedAt:         claim.UpdatedAt,
		StatusUpdatedAt:   claim.StatusUpdatedAt,
		FeedbackUpdatedAt: claim.FeedbackUpdatedAt,
	}

	return claimEntity, claim.UserEmail, nil
}

const opClaimsGetClaimsFromRows = "infrastructure.db.postgres.Claims.getClaimsFromRows"

func (cr claimsRepository) getClaimsFromRows(ctx context.Context, rows pgx.Rows) ([]entity.Claim, error) {
	claims, err := pgx.CollectRows(rows, pgx.RowToStructByNameLax[entity.Claim])
	if err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(ctx),
			opClaimsGetClaimsFromRows,
		)

		return nil, apperror.ErrRepository
	}

	return claims, nil
}

const opClaimsGetFirstPage = "infrastructure.db.postgres.Claims.GetFirstPage"

const queryClaimsGetFirstPage = `
	SELECT 
		id, title, description, category, status, latitude, longitude, created_at, updated_at
	FROM public.claim
	ORDER BY id DESC
	LIMIT $1
`

func (cr claimsRepository) GetFirstPage(ctx context.Context, pageSize uint64) ([]entity.Claim, error) {
	rows, err := cr.conn(ctx).Query(ctx, queryClaimsGetFirstPage, pageSize)
	if err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(ctx),
			opClaimsGetFirstPage,
		)

		return nil, apperror.ErrRepository
	}

	return cr.getClaimsFromRows(ctx, rows)
}

const opClaimsGetPage = "infrastructure.db.postgres.Claims.GetPage"

const queryClaimsGetPage = `
	SELECT 
		id, title, description, category, status, latitude, longitude, created_at, updated_at
	FROM public.claim
	WHERE id < $1
	ORDER BY id DESC
	LIMIT $2
`

func (cr claimsRepository) GetPage(ctx context.Context, ptr uint64, pageSize uint64) ([]entity.Claim, error) {
	rows, err := cr.conn(ctx).Query(ctx, queryClaimsGetPage, ptr, pageSize)
	if err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(ctx),
			opClaimsGetPage,
		)

		return nil, apperror.ErrRepository
	}

	return cr.getClaimsFromRows(ctx, rows)
}

const opClaimsGetFirstUserPage = "infrastructure.db.postgres.Claims.GetFirstUserPage"

const queryClaimsGetFirstUserPage = `
	SELECT 
		id, title, description, category, status, latitude, longitude, created_at, updated_at
	FROM public.claim
	WHERE created_by = $1
	ORDER BY id DESC
	LIMIT $2
`

func (cr claimsRepository) GetFirstUserPage(ctx context.Context, pageSize uint64, uid uint64) ([]entity.Claim, error) {
	rows, err := cr.conn(ctx).Query(ctx, queryClaimsGetFirstUserPage, uid, pageSize)
	if err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(ctx),
			opClaimsGetFirstUserPage,
		)

		return nil, apperror.ErrRepository
	}

	return cr.getClaimsFromRows(ctx, rows)
}

const opClaimsGetUserPage = "infrastructure.db.postgres.Claims.GetUserPage"

const queryClaimsGetUserPage = `
	SELECT 
		id, title, description, category, status, latitude, longitude, created_at, updated_at
	FROM claim
	WHERE id < $1 and created_by = $2
	ORDER BY id DESC
	LIMIT $3
`

func (cr claimsRepository) GetUserPage(ctx context.Context, ptr uint64, pageSize uint64, uid uint64) ([]entity.Claim, error) {
	rows, err := cr.conn(ctx).Query(ctx, queryClaimsGetUserPage, ptr, uid, pageSize)
	if err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(ctx),
			opClaimsGetUserPage,
		)

		return nil, apperror.ErrRepository
	}

	return cr.getClaimsFromRows(ctx, rows)
}

const opClaimsCreate = "infrastructure.db.postgres.Claims.Create"

const queryClaimsCreate = `
	INSERT INTO public.claim 
		(created_by, title, description, category, status, photos, latitude, longitude) 
	VALUES 
		($1, $2, $3, $4, $5, $6, $7, $8) 
	RETURNING id
`

func (cr claimsRepository) Create(ctx context.Context, data entity.Claim) (uint64, error) {
	var id uint64

	err := cr.conn(ctx).QueryRow(
		ctx,
		queryClaimsCreate,
		data.CreatedBy,
		data.Title,
		data.Description,
		data.Category,
		data.Status.ToString(),
		data.Photos,
		data.Latitude,
		data.Longitude,
	).Scan(&id)
	if err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(ctx),
			opClaimsCreate,
		)

		return id, apperror.ErrRepository
	}

	return id, nil
}

const opClaimsDelete = "infrastructure.db.postgres.Claims.Delete"

const queryClaimsDelete = `
	DELETE FROM claim
	WHERE id = $1
`

func (cr claimsRepository) Delete(ctx context.Context, id uint64) error {
	resp, err := cr.conn(ctx).Exec(ctx, queryClaimsDelete, id)
	if err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(ctx),
			opClaimsDelete,
		)

		return apperror.ErrRepository
	}

	if resp.RowsAffected() < 1 {
		logs.Warn(
			apperror.ErrClaimNotExists.Error(),
			common.GetCorrelationID(ctx),
			opClaimsDelete,
		)

		return apperror.ErrClaimNotExists
	}

	return nil
}

const opClaimsUpdate = "infrastructure.db.postgres.Claims.Update"

const queryClaimsUpdate = `
	UPDATE public.claim
	SET 
		title = $1,
		description = $2,
		longitude = $3,
		latitude = $4,
		updated_at = NOW() AT TIME ZONE 'UTC'
	WHERE
		id = $5
`

func (cr claimsRepository) Update(ctx context.Context, data entity.Claim) error {
	resp, err := cr.conn(ctx).Exec(
		ctx,
		queryClaimsUpdate,
		data.Title,
		data.Description,
		data.Longitude,
		data.Latitude,
		data.ID,
	)
	if err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(ctx),
			opClaimsUpdate,
		)

		return apperror.ErrRepository
	}

	if resp.RowsAffected() < 1 {
		logs.Warn(
			apperror.ErrClaimNotExists.Error(),
			common.GetCorrelationID(ctx),
			opClaimsUpdate,
		)

		return apperror.ErrClaimNotExists
	}

	return nil
}

const opClaimsChangeStatus = "infrastructure.db.postgres.Claims.ChangeStatus"

const queryClaimsChangeStatus = `
	UPDATE public.claim
	SET
		status = $1,
		status_updated_at = NOW() AT TIME ZONE 'UTC'
	WHERE
		id = $2
`

func (cr claimsRepository) ChangeStatus(ctx context.Context, id uint64, status types.ClaimStatus) error {
	resp, err := cr.conn(ctx).Exec(ctx, queryClaimsChangeStatus, status.ToString(), id)
	if err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(ctx),
			opClaimsChangeStatus,
		)

		return apperror.ErrRepository
	}

	if resp.RowsAffected() < 1 {
		logs.Warn(
			apperror.ErrClaimNotExists.Error(),
			common.GetCorrelationID(ctx),
			opClaimsChangeStatus,
		)

		return apperror.ErrClaimNotExists
	}

	return nil
}

const opClaimsAddFeedback = "infrastructure.db.postgres.Claims.AddFeedback"

const queryClaimsAddFeedback = `
	UPDATE public.claim
	SET
		feedback = $1,
		feedback_updated_at = NOW() AT TIME ZONE 'UTC'
	WHERE
		id = $2
`

func (cr claimsRepository) AddFeedback(ctx context.Context, id uint64, feedback string) error {
	resp, err := cr.conn(ctx).Exec(ctx, queryClaimsAddFeedback, feedback, id)
	if err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(ctx),
			opClaimsAddFeedback,
		)

		return apperror.ErrRepository
	}

	if resp.RowsAffected() < 1 {
		logs.Warn(
			apperror.ErrClaimNotExists.Error(),
			common.GetCorrelationID(ctx),
			opClaimsAddFeedback,
		)

		return apperror.ErrClaimNotExists
	}
	return nil
}

const opClaimsGetByArea = "infrastructure.db.postgres.Claims.GetByArea"

const queryClaimsGetByArea = `
	SELECT 
		id, latitude, longitude
	FROM public.claim
	WHERE 
		latitude <= $1 and longitude >= $2 and latitude > $3 and longitude < $4 and (status = $5 OR (status = $6 and status_updated_at >= $7))
`

func (cr claimsRepository) GetByArea(ctx context.Context, lat1, long1, lat2, long2 float64, acceptedStatus, completedStatus string, startingFrom time.Time) ([]entity.Claim, error) {
	rows, err := cr.conn(ctx).Query(
		ctx,
		queryClaimsGetByArea,
		lat1,
		long1,
		lat2,
		long2,
		acceptedStatus,
		completedStatus,
		startingFrom,
	)
	if err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(ctx),
			opClaimsGetByArea,
		)

		return nil, apperror.ErrRepository
	}

	claims, err := pgx.CollectRows(rows, pgx.RowToStructByNameLax[entity.Claim])
	if err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(ctx),
			opClaimsGetByArea,
		)

		return nil, apperror.ErrRepository
	}

	return claims, nil
}
