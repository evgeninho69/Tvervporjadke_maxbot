package handlers

import (
	"activeTver/internal/delivery/http/v1/dto"
	"activeTver/internal/delivery/http/v1/responses"
	ucports "activeTver/internal/usecase/ports"
	"net/http"
)

type Categories struct {
	Service ucports.CategoriesUseCase
}

func (h Categories) Setup() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /", h.Get)

	return mux
}

func (h Categories) Get(w http.ResponseWriter, r *http.Request) {
	categories, err := h.Service.Get(r.Context())
	if err != nil {
		responses.InternalServerError(r.Context(), w)
		return
	}

	categoriesResp := dto.CategoriesResponseAndRequest{
		Categories: make([]dto.CategoryResponseAndRequest, 0, len(categories.Categories)),
	}

	for _, category := range categories.Categories {
		categoriesResp.Categories = append(categoriesResp.Categories, dto.CategoryResponseAndRequest{
			Name: category.Name,
			Subcategories: category.Subcategories,
		})
	}

	responses.JSON(
		r.Context(),
		w,
		http.StatusOK,
		categoriesResp,
	)
}
