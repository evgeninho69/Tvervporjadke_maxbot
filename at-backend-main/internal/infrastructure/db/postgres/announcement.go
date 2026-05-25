package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"activeTver/internal/domain/entity"
	"activeTver/internal/pkg/apperror"
	"activeTver/internal/pkg/common"
	"activeTver/internal/pkg/logs"
)

type announcementRepository struct {
	baseRepository
}

func NewAnnouncementRepository(pool *pgxpool.Pool) *announcementRepository {
	return &announcementRepository{
		baseRepository{
			pool: pool,
		},
	}
}

var dummyAnnouncement = entity.Announcement{}

const opAnnouncementCreate = "infrastructure.db.postgres.Announcement.Create"

const queryAnnouncementCreate = `
	INSERT INTO public.announcement
		(title, description, created_by, updated_by)
	VALUES
		($1, $2, $3, $3)
	RETURNING id
`

func (ar announcementRepository) Create(ctx context.Context, announcement entity.Announcement) (uint64, error) {
	var id uint64

	if err := ar.conn(ctx).QueryRow(ctx, queryAnnouncementCreate, announcement.Title, announcement.Description, announcement.CreatedBy).Scan(&id); err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(ctx),
			opAnnouncementCreate,
		)

		return id, apperror.ErrRepository
	}

	return id, nil
}

const opAnnouncementGetByID = "infrastructure.db.postgres.Announcement.GetByID"

const queryAnnouncementGetByID = `
	SELECT
		id, title, description, created_at, created_by, updated_at, updated_by
	FROM public.announcement
	WHERE 
		id = $1
`

func (ar announcementRepository) GetByID(ctx context.Context, id uint64) (entity.Announcement, error) {
	rows, err := ar.conn(ctx).Query(ctx, queryAnnouncementGetByID, id)
	if err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(ctx),
			opAnnouncementGetByID,
		)

		return dummyAnnouncement, apperror.ErrRepository
	}

	announcement, err := pgx.CollectOneRow(rows, pgx.RowToStructByNameLax[entity.Announcement])
	if err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(ctx),
			opAnnouncementGetByID,
		)

		if errors.Is(err, pgx.ErrNoRows) {
			return dummyAnnouncement, apperror.ErrAnnouncementNotExists
		}

		return dummyAnnouncement, apperror.ErrRepository
	}

	announcement.ID = id

	return announcement, nil
}

const opAnnouncementDelete = "infrastructure.db.postgres.Announcement.Delete"

const queryAnnouncementDelete = `
	DELETE FROM public.announcement
	WHERE id = $1
`

func (ar announcementRepository) Delete(ctx context.Context, id uint64) error {
	resp, err := ar.conn(ctx).Exec(ctx, queryAnnouncementDelete, id)
	if err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(ctx),
			opAnnouncementDelete,
		)

		return apperror.ErrRepository
	}

	if resp.RowsAffected() < 1 {
		logs.Warn(
			apperror.ErrAnnouncementNotExists.Error(),
			common.GetCorrelationID(ctx),
			opAnnouncementDelete,
		)

		return apperror.ErrAnnouncementNotExists
	}

	return nil
}

const opAnnouncementGetAnnouncementFromRows = "infrastructure.db.postgres.Announcement.getAnnouncementFromRows"

func (ar announcementRepository) getAnnouncementFromRows(ctx context.Context, rows pgx.Rows) ([]entity.Announcement, error) {
	claims, err := pgx.CollectRows(rows, pgx.RowToStructByNameLax[entity.Announcement])
	if err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(ctx),
			opAnnouncementGetAnnouncementFromRows,
		)

		return nil, apperror.ErrRepository
	}

	return claims, nil
}

const opAnnouncementGetFirstPage = "infrastructure.db.postgres.Announcement.GetFirstPage"

const queryAnnouncementGetFirstPage = `
	SELECT 
		id, title, description, created_at, updated_at
	FROM public.announcement
	ORDER BY id DESC
	LIMIT $1
`

func (ar announcementRepository) GetFirstPage(ctx context.Context, pageSize uint64) ([]entity.Announcement, error) {
	rows, err := ar.conn(ctx).Query(ctx, queryAnnouncementGetFirstPage, pageSize)
	if err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(ctx),
			opAnnouncementGetFirstPage,
		)

		return nil, apperror.ErrRepository
	}

	return ar.getAnnouncementFromRows(ctx, rows)
}

const opAnnouncementGetPage = "infrastructure.db.postgres.Announcement.GetPage"

const queryAnnouncementGetPage = `
	SELECT 
		id, title, description, created_at, updated_at
	FROM public.announcement
	WHERE id < $1
	ORDER BY id DESC
	LIMIT $2
`

func (ar announcementRepository) GetPage(ctx context.Context, cursor, pageSize uint64) ([]entity.Announcement, error) {
	rows, err := ar.conn(ctx).Query(ctx, queryAnnouncementGetPage, cursor, pageSize)
	if err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(ctx),
			opAnnouncementGetPage,
		)

		return nil, apperror.ErrRepository
	}

	return ar.getAnnouncementFromRows(ctx, rows)
}

const opAnnouncementUpdate = "infrastructure.db.postgres.Announcement.Update"

const queryAnnouncementUpdate = `
	UPDATE public.announcement
	SET 
		title = $1,
		description = $2,
		updated_at = NOW() AT TIME ZONE 'utc';
		updated_by = $3
	WHERE 
		id = $4;
`

func (ar announcementRepository) Update(ctx context.Context, announcement entity.Announcement) error {
	resp, err := ar.conn(ctx).Exec(
		ctx, 
		queryAnnouncementUpdate, 
		announcement.Title, 
		announcement.Description,
		announcement.UpdatedBy, 
		announcement.ID,
	)
	if err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(ctx),
			opAnnouncementUpdate,
		)

		return apperror.ErrRepository
	}

	if resp.RowsAffected() < 1 {
		logs.Warn(
			apperror.ErrAnnouncementNotExists.Error(),
			common.GetCorrelationID(ctx),
			opAnnouncementUpdate,
		)

		return apperror.ErrAnnouncementNotExists
	}

	return nil
}