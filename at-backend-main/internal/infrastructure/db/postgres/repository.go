package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type baseRepository struct {
	pool *pgxpool.Pool
}

func (repo *baseRepository) conn(ctx context.Context) Executor {
	if tx := extractTx(ctx); tx != nil {
		return tx
	}

	return repo.pool
}
