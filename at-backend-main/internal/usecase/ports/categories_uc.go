package ucports

import (
	"activeTver/internal/domain/entity"
	"context"
)

type CategoriesUseCase interface {
	SubcategoryExist(context context.Context, subcategory string) bool

	// Get возвращает категории в формате JSON.
	Get(ctx context.Context) (*entity.Categories, error)

	Update(ctx context.Context, jsonBytes []byte) error
}