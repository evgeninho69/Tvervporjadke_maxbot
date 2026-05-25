package usecase

import (
	"activeTver/internal/domain/entity"
	"activeTver/internal/domain/ports/repository"
	"activeTver/internal/pkg/apperror"
	"activeTver/internal/pkg/common"
	"activeTver/internal/pkg/logs"
	"context"
	"encoding/json"
)

type CategoriesService struct {
	Repository    repository.CategoriesRepository
	categories    *entity.Categories
	subcategories map[string]struct{}
}

func (cs *CategoriesService) Get(ctx context.Context) (*entity.Categories, error) {
	if cs.categories == nil {
		j, err := cs.Repository.Get(ctx)
		if err != nil {
			return nil, err
		}

		var categories entity.Categories

		if err := json.Unmarshal([]byte(j), &categories); err != nil {
			logs.Error(
				err.Error(),
				common.GetCorrelationID(ctx),
				opCategoriesUpdate,
			)

			return nil, apperror.ErrUseCase
		}

		subcategories := make(map[string]struct{})
		for _, category := range categories.Categories {
			for _, subcategory := range category.Subcategories {
				subcategories[subcategory] = struct{}{}
			}
		}

		cs.subcategories = subcategories

		cs.categories = &categories
	}

	return cs.categories, nil
}

const opCategoriesUpdate = "usecase.Categories.Update"

func (cs *CategoriesService) Update(ctx context.Context, jsonBytes []byte) error {
	if !json.Valid(jsonBytes) {
		logs.Error(
			apperror.ErrBadJSON.Error(),
			common.GetCorrelationID(ctx),
			opCategoriesUpdate,
		)

		return apperror.ErrBadJSON
	}

	var categories entity.Categories

	if err := json.Unmarshal(jsonBytes, &categories); err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(ctx),
			opCategoriesUpdate,
		)

		return apperror.ErrUnprocessableJSON
	}

	subcategories := make(map[string]struct{})
	for _, category := range categories.Categories {
		for _, subcategory := range category.Subcategories {
			subcategories[subcategory] = struct{}{}
		}
	}

	if err := cs.Repository.Update(ctx, string(jsonBytes)); err != nil {
		return err
	}

	cs.subcategories = subcategories

	cs.categories = &categories

	return nil
}

func (cs *CategoriesService) SubcategoryExist(ctx context.Context, subcategory string) bool {
	if cs.subcategories == nil {
		if _, err := cs.Get(ctx); err != nil {
			return false
		}
	}

	if _, ok := cs.subcategories[subcategory]; ok {
		return true
	}

	return false
}
