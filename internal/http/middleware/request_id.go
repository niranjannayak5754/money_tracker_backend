package middleware

import (
	"net/http"

	"github.com/niranjannayak5754/money_tracker_backend/internal/http/requestctx"
)

func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := r.Header.Get("X-Request-Id")
		if reqID == "" {
			reqID = requestctx.NewRequestID()
		}

		ctx := requestctx.WithRequestID(r.Context(), reqID)
		w.Header().Set("X-Request-Id", reqID)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
