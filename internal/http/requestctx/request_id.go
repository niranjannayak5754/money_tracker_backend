package requestctx

import (
	"context"

	"github.com/google/uuid"
)

type requestIDKey struct{}

var requestIDKeyInstance requestIDKey

func WithRequestID(ctx context.Context, reqID string) context.Context {
	return context.WithValue(ctx, requestIDKeyInstance, reqID)
}

func RequestID(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKeyInstance).(string)
	return id
}

// Helper to generate a new request id
func NewRequestID() string {
	return uuid.NewString()
}
