package handler

import (
	"net/http"

	"log/slog"

	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/summary"
	"github.com/niranjannayak5754/money_tracker_backend/internal/http/response"
	"github.com/niranjannayak5754/money_tracker_backend/internal/http/requestctx"
)

type SummaryHandler struct {
	svc    summary.Service
	logger *slog.Logger
}

func NewSummaryHandler(
	svc summary.Service,
	logger *slog.Logger,
) *SummaryHandler {
	return &SummaryHandler{
		svc:    svc,
		logger: logger.With("handler", "summary"),
	}
}

func (h *SummaryHandler) Get(w http.ResponseWriter, r *http.Request) {
	uid, ok := mustUID(w, r)
	if !ok {
		return
	}

	month := r.URL.Query().Get("month")

	out, err := h.svc.Get(r.Context(), uid, month)
	if err != nil {
		h.logger.Error(
			"summary.get failed",
			"request_id", requestctx.UID(r.Context()),
			"user_id", uid.Hex(),
			"month", month,
			"err", err,
		)
		response.WriteError(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, out)
}
