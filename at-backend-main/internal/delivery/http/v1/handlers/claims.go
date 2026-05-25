package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

	"activeTver/internal/delivery/http/v1/dto"
	"activeTver/internal/delivery/http/v1/responses"
	"activeTver/internal/domain/entity"
	"activeTver/internal/domain/types"
	"activeTver/internal/pkg/apperror"
	"activeTver/internal/pkg/common"
	"activeTver/internal/pkg/logs"
	ucports "activeTver/internal/usecase/ports"
)

var defaultMaxMemory int64 = 10 << 20

type Claims struct {
	Service ucports.ClaimsUseCase
}

func (h Claims) Setup() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /problem", h.CreateProblem)   // create problem
	mux.HandleFunc("POST /proposal", h.CreateProposal) // create proposal
	mux.HandleFunc("GET /{id}", h.GetById)             // get claim by id
	mux.HandleFunc("PUT /{id}", h.Update)              // update claim
	mux.HandleFunc("DELETE /{id}", h.Delete)           // delete claim

	mux.HandleFunc("GET /chunk", h.GetPage)

	return mux
}

const opClaimsGetById = "delivery.http.handlers.Claims.GetById"

func (h Claims) GetById(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseUint(r.PathValue("id"), 10, 64)
	if err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(r.Context()),
			opClaimsGetById,
		)

		responses.NotFound(r.Context(), w)
		return
	}

	claim, err := h.Service.GetByID(r.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, apperror.ErrClaimNotExists):
			responses.NotFound(r.Context(), w)
		default:
			responses.InternalServerError(r.Context(), w)
		}

		return
	}

	claimResp := dto.ClaimResponse{
		ID:                claim.ID,
		Title:             claim.Title,
		Description:       claim.Description,
		Category:          claim.Category,
		Status:            claim.Status,
		Photos:            claim.Photos,
		Latitude:          claim.Latitude,
		Longitude:         claim.Longitude,
		Feedback:          claim.Feedback,
		CreatedAt:         claim.CreatedAt,
		UpdatedAt:         claim.UpdatedAt,
		StatusUpdatedAt:   claim.StatusUpdatedAt,
		FeedbackUpdatedAt: claim.FeedbackUpdatedAt,
	}

	responses.JSON(
		r.Context(),
		w,
		http.StatusOK,
		claimResp,
	)
}

const opClaimsDelete = "delivery.http.handlers.Claims.Delete"

func (h Claims) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseUint(r.PathValue("id"), 10, 64)
	if err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(r.Context()),
			opClaimsDelete,
		)

		responses.NotFound(r.Context(), w)
		return
	}

	uid := common.GetUID(r.Context())

	if err := h.Service.Delete(r.Context(), id, uid, false); err != nil {
		switch {
		case errors.Is(err, apperror.ErrClaimNotExists):
			responses.NotFound(r.Context(), w)
		case errors.Is(err, apperror.ErrUserCannotPerformOperation):
			responses.Error(
				r.Context(),
				w,
				http.StatusForbidden,
				http.StatusText(http.StatusForbidden),
			)
		default:
			responses.InternalServerError(r.Context(), w)
		}

		return
	}

	w.WriteHeader(http.StatusNoContent)
}

const opClaimsCreateProblem = "delivery.http.handlers.Claims.CreateProblem"

func (h Claims) CreateProblem(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(int64(defaultMaxMemory)); err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(r.Context()),
			opClaimsCreateProblem,
		)

		responses.Error(
			r.Context(),
			w,
			http.StatusRequestEntityTooLarge,
			http.StatusText(http.StatusRequestEntityTooLarge),
		)
		return
	}

	lat, err := strconv.ParseFloat(r.FormValue("latitude"), 64)
	if err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(r.Context()),
			opClaimsCreateProblem,
		)

		responses.Error(
			r.Context(),
			w,
			http.StatusUnprocessableEntity,
			http.StatusText(http.StatusUnprocessableEntity),
		)
		return
	}

	long, err := strconv.ParseFloat(r.FormValue("longitude"), 64)
	if err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(r.Context()),
			opClaimsCreateProblem,
		)

		responses.Error(
			r.Context(),
			w,
			http.StatusUnprocessableEntity,
			http.StatusText(http.StatusUnprocessableEntity),
		)
		return
	}

	var claimDTO = dto.CreateProblemRequest{
		ClaimWithoutCategory: dto.ClaimWithoutCategory{
			Title:       r.FormValue("title"),
			Description: r.FormValue("description"),
			Longitude:   long,
			Latitude:    lat,
		},
		Category: r.FormValue("category"),
	}

	if err := claimDTO.Validate(); err != nil {
		logs.Warn(
			err.Error(),
			common.GetCorrelationID(r.Context()),
			opClaimsCreateProblem,
		)

		responses.Error(
			r.Context(),
			w,
			http.StatusUnprocessableEntity,
			err.Error(),
		)

		return
	}

	claim := entity.Claim{
		CreatedBy:   common.GetUID(r.Context()),
		Title:       claimDTO.Title,
		Description: claimDTO.Description,
		Category:    claimDTO.Category,
		Photos:      nil,
		Status:      types.ClaimStatusPending,
		Latitude:    claimDTO.Latitude,
		Longitude:   claimDTO.Longitude,
	}

	files := make([][]byte, 0, len(r.MultipartForm.File))

	for fn := range r.MultipartForm.File {
		file, _, err := r.FormFile(fn)
		if err != nil {
			logs.Error(
				err.Error(),
				common.GetCorrelationID(r.Context()),
				opClaimsCreateProblem,
			)

			continue
		}
		defer file.Close()

		buf := &bytes.Buffer{}

		if _, err := io.Copy(buf, file); err != nil {
			logs.Error(
				err.Error(),
				common.GetCorrelationID(r.Context()),
				opClaimsCreateProblem,
			)

			continue
		}

		files = append(files, buf.Bytes())
	}

	id, err := h.Service.CreateProblem(r.Context(), claim, files)
	if err != nil {
		switch {
		case errors.Is(err, apperror.ErrUnknownClaimCategory):
			responses.Error(
				r.Context(),
				w,
				http.StatusUnprocessableEntity,
				err.Error(),
			)
		case errors.Is(err, apperror.ErrPointIsNotInPolygon):
			responses.Error(
				r.Context(),
				w,
				http.StatusUnprocessableEntity,
				http.StatusText(http.StatusUnprocessableEntity),
			)
		default:
			responses.InternalServerError(r.Context(), w)
		}

		return
	}

	resp := dto.IDResponse{
		ID: id,
	}

	responses.JSON(
		r.Context(),
		w,
		http.StatusOK,
		resp,
	)
}

const opClaimsCreateProposal = "delivery.http.handlers.Claims.CreateProposal"

func (h Claims) CreateProposal(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(int64(defaultMaxMemory)); err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(r.Context()),
			opClaimsCreateProblem,
		)

		responses.Error(
			r.Context(),
			w,
			http.StatusRequestEntityTooLarge,
			http.StatusText(http.StatusRequestEntityTooLarge),
		)
		return
	}

	lat, err := strconv.ParseFloat(r.FormValue("latitude"), 64)
	if err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(r.Context()),
			opClaimsCreateProblem,
		)

		responses.Error(
			r.Context(),
			w,
			http.StatusUnprocessableEntity,
			http.StatusText(http.StatusUnprocessableEntity),
		)
		return
	}

	long, err := strconv.ParseFloat(r.FormValue("longitude"), 64)
	if err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(r.Context()),
			opClaimsCreateProblem,
		)

		responses.Error(
			r.Context(),
			w,
			http.StatusUnprocessableEntity,
			http.StatusText(http.StatusUnprocessableEntity),
		)
		return
	}

	var claimDTO = dto.CreateProposalRequest{
		ClaimWithoutCategory: dto.ClaimWithoutCategory{
			Title:       r.FormValue("title"),
			Description: r.FormValue("description"),
			Longitude:   long,
			Latitude:    lat,
		},
	}

	if err := claimDTO.Validate(); err != nil {
		logs.Warn(
			err.Error(),
			common.GetCorrelationID(r.Context()),
			opClaimsCreateProposal,
		)

		responses.Error(
			r.Context(),
			w,
			http.StatusUnprocessableEntity,
			err.Error(),
		)

		return
	}

	claim := entity.Claim{
		CreatedBy:   common.GetUID(r.Context()),
		Title:       claimDTO.Title,
		Description: claimDTO.Description,
		Latitude:    claimDTO.Latitude,
		Longitude:   claimDTO.Longitude,
	}

	id, err := h.Service.CreateProposal(r.Context(), claim)
	if err != nil {
		switch {
		case errors.Is(err, apperror.ErrPointIsNotInPolygon):
			responses.Error(
				r.Context(),
				w,
				http.StatusUnprocessableEntity,
				http.StatusText(http.StatusUnprocessableEntity),
			)
		default:
			responses.InternalServerError(r.Context(), w)
		}

		return
	}

	resp := dto.IDResponse{
		ID: id,
	}

	responses.JSON(
		r.Context(),
		w,
		http.StatusOK,
		resp,
	)
}

func sliceClaimsEntityToSliceClaimsPaginationDTO(page []entity.Claim) []dto.ClaimForPaginationResponse {
	pageResp := make([]dto.ClaimForPaginationResponse, 0, len(page))
	for _, claim := range page {
		claimResp := dto.ClaimForPaginationResponse{
			ID:          claim.ID,
			Title:       claim.Title,
			Description: claim.Description,
			Category:    claim.Category,
			Status:      claim.Status,
			Latitude:    claim.Latitude,
			Longitude:   claim.Longitude,
			CreatedAt:   claim.CreatedAt,
			UpdatedAt:   claim.UpdatedAt,
		}

		pageResp = append(pageResp, claimResp)
	}

	return pageResp
}

const opClaimsGetPage = "delivery.http.handlers.Claims.GetPage"

func (h Claims) GetPage(w http.ResponseWriter, r *http.Request) {
	pageSize := common.GetPageSize(r.URL.Query())

	cursorString := r.URL.Query().Get("cursor")
	if cursorString == "" {
		h.GetFirstPage(w, r, pageSize)
		return
	}

	cursor, err := strconv.ParseUint(cursorString, 10, 64)
	if err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(r.Context()),
			opClaimsGetPage,
		)

		responses.Error(
			r.Context(),
			w,
			http.StatusBadRequest,
			http.StatusText(http.StatusBadRequest),
		)

		return
	}

	uid := common.GetUID(r.Context())

	page, err := h.Service.GetUserPage(r.Context(), cursor, uid, pageSize)
	if err != nil {
		responses.InternalServerError(r.Context(), w)

		return
	}

	pageResp := sliceClaimsEntityToSliceClaimsPaginationDTO(page)

	resp := make(map[string]interface{})
	resp["claims"] = pageResp

	responses.JSON(
		r.Context(),
		w,
		http.StatusOK,
		resp,
	)
}

func (h Claims) GetFirstPage(w http.ResponseWriter, r *http.Request, pageSize uint64) {
	uid := common.GetUID(r.Context())

	page, err := h.Service.GetFirstUserPage(r.Context(), uid, pageSize)
	if err != nil {
		responses.InternalServerError(r.Context(), w)

		return
	}

	pageResp := sliceClaimsEntityToSliceClaimsPaginationDTO(page)

	resp := make(map[string]interface{})
	resp["claims"] = pageResp

	responses.JSON(
		r.Context(),
		w,
		http.StatusOK,
		resp,
	)
}

// TODO: Необходимо уточнить поля, которые можно будет менять

const opClaimsUpdate = "delivery.http.handlers.Claims.Update"

func (h Claims) Update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseUint(r.PathValue("id"), 10, 64)
	if err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(r.Context()),
			opClaimsUpdate,
		)

		responses.NotFound(r.Context(), w)
		return
	}

	var claimDTO dto.UpdateClaimRequest

	if err = json.NewDecoder(r.Body).Decode(&claimDTO); err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(r.Context()),
			opClaimsUpdate,
		)

		responses.Error(
			r.Context(),
			w,
			http.StatusBadRequest,
			http.StatusText(http.StatusBadRequest),
		)

		return
	}

	if err = claimDTO.Validate(); err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(r.Context()),
			opClaimsUpdate,
		)

		responses.Error(
			r.Context(),
			w,
			http.StatusUnprocessableEntity,
			err.Error(),
		)
		return
	}

	uid := common.GetUID(r.Context())

	claim := entity.Claim{
		ID:          id,
		CreatedBy:   uid,
		Title:       claimDTO.Title,
		Description: claimDTO.Description,
		Status:      types.ClaimStatusPending,
		Latitude:    claimDTO.Latitude,
		Longitude:   claimDTO.Longitude,
	}

	if err := h.Service.Update(r.Context(), claim); err != nil {
		switch {
		case errors.Is(err, apperror.ErrUnknownClaimCategory):
			responses.Error(
				r.Context(),
				w,
				http.StatusUnprocessableEntity,
				err.Error(),
			)
		case errors.Is(err, apperror.ErrUserCannotPerformOperation):
			responses.Error(
				r.Context(),
				w,
				http.StatusForbidden,
				http.StatusText(http.StatusForbidden),
			)
		case errors.Is(err, apperror.ErrClaimNotExists):
			responses.NotFound(r.Context(), w)
		default:
			responses.InternalServerError(r.Context(), w)
		}

		return
	}

	w.WriteHeader(http.StatusNoContent)
}
