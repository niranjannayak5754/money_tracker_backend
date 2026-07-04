package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"log/slog"

	"github.com/go-chi/chi/v5"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/common"

	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/investment"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/shared"
	"github.com/niranjannayak5754/money_tracker_backend/internal/http/requestctx"
	"github.com/niranjannayak5754/money_tracker_backend/internal/http/response"
)

type InvestmentHandler struct {
	svc    investment.Service
	logger *slog.Logger
}

func NewInvestmentHandler(svc investment.Service, logger *slog.Logger) *InvestmentHandler {
	return &InvestmentHandler{svc: svc, logger: logger.With("handler", "investment")}
}

func (h *InvestmentHandler) Routes() http.Handler {
	r := chi.NewRouter()
	r.Get("/types", h.types)
	r.Get("/", h.list)
	r.Post("/", h.create)
	r.Put("/{id}", h.update)
	r.Post("/{id}/close", h.close)
	r.Delete("/{id}", h.delete)
	return r
}

func (h *InvestmentHandler) create(w http.ResponseWriter, r *http.Request) {
	uid, ok := mustUID(w, r)
	if !ok {
		return
	}

	var in struct {
		Type       string              `json:"type"`
		Instrument string              `json:"instrument"`
		Amount     float64             `json:"amount"`
		Date       shared.FlexibleTime `json:"date"`
		Notes      string              `json:"notes"`
	}

	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.BadReq(w, "invalid json payload")
		return
	}

	if in.Amount <= 0 {
		response.BadReq(w, "amount must be > 0")
		return
	}

	rec, err := h.svc.Create(r.Context(), uid, investment.CreateInput{
		Type:       in.Type,
		Instrument: in.Instrument,
		Amount:     in.Amount,
		Date:       shared.ChooseDate(in.Date.Time),
		Notes:      in.Notes,
	})
	if err != nil {
		h.logger.Error("investment.create failed", "request_id", requestctx.RequestID(r.Context()), "uid", uid, "err", err)
		response.WriteError(w, r, err)
		return
	}

	response.JSON(w, http.StatusCreated, rec)
}

func (h *InvestmentHandler) list(w http.ResponseWriter, r *http.Request) {
	uid, ok := mustUID(w, r)
	if !ok {
		return
	}

	month := r.URL.Query().Get("month")
	items, err := h.svc.List(r.Context(), uid, month)
	if err != nil {
		h.logger.Error("investment.list failed", "request_id", requestctx.RequestID(r.Context()), "uid", uid, "err", err)
		response.WriteError(w, r, err)
		return
	}

	if items == nil {
		items = []investment.Model{}
	}
	response.JSON(w, http.StatusOK, items)
}

func (h *InvestmentHandler) types(w http.ResponseWriter, r *http.Request) {
	vals, err := h.svc.ListTypes(r.Context())
	if err != nil {
		h.logger.Error("investment.types failed", "request_id", requestctx.RequestID(r.Context()), "err", err)
		response.WriteError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, vals)
}

func (h *InvestmentHandler) update(w http.ResponseWriter, r *http.Request) {
	uid, ok := mustUID(w, r)
	if !ok {
		return
	}

	invID := common.InvestmentID(chi.URLParam(r, "id"))

	var in struct {
		Type       *string    `json:"type"`
		Instrument *string    `json:"instrument"`
		Amount     *float64   `json:"amount"`
		Date       *time.Time `json:"date"`
		Notes      *string    `json:"notes"`
	}

	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.BadReq(w, "invalid json payload")
		return
	}

	err := h.svc.Update(r.Context(), uid, invID, investment.UpdateInput{
		Type:       in.Type,
		Instrument: in.Instrument,
		Amount:     in.Amount,
		Date:       in.Date,
		Notes:      in.Notes,
	})
	if err != nil {
		h.logger.Error("investment.update failed", "request_id", requestctx.RequestID(r.Context()), "uid", uid, "investment_id", invID, "err", err)
		response.WriteError(w, r, err)
		return
	}

	response.OK(w)
}

func (h *InvestmentHandler) close(w http.ResponseWriter, r *http.Request) {
	uid, ok := mustUID(w, r)
	if !ok {
		return
	}

	invID := common.InvestmentID(chi.URLParam(r, "id"))

	var in struct {
		WithdrawnAmount   float64             `json:"withdrawn_amount"`
		CostBasisConsumed float64             `json:"cost_basis_consumed"`
		Date              shared.FlexibleTime `json:"date"`
		Notes             string              `json:"notes"`
	}

	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.BadReq(w, "invalid json payload")
		return
	}

	rec, err := h.svc.Close(r.Context(), uid, invID, investment.CloseInput{
		WithdrawnAmount:   in.WithdrawnAmount,
		CostBasisConsumed: in.CostBasisConsumed,
		Date:              in.Date.Time,
		Notes:             in.Notes,
	})
	if err != nil {
		h.logger.Error("investment.close failed", "request_id", requestctx.RequestID(r.Context()), "uid", uid, "investment_id", invID, "err", err)
		response.WriteError(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, rec)
}

func (h *InvestmentHandler) delete(w http.ResponseWriter, r *http.Request) {
	uid, ok := mustUID(w, r)
	if !ok {
		return
	}

	invID := common.InvestmentID(chi.URLParam(r, "id"))
	err := h.svc.Delete(r.Context(), uid, invID)
	if err != nil {
		h.logger.Error("investment.delete failed", "request_id", requestctx.RequestID(r.Context()), "uid", uid, "investment_id", invID, "err", err)
		response.WriteError(w, r, err)
		return
	}
	response.OK(w)
}
