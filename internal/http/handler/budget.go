package handler

import (
	"encoding/json"
	"net/http"

	"log/slog"

	"github.com/go-chi/chi/v5"

	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/budget"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/common"
	"github.com/niranjannayak5754/money_tracker_backend/internal/http/requestctx"
	"github.com/niranjannayak5754/money_tracker_backend/internal/http/response"
)

type BudgetHandler struct {
	svc    budget.Service
	logger *slog.Logger
}

func NewBudgetHandler(svc budget.Service, logger *slog.Logger) *BudgetHandler {
	return &BudgetHandler{svc: svc, logger: logger.With("handler", "budget")}
}

func (h *BudgetHandler) Routes() http.Handler {
	r := chi.NewRouter()
	r.Get("/", h.list)
	r.Post("/", h.set)
	r.Delete("/{id}", h.delete)
	return r
}

func (h *BudgetHandler) set(w http.ResponseWriter, r *http.Request) {
	uid, ok := mustUID(w, r)
	if !ok {
		return
	}

	var in struct {
		CategoryID    *string `json:"category_id"`
		Amount        float64 `json:"amount"`
		EffectiveFrom string  `json:"effective_from"`
	}

	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.BadReq(w, "invalid json payload")
		return
	}

	var categoryID *common.CategoryID
	if in.CategoryID != nil && *in.CategoryID != "" {
		cid := common.CategoryID(*in.CategoryID)
		categoryID = &cid
	}

	out, err := h.svc.Set(r.Context(), uid, budget.SetInput{
		CategoryID:    categoryID,
		Amount:        in.Amount,
		EffectiveFrom: in.EffectiveFrom,
	})
	if err != nil {
		h.logger.Error("budget.set failed", "request_id", requestctx.RequestID(r.Context()), "uid", uid, "err", err)
		response.WriteError(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, out)
}

func (h *BudgetHandler) list(w http.ResponseWriter, r *http.Request) {
	uid, ok := mustUID(w, r)
	if !ok {
		return
	}

	items, err := h.svc.List(r.Context(), uid)
	if err != nil {
		h.logger.Error("budget.list failed", "request_id", requestctx.RequestID(r.Context()), "uid", uid, "err", err)
		response.WriteError(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, items)
}

func (h *BudgetHandler) delete(w http.ResponseWriter, r *http.Request) {
	uid, ok := mustUID(w, r)
	if !ok {
		return
	}

	id := common.BudgetID(chi.URLParam(r, "id"))
	if err := h.svc.Delete(r.Context(), uid, id); err != nil {
		h.logger.Error("budget.delete failed", "request_id", requestctx.RequestID(r.Context()), "uid", uid, "err", err)
		response.WriteError(w, r, err)
		return
	}

	response.OK(w)
}
