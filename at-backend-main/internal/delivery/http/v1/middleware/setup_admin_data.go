package middleware

import (
	"activeTver/internal/delivery/http/v1/responses"
	"activeTver/internal/domain/entity"
	"activeTver/internal/pkg/apperror"
	"activeTver/internal/pkg/common"
	"activeTver/internal/pkg/logs"
	"context"
	"errors"
	"net/http"
)

// ? Возможно стоит перейти на систему ролей

const opSetupAdminDataOnContext = "middleware.SetupAdminDataOnContext"

func SetupAdminDataOnContext(handler http.Handler, getAdminData func(context.Context, uint64) (entity.Admin, error)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		adminData, err := getAdminData(r.Context(), common.GetUID(r.Context()))
		if err != nil && !errors.Is(err, apperror.ErrUserNotAdmin) {
			logs.Error(
				err.Error(),
				common.GetCorrelationID(r.Context()),
				opSetupAdminDataOnContext,
			)

			responses.InternalServerError(r.Context(), w)
			
			return
		}

		if errors.Is(err, apperror.ErrUserNotAdmin) {
			logs.Warn(
				"access to admin panel denied",
				common.GetCorrelationID(r.Context()),
				opSetupAdminDataOnContext,
			)

			responses.Error(
				r.Context(),
				w,
				http.StatusForbidden,
				http.StatusText(http.StatusForbidden),
			)
			return
		}

		logs.Info(
			"gained access to the admin panel",
			common.GetCorrelationID(r.Context()),
			opSetupAdminDataOnContext,
		)

		common.SetValueIntoContext(r.Context(), common.ADMIN_DATA_KEY, adminData)
		handler.ServeHTTP(w, r)
	})
}
