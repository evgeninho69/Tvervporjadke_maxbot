package repository

import "context"

type CategoriesRepository interface {
	// Update обновляет категории в базе данных.
	// На вход получает JSON.
	Update(ctx context.Context, json string) error

	// Get возвращает JSON, содержащий категории.
	Get(ctx context.Context) (string, error)
}
