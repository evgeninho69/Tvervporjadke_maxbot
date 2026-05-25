package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"activeTver/internal/delivery/http/v1/responses"
	"activeTver/internal/pkg/common"
	"activeTver/internal/pkg/logs"
	"activeTver/internal/pkg/tokens"
)

const opAuth = "middleware.Auth"

func Auth(tm *tokens.TokensManager) func(handler http.Handler) http.Handler {
	return func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				logs.Warn(
					"authorization header empty",
					common.GetCorrelationID(r.Context()),
					opAuth,
				)

				responses.Error(
					r.Context(),
					w,
					http.StatusBadRequest,
					"authorization header empty",
				)
				return
			}

			authHeaderParts := strings.Fields(authHeader)
			if len(authHeaderParts) != 2 || strings.ToLower(authHeaderParts[0]) != "bearer" {
				logs.Warn(
					"invalid value in authorization header",
					common.GetCorrelationID(r.Context()),
					opAuth,
				)

				responses.Error(
					r.Context(),
					w,
					http.StatusBadRequest,
					"400 authorization header must be \"Bearer {token}\"",
				)
				return
			}

			claims, err := tm.ParseJWT(authHeaderParts[1])
			if err != nil {
				logs.Error(
					err.Error(),
					common.GetCorrelationID(r.Context()),
					opAuth,
				)

				responses.Error(
					r.Context(),
					w,
					http.StatusUnauthorized,
					http.StatusText(http.StatusUnauthorized),
				)
				return
			}

			common.SetValueIntoContext(r.Context(), common.UID_KEY, claims.UID)

			logs.Info(
				fmt.Sprintf("authorized uid %v", claims.UID),
				common.GetCorrelationID(r.Context()),
				opAuth,
			)

			handler.ServeHTTP(w, r)
		})
	}
}
