package logging

import (
	"context"
	"log/slog"

	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/niranjannayak5754/money_tracker_backend/internal/http/requestctx"
)

func WithRequest(ctx context.Context, log *slog.Logger) *slog.Logger {
	attrs := []any{}

	if rid := chimw.GetReqID(ctx); rid != "" {
		attrs = append(attrs, "request_id", rid)
	}
	if uid := requestctx.UID(ctx); uid != "" {
		attrs = append(attrs, "user_id", uid)
	}

	if len(attrs) > 0 {
		return log.With(attrs...)
	}
	return log
}
