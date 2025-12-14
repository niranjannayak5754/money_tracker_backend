package handler

import (
	"net/http"

	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/summary"
	"github.com/niranjannayak5754/money_tracker_backend/internal/http/response"
)

type SummaryHandler struct {
	svc *summary.Service
}

func NewSummaryHandler(svc *summary.Service) *SummaryHandler {
	return &SummaryHandler{svc: svc}
}

func (h *SummaryHandler) Get(w http.ResponseWriter, r *http.Request) {
	uid, ok := mustUID(w, r)
	if !ok {
		return
	}

	month := r.URL.Query().Get("month")

	out, err := h.svc.Get(r.Context(), uid, month)
	if err != nil {
		response.ServerErr(w, err)
		return
	}

	response.JSON(w, http.StatusOK, out)
}
