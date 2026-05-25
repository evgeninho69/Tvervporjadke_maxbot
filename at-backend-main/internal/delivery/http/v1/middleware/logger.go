package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

var opLogger = slog.String("op", "middleware.Logger")

func Logger(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slog.InfoContext(
			r.Context(),
			"request to server",
			opLogger,
			slog.Group(
				"request",
				"method", r.Method,
				"path", r.URL.Path,
			),
		)
		
		start := time.Now()

		handler.ServeHTTP(w, r)

		slog.InfoContext(
			r.Context(),
			"request completed",
			opLogger,
			slog.String("request processing time", time.Since(start).String()),
		)
	})
}
