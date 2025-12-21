package handler

import (
	"net/http"

	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/common"
	"github.com/niranjannayak5754/money_tracker_backend/internal/http/requestctx"
	"github.com/niranjannayak5754/money_tracker_backend/internal/http/response"
)

func mustUID(w http.ResponseWriter, r *http.Request) (common.UserID, bool) {
	uid := requestctx.UID(r.Context())
	if uid == "" {
		response.Unauthorized(w, "missing uid")
		return "", false
	}

	return common.UserID(uid), true
}
