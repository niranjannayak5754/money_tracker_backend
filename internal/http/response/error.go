package response

import (
	"encoding/json"
	"net/http"

	"github.com/niranjannayak5754/money_tracker_backend/internal/apperr"
	"github.com/niranjannayak5754/money_tracker_backend/internal/http/requestctx"
)

func WriteError(w http.ResponseWriter, r *http.Request, err error) {
	status := http.StatusInternalServerError
	msg := "internal server error"

	if ae, ok := apperr.AsAppError(err); ok {
		status = ae.Status
		msg = ae.Message
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(map[string]string{
		"message":    msg,
		"request_id": requestctx.RequestID(r.Context()),
	})
}
