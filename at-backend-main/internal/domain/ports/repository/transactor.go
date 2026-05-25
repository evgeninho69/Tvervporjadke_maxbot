package repository

import "context"

type Transactor interface {
	WithinTransaction(ctx context.Context, txFunc func(ctx context.Context) error) error
}