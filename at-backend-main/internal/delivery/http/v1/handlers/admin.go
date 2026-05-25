package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

	"activeTver/internal/delivery/http/v1/dto"
	"activeTver/internal/delivery/http/v1/responses"
	"activeTver/internal/domain/entity"
	"activeTver/internal/pkg/apperror"
	"activeTver/internal/pkg/common"
	"activeTver/internal/pkg/logs"
	ucports "activeTver/internal/usecase/ports"
)

type Admin struct {
	ClaimsService       ucports.ClaimsUseCase
	AnnouncementService ucports.AnnouncementUseCase
	AdminService        ucports.AdminUseCase
	UserService         ucports.UserUseCase
	CategoriesService   ucports.CategoriesUseCase
}

func (h Admin) Setup() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /claims/{id}", h.GetClaimByID)
	mux.HandleFunc("GET /claims/chunk", h.GetPageClaims)
	mux.HandleFunc("POST /claims/{id}/feedback", h.AddFeedbackToClaim)
	mux.HandleFunc("POST /claims/{id}/status", h.ChangeClaimStatus)
	mux.HandleFunc("DELETE /claims/{id}", h.DeleteClaim)

	mux.HandleFunc("GET /admins", h.GetListAdmins)
	mux.HandleFunc("POST /admins/add", h.SetAdmin)
	mux.HandleFunc("POST /admins/remove", h.DeleteAdmin)
	mux.HandleFunc("GET /admins/privileges", h.GetPrivileges)

	mux.HandleFunc("GET /announcements/{id}", h.GetAnnouncementByID)
	mux.HandleFunc("POST /announcements", h.CreateAnnouncement)
	mux.HandleFunc("DELETE /announcements/{id}", h.DeleteAnnouncement)
	mux.HandleFunc("PUT /announcement/{id}", h.UpdateAnnouncement)

	mux.HandleFunc("POST /users/search", h.GetUserByEmail)
	mux.HandleFunc("GET /users/{uid}/", h.GetUserByUID)
	mux.HandleFunc("GET /users/{uid}/check-block", h.CheckUserBlock)
	mux.HandleFunc("GET /users/blocked", h.BlockedUsersPage)
	mux.HandleFunc("POST /users/{uid}/block", h.BlockUser)
	mux.HandleFunc("POST /users/{uid}/unblock", h.UnblockUser)
	mux.HandleFunc("GET /users/{uid}/claims", h.GetUserClaimsPage)

	mux.HandleFunc("PUT /categories/", h.CategoriesUpdate)

	return mux
}

func (h Admin) superAdmin(w http.ResponseWriter, r *http.Request, op string) bool {
	if !common.GetAdminData(r.Context()).IsSuperAdmin {
		logs.Warn(
			"access denied, user is not SuperAdmin",
			common.GetCorrelationID(r.Context()),
			op,
		)

		responses.Error(
			r.Context(),
			w,
			http.StatusForbidden,
			http.StatusText(http.StatusForbidden),
		)

		return false
	}

	return true
}

// ! Users

func (h Admin) getUIDFromPath(w http.ResponseWriter, r *http.Request, op string) (uint64, error) {
	uidString := r.PathValue("uid")
	if uidString == "" {
		responses.NotFound(r.Context(), w)
		return 0, errors.New("uid not found in path")
	}

	uid, err := strconv.ParseUint(uidString, 10, 64)
	if err != nil {
		logs.Warn(
			err.Error(),
			common.GetCorrelationID(r.Context()),
			op,
		)

		responses.NotFound(r.Context(), w)
		return 0, errors.New("uid has type different from uint64")
	}

	return uid, nil
}

const opAdminGetUserByEmail = "delivery.http.handlers.Admin.GetUserByEmail"

func (h Admin) GetUserByEmail(w http.ResponseWriter, r *http.Request) {
	var emailDTO dto.EmailResponseAndRequest

	if err := json.NewDecoder(r.Body).Decode(&emailDTO); err != nil {
		logs.Warn(
			err.Error(),
			common.GetCorrelationID(r.Context()),
			opAdminGetUserByEmail,
		)

		responses.Error(
			r.Context(),
			w,
			http.StatusBadRequest,
			http.StatusText(http.StatusBadRequest),
		)

		return
	}

	if err := emailDTO.Validate(); err != nil {
		logs.Warn(
			err.Error(),
			common.GetCorrelationID(r.Context()),
			opAdminGetUserByEmail,
		)

		responses.Error(
			r.Context(),
			w,
			http.StatusUnprocessableEntity,
			err.Error(),
		)

		return
	}

	user, err := h.UserService.GetByEmail(r.Context(), *emailDTO.Email)
	if err != nil {
		switch {
		case errors.Is(err, apperror.ErrUserNotExists):
			responses.NotFound(r.Context(), w)
		default:
			responses.InternalServerError(r.Context(), w)
		}

		return
	}

	resp := dto.UserResponse{
		UserIDResponse: dto.UserIDResponse{
			UID: user.ID,
		},
		EmailResponseAndRequest: dto.EmailResponseAndRequest{
			Email:&user.Email,
		},
		RegisteredAt: user.RegisteredAt,
	}

	responses.JSON(
		r.Context(),
		w,
		http.StatusOK,
		resp,
	)
}

const opAdminGetUserByUID = "delivery.http.handlers.Admin.GetUserByUID"

func (h Admin) GetUserByUID(w http.ResponseWriter, r *http.Request) {
	uid, err := h.getUIDFromPath(w, r, opAdminGetUserByUID)
	if err != nil {
		return
	}

	user, err := h.UserService.GetByID(r.Context(), uid)
	if err != nil {
		switch {
		case errors.Is(err, apperror.ErrUserNotExists):
			responses.NotFound(r.Context(), w)
		default:
			responses.InternalServerError(r.Context(), w)
		}

		return
	}

	resp := dto.UserResponse{
		UserIDResponse: dto.UserIDResponse{
			UID: user.ID,
		},
		EmailResponseAndRequest: dto.EmailResponseAndRequest{
			Email: &user.Email,
		},
		RegisteredAt: user.RegisteredAt,
	}

	responses.JSON(
		r.Context(),
		w,
		http.StatusOK,
		resp,
	)
}

const opAdminCheckUserBlock = "delivery.http.handlers.Admin.Users.CheckBlock"

func (h Admin) CheckUserBlock(w http.ResponseWriter, r *http.Request) {
	uid, err := h.getUIDFromPath(w, r, opAdminCheckUserBlock)
	if err != nil {
		return
	}

	blockedUser, err := h.UserService.GetBlockedUser(r.Context(), uid)
	if err != nil {
		switch err {
		case apperror.ErrUserNotBlocked:
			responses.NotFound(r.Context(), w)
		default:
			responses.InternalServerError(r.Context(), w)
		}

		return
	}

	resp := dto.BlockDataResponse{
		Reason:    blockedUser.Reason,
		BlockedBy: blockedUser.BlockedBy,
		BlockedAt: blockedUser.BlockedAt,
	}

	responses.JSON(
		r.Context(),
		w,
		http.StatusOK,
		resp,
	)
}

func (h Admin) blockedUserEntityToBlockedDataPaginationDTO(blockedData entity.BlockedUser) dto.BlockDataForPaginationResponse {
	return dto.BlockDataForPaginationResponse{
		ID:        blockedData.ID,
		BlockedAt: blockedData.BlockedAt,
		EmailResponseAndRequest: dto.EmailResponseAndRequest{
			Email: &blockedData.Email,
		},
		UserIDResponse: dto.UserIDResponse{
			UID: blockedData.UID,
		},
	}
}

func (h Admin) sliceBlockedUserEntityToSliceBlockedDataPaginationDTO(slice []entity.BlockedUser) []dto.BlockDataForPaginationResponse {
	sliceBlockedUserDataDTO := make([]dto.BlockDataForPaginationResponse, 0, len(slice))

	for i := range len(slice) {
		blockedUserDTO := h.blockedUserEntityToBlockedDataPaginationDTO(slice[i])

		sliceBlockedUserDataDTO = append(sliceBlockedUserDataDTO, blockedUserDTO)
	}

	return sliceBlockedUserDataDTO
}

const opAdminBlockedUsersFirstPage = "delivery.http.handlers.Admin.Users.BlockedUsersFirstPage"

func (h Admin) BlockedUsersFirstPage(w http.ResponseWriter, r *http.Request, pageSize uint64) {
	entities, err := h.UserService.BlockedUsersFirstPage(r.Context(), pageSize)
	if err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(r.Context()),
			opAdminBlockedUsersFirstPage,
		)

		responses.InternalServerError(r.Context(), w)

		return
	}

	page := h.sliceBlockedUserEntityToSliceBlockedDataPaginationDTO(entities)

	resp := make(map[string]any)
	resp["blocked_users"] = page

	responses.JSON(
		r.Context(),
		w,
		http.StatusOK,
		resp,
	)
}

const opAdminBlockedUsersPage = "delivery.http.handlers.Admin.Users.BlockedUsersPage"

func (h Admin) BlockedUsersPage(w http.ResponseWriter, r *http.Request) {
	pageSize := common.GetPageSize(r.URL.Query())

	cursorString := r.URL.Query().Get("cursor")
	if cursorString == "" {
		h.BlockedUsersFirstPage(w, r, pageSize)

		return
	}

	cursor, err := strconv.ParseUint(cursorString, 10, 64)
	if err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(r.Context()),
			opAdminBlockedUsersPage,
		)

		responses.Error(
			r.Context(),
			w,
			http.StatusBadRequest,
			http.StatusText(http.StatusBadRequest),
		)

		return
	}

	page, err := h.UserService.BlockedUsersPage(r.Context(), cursor, pageSize)
	if err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(r.Context()),
			opAdminBlockedUsersPage,
		)

		responses.InternalServerError(r.Context(), w)

		return
	}

	resp := make(map[string]any)
	resp["blocked_users"] = page

	responses.JSON(
		r.Context(),
		w,
		http.StatusOK,
		resp,
	)
}

const opAdminBlockUser = "delivery.http.handlers.Admin.Users.BlockUser"

func (h Admin) BlockUser(w http.ResponseWriter, r *http.Request) {
	uid, err := h.getUIDFromPath(w, r, opAdminBlockUser)
	if err != nil {
		return
	}

	var blockRequestDTO dto.BlockRequest

	if err := json.NewDecoder(r.Body).Decode(&blockRequestDTO); err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(r.Context()),
			opAdminBlockUser,
		)

		responses.Error(
			r.Context(),
			w,
			http.StatusBadRequest,
			http.StatusText(http.StatusBadRequest),
		)

		return
	}

	if err := blockRequestDTO.Validate(); err != nil {
		logs.Warn(
			err.Error(),
			common.GetCorrelationID(r.Context()),
			opAdminBlockUser,
		)

		responses.Error(
			r.Context(),
			w,
			http.StatusUnprocessableEntity,
			err.Error(),
		)

		return
	}

	adminUID := common.GetUID(r.Context())

	if err := h.UserService.BlockByUID(r.Context(), adminUID, uid, *blockRequestDTO.Reason); err != nil {
		switch {
		case errors.Is(err, apperror.ErrUserNotExists):
			responses.NotFound(r.Context(), w)
		case errors.Is(err, apperror.ErrUserAlreadyBlocked):
			responses.Error(
				r.Context(),
				w,
				http.StatusConflict,
				err.Error(),
			)
		default:
			responses.InternalServerError(r.Context(), w)
		}

		return
	}

	w.WriteHeader(http.StatusNoContent)
}

const opAdminUnblockUser = "delivery.http.handlers.Admin.UnblockUser"

func (h Admin) UnblockUser(w http.ResponseWriter, r *http.Request) {
	uid, err := h.getUIDFromPath(w, r, opAdminUnblockUser)
	if err != nil {
		return
	}

	if err := h.UserService.UnblockByUID(r.Context(), uid); err != nil {
		switch {
		case errors.Is(err, apperror.ErrUserNotBlocked):
			responses.Error(
				r.Context(),
				w,
				http.StatusConflict,
				err.Error(),
			)
		default:
			responses.InternalServerError(r.Context(), w)
		}

		return
	}

	w.WriteHeader(http.StatusNoContent)
}

const opAdminGetUserClaimsPage = "delivery.http.handlers.Admin.GetUsersClaimsPage"

func (h Admin) GetUserClaimsPage(w http.ResponseWriter, r *http.Request) {
	uid, err := h.getUIDFromPath(w, r, opAdminGetUserClaimsPage)
	if err != nil {
		return
	}

	pageSize := common.GetPageSize(r.URL.Query())

	cursorString := r.URL.Query().Get("cursor")
	if cursorString == "" {
		h.GetUserClaimsFirstPage(w, r, pageSize, uid)
		return
	}

	cursor, err := strconv.ParseUint(cursorString, 10, 64)
	if err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(r.Context()),
			opAdminGetUserClaimsPage,
		)

		responses.Error(
			r.Context(),
			w,
			http.StatusBadRequest,
			http.StatusText(http.StatusBadRequest),
		)

		return
	}

	page, err := h.ClaimsService.GetUserPage(r.Context(), cursor, uid, pageSize)
	if err != nil {
		responses.InternalServerError(r.Context(), w)

		return
	}

	pageResp := sliceClaimsEntityToSliceClaimsPaginationDTO(page)

	resp := map[string]any{
		"claims": pageResp,
	}

	responses.JSON(
		r.Context(),
		w,
		http.StatusOK,
		resp,
	)
}

func (h Admin) GetUserClaimsFirstPage(w http.ResponseWriter, r *http.Request, pageSize, uid uint64) {
	page, err := h.ClaimsService.GetFirstUserPage(r.Context(), uid, pageSize)
	if err != nil {
		responses.InternalServerError(r.Context(), w)

		return
	}

	pageResp := sliceClaimsEntityToSliceClaimsPaginationDTO(page)

	resp := map[string]any{
		"claims": pageResp,
	}

	responses.JSON(
		r.Context(),
		w,
		http.StatusOK,
		resp,
	)
}

// ! Claims

func (h Admin) claimEntityToClaimPaginationDTO(claim entity.Claim) dto.ClaimForPaginationResponse {
	claimDTO := dto.ClaimForPaginationResponse{
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

	return claimDTO
}

func (h Admin) sliceClaimsEntitiesToSliceClaimsDTO(page []entity.Claim) []dto.ClaimForPaginationResponse {
	sliceClaimsDTO := make([]dto.ClaimForPaginationResponse, 0, len(page))
	for _, claim := range page {
		claimDTO := h.claimEntityToClaimPaginationDTO(claim)

		sliceClaimsDTO = append(sliceClaimsDTO, claimDTO)
	}

	return sliceClaimsDTO
}

const opAdminGetClaimByID = "delivery.http.handlers.Admin.Claims.GetById"

func (h Admin) GetClaimByID(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseUint(r.PathValue("id"), 10, 64)
	if err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(r.Context()),
			opAdminGetClaimByID,
		)

		responses.NotFound(r.Context(), w)

		return
	}

	claim, email, err := h.ClaimsService.GetByIDWithUserEmail(r.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, apperror.ErrClaimNotExists):
			responses.NotFound(r.Context(), w)
		default:
			responses.InternalServerError(r.Context(), w)
		}

		return
	}

	claimResp := dto.ClaimForAdminsResponse{
		ClaimResponse: dto.ClaimResponse{
			ID:                id,
			Title:             claim.Title,
			Description:       claim.Description,
			Category:          claim.Category,
			Status:            claim.Status,
			Feedback:          claim.Feedback,
			Photos:            claim.Photos,
			Latitude:          claim.Latitude,
			Longitude:         claim.Longitude,
			CreatedAt:         claim.CreatedAt,
			UpdatedAt:         claim.UpdatedAt,
			StatusUpdatedAt:   claim.StatusUpdatedAt,
			FeedbackUpdatedAt: claim.FeedbackUpdatedAt,
		},
		Email:     email,
		CreatedBy: claim.CreatedBy,
	}

	responses.JSON(
		r.Context(),
		w,
		http.StatusOK,
		claimResp,
	)
}

const opAdminChangeClaimStatus = "delivery.http.handlers.Admin.Claims.ChangeClaimStatus"

func (h Admin) ChangeClaimStatus(w http.ResponseWriter, r *http.Request) {
	var statusDTO dto.ClaimStatusUpdateRequest

	id, err := strconv.ParseUint(r.PathValue("id"), 10, 64)
	if err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(r.Context()),
			opAdminChangeClaimStatus,
		)

		responses.NotFound(r.Context(), w)

		return
	}

	if err := json.NewDecoder(r.Body).Decode(&statusDTO); err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(r.Context()),
			opAdminChangeClaimStatus,
		)

		responses.Error(
			r.Context(),
			w,
			http.StatusBadRequest,
			http.StatusText(http.StatusBadRequest),
		)

		return
	}

	if err := statusDTO.Validate(); err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(r.Context()),
			opAdminChangeClaimStatus,
		)

		responses.Error(
			r.Context(),
			w,
			http.StatusUnprocessableEntity,
			err.Error(),
		)

		return
	}

	if err := h.ClaimsService.ChangeStatus(r.Context(), id, *statusDTO.Status); err != nil {
		switch {
		case errors.Is(err, apperror.ErrUnknownClaimStatus):
			responses.Error(
				r.Context(),
				w,
				http.StatusUnprocessableEntity,
				err.Error(),
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

func (h Admin) GetPageClaims(w http.ResponseWriter, r *http.Request) {
	pageSize := common.GetPageSize(r.URL.Query())

	cursorString := r.URL.Query().Get("cursor")
	if cursorString == "" {
		h.GetFirstPageClaims(w, r, pageSize)
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

	page, err := h.ClaimsService.GetPage(r.Context(), cursor, pageSize)
	if err != nil {
		responses.InternalServerError(r.Context(), w)

		return
	}

	pageResp := h.sliceClaimsEntitiesToSliceClaimsDTO(page)

	resp := make(map[string]interface{})
	resp["claims"] = pageResp

	responses.JSON(
		r.Context(),
		w,
		http.StatusOK,
		resp,
	)
}

func (h Admin) GetFirstPageClaims(w http.ResponseWriter, r *http.Request, pageSize uint64) {
	page, err := h.ClaimsService.GetFirstPage(r.Context(), pageSize)
	if err != nil {
		responses.InternalServerError(r.Context(), w)

		return
	}

	pageResp := h.sliceClaimsEntitiesToSliceClaimsDTO(page)

	resp := make(map[string]interface{})
	resp["claims"] = pageResp

	responses.JSON(
		r.Context(),
		w,
		http.StatusOK,
		resp,
	)
}

const opAdminAddFeedbackToClaim = "delivery.http.handlers.Admin.Claims.AddFeedbackToClaim"

func (h Admin) AddFeedbackToClaim(w http.ResponseWriter, r *http.Request) {
	var feedbackDTO dto.ClaimFeedbackRequest

	id, err := strconv.ParseUint(r.PathValue("id"), 10, 64)
	if err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(r.Context()),
			opAdminAddFeedbackToClaim,
		)

		responses.NotFound(r.Context(), w)

		return
	}

	if err := json.NewDecoder(r.Body).Decode(&feedbackDTO); err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(r.Context()),
			opAdminAddFeedbackToClaim,
		)

		responses.Error(
			r.Context(),
			w,
			http.StatusBadRequest,
			http.StatusText(http.StatusBadRequest),
		)

		return
	}

	if err := feedbackDTO.Validate(); err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(r.Context()),
			opAdminAddFeedbackToClaim,
		)

		responses.Error(
			r.Context(),
			w,
			http.StatusUnprocessableEntity,
			err.Error(),
		)

		return
	}

	if err := h.ClaimsService.AddFeedback(r.Context(), id, *feedbackDTO.Feedback); err != nil {
		switch {
		case errors.Is(err, apperror.ErrClaimNotExists):
			responses.NotFound(r.Context(), w)
		default:
			responses.InternalServerError(r.Context(), w)
		}

		return
	}

	w.WriteHeader(http.StatusNoContent)
}

const opAdminDeleteClaim = "delivery.http.handlers.Admin.Claims.DeleteClaim"

func (h Admin) DeleteClaim(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseUint(r.PathValue("id"), 10, 64)
	if err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(r.Context()),
			opAdminDeleteClaim,
		)

		responses.NotFound(r.Context(), w)

		return
	}

	uid := common.GetUID(r.Context())

	if err := h.ClaimsService.Delete(r.Context(), id, uid, true); err != nil {
		switch {
		case errors.Is(err, apperror.ErrClaimNotExists):
			responses.NotFound(r.Context(), w)
		default:
			responses.InternalServerError(r.Context(), w)
		}

		return

	}

	w.WriteHeader(http.StatusNoContent)
}

// ! Announcements

const opAdminCreateAnnouncement = "delivery.http.handlers.Admin.Announcement.CreateAnnouncement"

func (h Admin) CreateAnnouncement(w http.ResponseWriter, r *http.Request) {
	var createAnnouncementDTO dto.CreateAnnouncementRequest

	if err := json.NewDecoder(r.Body).Decode(&createAnnouncementDTO); err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(r.Context()),
			opAdminCreateAnnouncement,
		)

		responses.Error(
			r.Context(),
			w,
			http.StatusBadRequest,
			http.StatusText(http.StatusBadRequest),
		)

		return
	}

	if err := createAnnouncementDTO.Validate(); err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(r.Context()),
			opAdminCreateAnnouncement,
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

	announcement := entity.Announcement{
		Title:       *createAnnouncementDTO.Title,
		Description: *createAnnouncementDTO.Description,
		CreatedBy:   uid,
	}

	id, err := h.AnnouncementService.Create(r.Context(), announcement)
	if err != nil {
		responses.InternalServerError(r.Context(), w)
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

const opAdminDeleteAnnouncement = "delivery.http.handlers.Admin.Announcement.DeleteAnnouncement"

func (h Admin) DeleteAnnouncement(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseUint(r.PathValue("id"), 10, 64)
	if err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(r.Context()),
			opAdminDeleteAnnouncement,
		)

		responses.NotFound(r.Context(), w)

		return
	}

	if err := h.AnnouncementService.Delete(r.Context(), id); err != nil {
		switch {
		case errors.Is(err, apperror.ErrAnnouncementNotExists):
			responses.NotFound(r.Context(), w)
		default:
			responses.InternalServerError(r.Context(), w)
		}

		return
	}

	w.WriteHeader(http.StatusNoContent)
}

const opAdminUpdateAnnouncement = "delivery.http.handlers.Admin.Announcement.UpdateAnnouncement"

func (h Admin) UpdateAnnouncement(w http.ResponseWriter, r *http.Request) {
	announcementID, err := strconv.ParseUint(r.PathValue("id"), 10, 64)
	if err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(r.Context()),
			opAdminDeleteAnnouncement,
		)

		responses.NotFound(r.Context(), w)

		return
	}

	var dto dto.AnnouncementUpdateRequest

	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		logs.Warn(
			err.Error(),
			common.GetCorrelationID(r.Context()),
			opAdminUpdateAnnouncement,
		)

		responses.Error(
			r.Context(),
			w,
			http.StatusBadRequest,
			http.StatusText(http.StatusBadRequest),
		)

		return
	}

	if err := dto.Validate(); err != nil {
		logs.Warn(
			err.Error(),
			common.GetCorrelationID(r.Context()),
			opAdminUpdateAnnouncement,
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

	announcement := entity.Announcement{
		ID:          announcementID,
		Title:       *dto.Title,
		Description: *dto.Description,
		UpdatedBy:   uid,
	}

	if err := h.AnnouncementService.Update(r.Context(), announcement); err != nil {
		switch {
		case errors.Is(err, apperror.ErrAnnouncementNotExists):
			responses.NotFound(r.Context(), w)
		default:
			responses.InternalServerError(r.Context(), w)
		}

		return
	}

	w.WriteHeader(http.StatusNoContent)
}

const opAdminGetAnnouncementByID = "delivery.http.handlers.Admin.Announcement.GetAnnouncementByID"

func (h Admin) GetAnnouncementByID(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseUint(r.PathValue("id"), 10, 64)
	if err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(r.Context()),
			opAdminGetAnnouncementByID,
		)

		responses.NotFound(r.Context(), w)

		return
	}

	announcement, err := h.AnnouncementService.GetByID(r.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, apperror.ErrAnnouncementNotExists):
			responses.NotFound(r.Context(), w)
		default:
			responses.InternalServerError(r.Context(), w)
		}

		return
	}

	announcementResp := dto.AnnouncementForAdminsResponse{
		AnnouncementResponse: dto.AnnouncementResponse{
			ID:          announcement.ID,
			Title:       announcement.Title,
			Description: announcement.Description,
			CreatedAt:   announcement.CreatedAt,
		},
		CreatedBy: announcement.CreatedBy,
	}

	responses.JSON(
		r.Context(),
		w,
		http.StatusOK,
		announcementResp,
	)
}

// ! Categories

const opAdminCategoriesUpdate = "delivery.http.handlers.Admin.Categories.Update"

func (h Admin) CategoriesUpdate(w http.ResponseWriter, r *http.Request) {
	if ok := h.superAdmin(w, r, opAdminCategoriesUpdate); !ok {
		return
	}

	jsonBytes, err := io.ReadAll(r.Body)
	if err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(r.Context()),
			opAdminCategoriesUpdate,
		)

		responses.Error(
			r.Context(),
			w,
			http.StatusBadRequest,
			http.StatusText(http.StatusBadRequest),
		)
		return
	}

	if err := h.CategoriesService.Update(r.Context(), jsonBytes); err != nil {
		switch {
		case errors.Is(err, apperror.ErrBadJSON):
			responses.Error(
				r.Context(),
				w,
				http.StatusBadRequest,
				http.StatusText(http.StatusBadRequest),
			)
		case errors.Is(err, apperror.ErrUnprocessableJSON):
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

	w.WriteHeader(http.StatusNoContent)
}

// ! Admin management

const opAdminSetAdmin = "delivery.http.handlers.Admin.SetAdmin"

func (h Admin) SetAdmin(w http.ResponseWriter, r *http.Request) {
	if ok := h.superAdmin(w, r, opAdminSetAdmin); !ok {
		return
	}

	var emailDTO dto.EmailResponseAndRequest

	if err := json.NewDecoder(r.Body).Decode(&emailDTO); err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(r.Context()),
			opAdminSetAdmin,
		)

		responses.Error(
			r.Context(),
			w,
			http.StatusBadRequest,
			http.StatusText(http.StatusBadRequest),
		)

		return
	}

	if err := emailDTO.Validate(); err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(r.Context()),
			opAdminSetAdmin,
		)

		responses.Error(
			r.Context(),
			w,
			http.StatusUnprocessableEntity,
			err.Error(),
		)

		return
	}

	if err := h.AdminService.SetAdminByEmail(r.Context(), *emailDTO.Email); err != nil {
		switch {
		case errors.Is(err, apperror.ErrUserAlreadyAdmin):
			responses.Error(
				r.Context(),
				w,
				http.StatusConflict,
				err.Error(),
			)
		case errors.Is(err, apperror.ErrUserNotExists):
			responses.Error(
				r.Context(),
				w,
				http.StatusNotFound,
				err.Error(),
			)
		default:
			responses.InternalServerError(r.Context(), w)
		}

		return
	}

	w.WriteHeader(http.StatusNoContent)
}

const opAdminDeleteAdmin = "delivery.http.handlers.Admin.DeleteAdmin"

func (h Admin) DeleteAdmin(w http.ResponseWriter, r *http.Request) {
	if ok := h.superAdmin(w, r, opAdminDeleteAdmin); !ok {
		return
	}

	var emailDTO dto.EmailResponseAndRequest

	if err := json.NewDecoder(r.Body).Decode(&emailDTO); err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(r.Context()),
			opAdminDeleteAdmin,
		)

		responses.Error(
			r.Context(),
			w,
			http.StatusBadRequest,
			http.StatusText(http.StatusBadRequest),
		)

		return
	}

	if err := emailDTO.Validate(); err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(r.Context()),
			opAdminDeleteAdmin,
		)

		responses.Error(
			r.Context(),
			w,
			http.StatusUnprocessableEntity,
			err.Error(),
		)

		return
	}

	if err := h.AdminService.DeleteAdminByEmail(r.Context(), *emailDTO.Email); err != nil {
		switch {
		case errors.Is(err, apperror.ErrUserSuperAdmin):
			responses.Error(
				r.Context(),
				w,
				http.StatusForbidden,
				http.StatusText(http.StatusForbidden),
			)
		case errors.Is(err, apperror.ErrUserNotAdmin):
			responses.Error(
				r.Context(),
				w,
				http.StatusConflict,
				err.Error(),
			)
		case errors.Is(err, apperror.ErrUserNotExists):
			responses.Error(
				r.Context(),
				w,
				http.StatusNotFound,
				err.Error(),
			)
		default:
			responses.InternalServerError(r.Context(), w)
		}

		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h Admin) GetPrivileges(w http.ResponseWriter, r *http.Request) {
	uid := common.GetUID(r.Context())

	adminData, err := h.AdminService.GetAdminDataByUID(r.Context(), uid)
	if err != nil {
		responses.InternalServerError(r.Context(), w)

		return
	}

	resp := make(map[string]any)
	resp["privileges"] = dto.AdminPrivilegesResponse{
		IsAdmin:      true,
		IsSuperAdmin: adminData.IsSuperAdmin,
	}

	responses.JSON(
		r.Context(),
		w,
		http.StatusOK,
		resp,
	)
}

func (h Admin) adminEntityToAdminDTO(admin entity.Admin) dto.AdminResponse {
	adminDTO := dto.AdminResponse{
		Email:        admin.Email,
		IsSuperAdmin: admin.IsSuperAdmin,
	}

	return adminDTO
}

func (h Admin) sliceAdminsEntitiesToSliceAdminsDTO(page []entity.Admin) []dto.AdminResponse {
	sliceAdminsDTO := make([]dto.AdminResponse, 0, len(page))
	for _, admin := range page {
		adminDTO := h.adminEntityToAdminDTO(admin)

		sliceAdminsDTO = append(sliceAdminsDTO, adminDTO)
	}

	return sliceAdminsDTO
}

const opAdminGetListAdmins = "delivery.http.handlers.Admin.GetListAdmins"

func (h Admin) GetListAdmins(w http.ResponseWriter, r *http.Request) {
	if ok := h.superAdmin(w, r, opAdminGetListAdmins); !ok {
		return
	}

	admins, err := h.AdminService.GetListAdmins(r.Context())
	if err != nil {
		responses.InternalServerError(r.Context(), w)

		return
	}

	adminsResp := h.sliceAdminsEntitiesToSliceAdminsDTO(admins)

	resp := make(map[string]any)
	resp["admins"] = adminsResp

	responses.JSON(
		r.Context(),
		w,
		http.StatusOK,
		resp,
	)
}
