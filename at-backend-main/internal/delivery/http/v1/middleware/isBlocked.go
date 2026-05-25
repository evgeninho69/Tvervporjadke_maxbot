package middleware

import (
	"activeTver/internal/delivery/http/v1/responses"
	"activeTver/internal/pkg/common"
	"activeTver/internal/pkg/logs"
	"context"
	"fmt"
	"net/http"
)

const opCheckUserBlocked = "middleware.CheckUserBlocked"

func CheckUserBlocked(handler http.Handler, isBlocked func(context.Context, uint64) (bool, error)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uid := common.GetUID(r.Context())

		blocked, err := isBlocked(r.Context(), uid)
		if err != nil {
			logs.Error(
				err.Error(),
				common.GetCorrelationID(r.Context()),
				opCheckUserBlocked,
			)

			responses.InternalServerError(r.Context(), w)
			return
		}

		if !blocked {
			handler.ServeHTTP(w, r)
			return
		}

		responses.Error(
			r.Context(),
			w,
			http.StatusForbidden,
			"user is blocked",
		)

		logs.Warn(
			fmt.Sprintf("user %d is blocked", uid),
			common.GetCorrelationID(r.Context()),
			opCheckUserBlocked,
		)
	})
}
