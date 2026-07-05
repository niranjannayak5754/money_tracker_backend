package handler

import (
	"net/http"
	"strconv"

	"log/slog"

	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/summary"
	"github.com/niranjannayak5754/money_tracker_backend/internal/http/requestctx"
	"github.com/niranjannayak5754/money_tracker_backend/internal/http/response"
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
			"request_id", requestctx.RequestID(r.Context()),
			"uid", uid,
			"month", month,
			"err", err,
		)
		response.WriteError(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, out)
}

// Compare returns monthly income, expense and savings for the last N months.
func (h *SummaryHandler) Compare(w http.ResponseWriter, r *http.Request) {
	uid, ok := mustUID(w, r)
	if !ok {
		return
	}

	qs := r.URL.Query().Get("range")
	months := 3
	// parse range query param
	if qs != "" {
		if v, err := strconv.Atoi(qs); err == nil {
			months = v
		}
	}

	out, err := h.svc.Compare(r.Context(), uid, months)
	if err != nil {
		h.logger.Error(
			"summary.compare failed",
			"request_id", requestctx.RequestID(r.Context()),
			"uid", uid,
			"range", months,
			"err", err,
		)
		response.WriteError(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, out)
}
