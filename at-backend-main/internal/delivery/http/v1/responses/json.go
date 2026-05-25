package responses

import (
	"activeTver/internal/pkg/common"
	"activeTver/internal/pkg/logs"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
)

const opJSON = "responses.JSON"

// JSON формирует ответ в формате JSON и записывает его в http.ResponseWriter.
func JSON(ctx context.Context, w http.ResponseWriter, statusCode int, data any) {
	correlationID := common.GetCorrelationID(ctx)

	bytes, err := json.Marshal(data)
	if err != nil {
		logs.Error(
			err.Error(),
			correlationID,
			opJSON,
		)

		InternalServerError(ctx, w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if _, err := w.Write(bytes); err != nil {
		logs.Error(
			err.Error(),
			correlationID,
			opJSON,
		)

		return
	}

	logs.Info(
		"written answer",
		correlationID,
		opJSON,
		slog.Int("statusCode", statusCode),
	)
}
