package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"activeTver/internal/delivery/http/v1/dto"
	"activeTver/internal/delivery/http/v1/responses"
	"activeTver/internal/domain/entity"
	"activeTver/internal/pkg/apperror"
	"activeTver/internal/pkg/common"
	"activeTver/internal/pkg/logs"
	ucports "activeTver/internal/usecase/ports"

	"github.com/google/uuid"
)

type Auth struct {
	Service ucports.AuthUseCase
}

func (h Auth) Setup(authMiddleware func(http.Handler) http.Handler) *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /login", h.RequestLogin)
	mux.HandleFunc("POST /confirm-login", h.ConfirmLogin)
	mux.HandleFunc("POST /refresh", h.RefreshTokenPair)

	mux.Handle("POST /logout", authMiddleware(http.HandlerFunc(h.Logout)))
	mux.Handle("POST /logout-all-devices", authMiddleware(http.HandlerFunc(h.LogoutAllUserDevices)))

	return mux
}

const opAuthRequestLogin = "delivery.http.handlers.Auth.RequestLogin"

func (h Auth) RequestLogin(w http.ResponseWriter, r *http.Request) {
	var emailDTO dto.EmailResponseAndRequest

	if err := json.NewDecoder(r.Body).Decode(&emailDTO); err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(r.Context()),
			opAuthRequestLogin,
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
			opAuthRequestLogin,
		)

		responses.Error(
			r.Context(),
			w,
			http.StatusUnprocessableEntity,
			err.Error(),
		)
		return
	}

	if err := h.Service.RequestLogin(r.Context(), *emailDTO.Email); err != nil {
		responses.InternalServerError(r.Context(), w)

		return
	}

	w.WriteHeader(http.StatusNoContent)
}

const opAuthConfirmLogin = "delivery.http.handlers.Auth.ConfirmLogin"

func (h Auth) ConfirmLogin(w http.ResponseWriter, r *http.Request) {
	var confirmLoginRequest dto.ConfirmLoginRequest

	if err := json.NewDecoder(r.Body).Decode(&confirmLoginRequest); err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(r.Context()),
			opAuthConfirmLogin,
		)

		responses.Error(
			r.Context(),
			w,
			http.StatusBadRequest,
			http.StatusText(http.StatusBadRequest),
		)
		return
	}

	if err := confirmLoginRequest.Validate(); err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(r.Context()),
			opAuthConfirmLogin,
		)

		responses.Error(
			r.Context(),
			w,
			http.StatusUnprocessableEntity,
			err.Error(),
		)

		return
	}

	jwt, rt, err := h.Service.Login(r.Context(), *confirmLoginRequest.Email, *confirmLoginRequest.Code)
	if err != nil {
		switch {
		case errors.Is(err, apperror.ErrUserNotRequestLogin):
			responses.Error(
				r.Context(),
				w,
				http.StatusBadRequest,
				err.Error(),
			)
		case errors.Is(err, apperror.ErrInvalidMailCode):
			responses.Error(
				r.Context(),
				w,
				http.StatusUnauthorized,
				err.Error(),
			)
		case errors.Is(err, apperror.ErrLoginAttemptsLimitExceeded):
			responses.Error(
				r.Context(),
				w,
				http.StatusTooManyRequests,
				err.Error(),
			)
		case errors.Is(err, apperror.ErrUserNotExists):
			// для сокрытия существования пользователя возвращаем 401
			responses.Error(
				r.Context(),
				w,
				http.StatusUnauthorized,
				http.StatusText(http.StatusUnauthorized),
			)
		default:
			responses.InternalServerError(r.Context(), w)
		}
		return
	}

	tokenPair := dto.TokenPairResponse{
		RefreshToken: fmt.Sprintf("%v.%v", rt.ID, rt.Token),
		AccessToken:  jwt,
	}

	responses.JSON(
		r.Context(),
		w,
		http.StatusOK,
		tokenPair,
	)
}

var dummyRT entity.RefreshToken

func (h Auth) getRTfromBody(ctx context.Context, body io.Reader) (entity.RefreshToken, error) {
	var refreshTokenDTO dto.RefreshTokenRequest

	if err := json.NewDecoder(body).Decode(&refreshTokenDTO); err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(ctx),
			opAuthRefreshTokenPair,
		)

		return dummyRT, err
	}

	if err := refreshTokenDTO.Validate(); err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(ctx),
			opAuthRefreshTokenPair,
		)

		return dummyRT, errors.Join(err, apperror.ErrInvalidRefreshToken)
	}

	idString, token, ok := strings.Cut(*refreshTokenDTO.Token, ".")
	if !ok {
		return dummyRT, apperror.ErrInvalidRefreshToken
	}

	id, err := uuid.Parse(idString)
	if err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(ctx),
			opAuthRefreshTokenPair,
		)

		return dummyRT, apperror.ErrInvalidRefreshToken
	}

	rt := entity.RefreshToken{
		ID:    id,
		Token: token,
	}

	return rt, nil
}

const opAuthRefreshTokenPair = "delivery.http.handlers.Auth.RefreshTokenPair"

func (h Auth) RefreshTokenPair(w http.ResponseWriter, r *http.Request) {
	oldRT, err := h.getRTfromBody(r.Context(), r.Body)
	if err != nil {
		switch {
		case errors.Is(err, apperror.ErrInvalidRefreshToken):
			responses.Error(
				r.Context(),
				w,
				http.StatusUnprocessableEntity,
				err.Error(),
			)
		default:
			responses.Error(
				r.Context(),
				w,
				http.StatusBadRequest,
				http.StatusText(http.StatusBadRequest),
			)
		}

		return
	}

	newJWT, newRT, err := h.Service.RefreshPairTokens(r.Context(), oldRT)
	if err != nil {
		switch {
		case errors.Is(err, apperror.ErrTokenNotExists), errors.Is(err, apperror.ErrInvalidRefreshToken), errors.Is(err, apperror.ErrRefreshTokenRevoked):
			responses.Error(
				r.Context(),
				w,
				http.StatusUnauthorized,
				http.StatusText(http.StatusUnauthorized),
			)
		case errors.Is(err, apperror.ErrRefreshTokenExpired):
			responses.Error(
				r.Context(),
				w,
				http.StatusUnauthorized,
				err.Error(),
			)
		default:
			responses.InternalServerError(r.Context(), w)
		}

		return
	}

	tokenPair := dto.TokenPairResponse{
		RefreshToken: fmt.Sprintf("%v.%v", newRT.ID, newRT.Token),
		AccessToken:  newJWT,
	}

	responses.JSON(
		r.Context(),
		w,
		http.StatusOK,
		tokenPair,
	)
}

const opAuthLogout = "delivery.http.handlers.Auth.Logout"

func (h Auth) Logout(w http.ResponseWriter, r *http.Request) {
	rt, err := h.getRTfromBody(r.Context(), r.Body)
	if err != nil {
		switch {
		case errors.Is(err, apperror.ErrInvalidRefreshToken):
			responses.Error(
				r.Context(),
				w,
				http.StatusUnprocessableEntity,
				http.StatusText(http.StatusUnprocessableEntity),
			)
		default:
			logs.Error(
				err.Error(),
				common.GetCorrelationID(r.Context()),
				opAuthLogout,
			)

			responses.Error(
				r.Context(),
				w,
				http.StatusBadRequest,
				http.StatusText(http.StatusBadRequest),
			)
		}

		return
	}

	// TODO: Добавить проверку что токен принадлежит авторизованному пользователю

	if err := h.Service.Logout(r.Context(), rt); err != nil {
		switch {
		case errors.Is(err, apperror.ErrInvalidRefreshToken), errors.Is(err, apperror.ErrTokenNotExists):
			responses.Error(
				r.Context(),
				w,
				http.StatusUnauthorized,
				http.StatusText(http.StatusUnauthorized),
			)
		default:
			responses.InternalServerError(r.Context(), w)
		}

		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h Auth) LogoutAllUserDevices(w http.ResponseWriter, r *http.Request) {
	if err := h.Service.LogoutAllUserDevices(r.Context(), common.GetUID(r.Context())); err != nil {
		responses.InternalServerError(r.Context(), w)

		return
	}

	w.WriteHeader(http.StatusNoContent)
}
