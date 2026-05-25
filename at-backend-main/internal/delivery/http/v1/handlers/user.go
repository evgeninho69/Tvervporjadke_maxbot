package handlers

import (
	"activeTver/internal/delivery/http/v1/dto"
	"activeTver/internal/delivery/http/v1/responses"
	"activeTver/internal/pkg/common"
	ucports "activeTver/internal/usecase/ports"
	"net/http"
)

type User struct {
	Service ucports.UserUseCase
}

func (h User) Setup() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /", h.GetPersonalData)

	return mux
}

func (h User) GetPersonalData(w http.ResponseWriter, r *http.Request) {
	user, err := h.Service.GetByID(r.Context(), common.GetUID(r.Context()))
	if err != nil {
		responses.InternalServerError(r.Context(), w)

		return
	}

	userPersonalData := dto.EmailResponseAndRequest{
		Email: &user.Email,
	}

	responses.JSON(
		r.Context(),
		w,
		http.StatusOK,
		userPersonalData,
	)
}
