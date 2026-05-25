package middleware

import (
	"activeTver/internal/delivery/http/v1/responses"
	"net/http"
	"regexp"
)

var (
	fileRegex = regexp.MustCompile(`[a-zA-Z0-9\/-]+\.jpeg`)
)

func HintFiles(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !fileRegex.MatchString(r.URL.Path) {
			responses.Error(
				r.Context(),
				w,
				http.StatusForbidden,
				http.StatusText(http.StatusForbidden),
			)
			return
		}

		handler.ServeHTTP(w, r)
	})
}
