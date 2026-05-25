package postgres

import (
	"activeTver/internal/pkg/common"
	"context"

	"github.com/jackc/pgx/v5"
)

const txCtxKey = "executor"

func injectTx(ctx context.Context, tx pgx.Tx) {
	common.SetValueIntoContext(ctx, txCtxKey, tx)
}

func extractTx(ctx context.Context) pgx.Tx {
	if tx, ok := common.GetValueFromContext(ctx, txCtxKey).(pgx.Tx); ok {
		return tx
	}

	return nil
}
