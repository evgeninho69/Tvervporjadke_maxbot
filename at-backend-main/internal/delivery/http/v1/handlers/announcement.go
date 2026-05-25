package handlers

import (
	"activeTver/internal/delivery/http/v1/dto"
	"activeTver/internal/delivery/http/v1/responses"
	"activeTver/internal/domain/entity"
	"activeTver/internal/pkg/apperror"
	"activeTver/internal/pkg/common"
	"activeTver/internal/pkg/logs"
	ucports "activeTver/internal/usecase/ports"
	"errors"
	"net/http"
	"strconv"
)

type Announcement struct {
	Service ucports.AnnouncementUseCase
}

func (h Announcement) Setup() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /{id}", h.GetByID)
	mux.HandleFunc("GET /chunk", h.GetPage)

	return mux
}

const opAnnouncementGetByID = "delivery.http.handlers.Announcement.GetByID"

func (h Announcement) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseUint(r.PathValue("id"), 10, 64)
	if err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(r.Context()),
			opAnnouncementGetByID,
		)

		responses.NotFound(r.Context(), w)
		return
	}

	announcement, err := h.Service.GetByID(r.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, apperror.ErrAnnouncementNotExists):
			responses.NotFound(r.Context(), w)
		default:
			responses.InternalServerError(r.Context(), w)
		}

		return
	}

	announcementResp := dto.AnnouncementResponse{
		ID:          announcement.ID,
		Title:       announcement.Title,
		Description: announcement.Description,
		CreatedAt:   announcement.CreatedAt,
	}

	responses.JSON(
		r.Context(),
		w,
		http.StatusOK,
		announcementResp,
	)
}

func (h Announcement) pageToRespPage(page []entity.Announcement) []dto.AnnouncementResponse {
	pageResp := make([]dto.AnnouncementResponse, 0, len(page))
	for _, announcement := range page {
		announcementResp := dto.AnnouncementResponse{
			ID:          announcement.ID,
			Title:       announcement.Title,
			Description: announcement.Description,
			CreatedAt:   announcement.CreatedAt,
		}

		pageResp = append(pageResp, announcementResp)
	}

	return pageResp
}

func (h Announcement) GetFirstPage(w http.ResponseWriter, r *http.Request, pageSize uint64) {
	page, err := h.Service.GetFirstPage(r.Context(), pageSize)
	if err != nil {
		responses.InternalServerError(r.Context(), w)
		return
	}

	announcementsResp := h.pageToRespPage(page)

	resp := make(map[string]any)
	resp["announcements"] = announcementsResp

	responses.JSON(
		r.Context(),
		w,
		http.StatusOK,
		resp,
	)
}

const opAnnouncementGetPage = "delivery.http.handlers.Announcement.GetPage"

func (h Announcement) GetPage(w http.ResponseWriter, r *http.Request) {
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
			opAnnouncementGetPage,
		)

		responses.Error(
			r.Context(),
			w,
			http.StatusBadRequest,
			http.StatusText(http.StatusBadRequest),
		)
		return
	}

	page, err := h.Service.GetPage(r.Context(), cursor, pageSize)
	if err != nil {
		responses.InternalServerError(r.Context(), w)
		return
	}

	announcementsResp := h.pageToRespPage(page)

	resp := make(map[string]interface{})
	resp["announcements"] = announcementsResp

	responses.JSON(
		r.Context(),
		w,
		http.StatusOK,
		resp,
	)
}
