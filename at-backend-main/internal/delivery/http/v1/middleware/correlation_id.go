package middleware

import (
	"activeTver/internal/pkg/common"
	"net/http"

	"github.com/google/uuid"
)

func SetupCorrelationID(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		common.SetValueIntoContext(r.Context(), common.CORRELATION_ID_KEY, uuid.New().String())

		handler.ServeHTTP(w, r)
	})
}
