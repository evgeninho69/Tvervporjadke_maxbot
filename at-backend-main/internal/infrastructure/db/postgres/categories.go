package postgres

import (
	"activeTver/internal/pkg/apperror"
	"activeTver/internal/pkg/common"
	"activeTver/internal/pkg/logs"
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type categoriesRepository struct {
	baseRepository
}

func NewCategoriesRepository(pool *pgxpool.Pool) *categoriesRepository {
	return &categoriesRepository{
		baseRepository{
			pool: pool,
		},
	}
}

const opCategoriesUpdate = "infrastructure.db.postgres.Categories.Update"

const queryCategoriesUpdate = `
	UPDATE public.categories
	SET
		categories = $1
`

func (cr categoriesRepository) Update(ctx context.Context, json string) error {
	if _, err := cr.conn(ctx).Exec(ctx, queryCategoriesUpdate, json); err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(ctx),
			opCategoriesUpdate,
		)

		return apperror.ErrRepository
	}

	// TODO: Нужна обработка случая, если каким то образом таблица категорий оказалась пустой

	return nil
}

const opCategoriesGet = "infrastructure.db.postgres.Categories.Get"

const queryCategoriesGet = `
	SELECT categories FROM public.categories
`

func (cr categoriesRepository) Get(ctx context.Context) (string, error) {
	var json string

	if err := cr.conn(ctx).QueryRow(ctx, queryCategoriesGet).Scan(&json); err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(ctx),
			opCategoriesGet,
		)

		return "", apperror.ErrRepository
	}

	return json, nil
}
