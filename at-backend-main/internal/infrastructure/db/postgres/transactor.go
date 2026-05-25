package postgres

import (
	"activeTver/internal/pkg/apperror"
	"activeTver/internal/pkg/common"
	"activeTver/internal/pkg/logs"
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
)

func NewTransactor(pool *pgxpool.Pool) *transactor {
	return &transactor{
		Pool: pool,
	}
}

type transactor struct {
	Pool *pgxpool.Pool
}

const opTransactor = "infrastructure.db.postgres.Transactor"

func (t transactor) WithinTransaction(ctx context.Context, txFunc func(ctx context.Context) error) error {
	tx, err := t.Pool.Begin(ctx)
	if err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(ctx),
			opTransactor,
		)

		return apperror.ErrRepository
	}

	logs.Info(
		"transaction started",
		common.GetCorrelationID(ctx),
		opTransactor,
	)

	injectTx(ctx, tx)

	txFuncErr := txFunc(ctx)
	if errors.Is(txFuncErr, apperror.ErrRepository) {
		if err := tx.Rollback(ctx); err != nil {
			logs.Error(
				err.Error(),
				common.GetCorrelationID(ctx),
				opTransactor,
			)

			return apperror.ErrRepository
		}

		logs.Info(
			"transaction canceled",
			common.GetCorrelationID(ctx),
			opTransactor,
		)

		return err
	}

	if err := tx.Commit(ctx); err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(ctx),
			opTransactor,
		)

		logs.Info(
			"transaction canceled",
			common.GetCorrelationID(ctx),
			opTransactor,
		)

		return err
	}

	logs.Info(
		"transaction committed",
		common.GetCorrelationID(ctx),
		opTransactor,
	)

	return txFuncErr
}
