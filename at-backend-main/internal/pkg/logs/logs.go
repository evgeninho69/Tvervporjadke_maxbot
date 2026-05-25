package logs

import (
	"activeTver/internal/pkg/common"
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"os"
	"runtime"
	"time"
)

var (
	ErrLoggerSetup = errors.New("logger setup error")

	DEBUG = false

	writerForLogs io.Writer
	flushFunc     func() error
)

func init() {
	DEBUG = os.Getenv("ENV") == "DEBUG"
}

type PrettyWriter struct {
	writer io.Writer
}

func (pw PrettyWriter) Write(p []byte) (n int, err error) {
	var buf bytes.Buffer
	if err := json.Indent(&buf, p, "", "  "); err != nil {
		return pw.writer.Write(p)
	}
	return pw.writer.Write(buf.Bytes())
}

type CustomHandler struct {
	slog.Handler
}

func (h CustomHandler) Handle(ctx context.Context, r slog.Record) error {
	if common.CTXValuesExists(ctx) {
		correlationID := common.GetCorrelationID(ctx)
		uid := common.GetUID(ctx)

		if correlationID != "" {
			r.AddAttrs(slog.String("correlation_id", correlationID))
		}

		if uid != 0 {
			r.AddAttrs(slog.Uint64("uid", uid))
		}
	}

	if DEBUG {
		var pcs [1]uintptr
		runtime.Callers(5, pcs[:])

		r.AddAttrs(
			getSource(pcs[0]),
		)
	}

	return h.Handler.Handle(ctx, r)
}

func getSource(pc uintptr) slog.Attr {
	fs := runtime.CallersFrames([]uintptr{pc})
	f, _ := fs.Next()

	return slog.Group(
		"source",
		slog.String("func", f.Function),
		slog.Int("line", f.Line),
	)
}

func Setup() {
	if os.Getenv("ENV") == "DEBUG" {
		writerForLogs = PrettyWriter{writer: os.Stdout}
	} else {
		writer := bufio.NewWriter(os.Stdout)
		flushFunc = writer.Flush

		writerForLogs = writer

		go func() {
			ticker := time.NewTicker(1 * time.Minute)
			defer ticker.Stop()

			for range ticker.C {
				if err := writer.Flush(); err != nil {
					Error("failed to flush logs", "logs", "logs.Setup")
				}
			}
		}()
	}

	handler := CustomHandler{
		Handler: slog.NewJSONHandler(writerForLogs, nil),
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)
}

func Flush() error {
	if flushFunc != nil {
		return flushFunc()
	}
	return nil
}

func Error(msg, correlationID, op string, other ...any) {
	slog.Error(
		msg,
		"correlation_id", correlationID,
		"op", op,
		slog.Group("data", other...),
	)
}

func Info(msg, correlationID, op string, other ...any) {
	slog.Info(
		msg,
		"correlation_id", correlationID,
		"op", op,
		slog.Group("data", other...),
	)
}

func Warn(msg, correlationID, op string, other ...any) {
	slog.Warn(
		msg,
		"correlation_id", correlationID,
		"op", op,
		slog.Group("data", other...),
	)
}
