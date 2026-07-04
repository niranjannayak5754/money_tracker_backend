package handler

import (
	"encoding/json"
	"net/http"

	"log/slog"

	"github.com/go-chi/chi/v5"

	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/common"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/debt"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/shared"
	"github.com/niranjannayak5754/money_tracker_backend/internal/http/requestctx"
	"github.com/niranjannayak5754/money_tracker_backend/internal/http/response"
)

type DebtHandler struct {
	svc    debt.Service
	logger *slog.Logger
}

func NewDebtHandler(svc debt.Service, logger *slog.Logger) *DebtHandler {
	return &DebtHandler{svc: svc, logger: logger.With("handler", "debt")}
}

func (h *DebtHandler) Routes() http.Handler {
	r := chi.NewRouter()
	r.Get("/", h.list)
	r.Post("/", h.create)
	r.Put("/{id}", h.update)
	r.Post("/{id}/payments", h.recordPayment)
	r.Delete("/{id}", h.delete)
	return r
}

func (h *DebtHandler) create(w http.ResponseWriter, r *http.Request) {
	uid, ok := mustUID(w, r)
	if !ok {
		return
	}

	var in struct {
		Name         string              `json:"name"`
		Principal    float64             `json:"principal"`
		InterestRate float64             `json:"interest_rate"`
		EMIAmount    float64             `json:"emi_amount"`
		TenureMonths int                 `json:"tenure_months"`
		StartDate    shared.FlexibleTime `json:"start_date"`
		Notes        string              `json:"notes"`
	}

	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.BadReq(w, "invalid json payload")
		return
	}

	out, err := h.svc.Create(r.Context(), uid, debt.CreateInput{
		Name:         in.Name,
		Principal:    in.Principal,
		InterestRate: in.InterestRate,
		EMIAmount:    in.EMIAmount,
		TenureMonths: in.TenureMonths,
		StartDate:    in.StartDate.Time,
		Notes:        in.Notes,
	})
	if err != nil {
		h.logger.Error("debt.create failed", "request_id", requestctx.RequestID(r.Context()), "uid", uid, "err", err)
		response.WriteError(w, r, err)
		return
	}

	response.JSON(w, http.StatusCreated, out)
}

func (h *DebtHandler) list(w http.ResponseWriter, r *http.Request) {
	uid, ok := mustUID(w, r)
	if !ok {
		return
	}

	items, err := h.svc.List(r.Context(), uid)
	if err != nil {
		h.logger.Error("debt.list failed", "request_id", requestctx.RequestID(r.Context()), "uid", uid, "err", err)
		response.WriteError(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, items)
}

func (h *DebtHandler) update(w http.ResponseWriter, r *http.Request) {
	uid, ok := mustUID(w, r)
	if !ok {
		return
	}

	id := common.DebtID(chi.URLParam(r, "id"))

	var in struct {
		Name         *string  `json:"name"`
		InterestRate *float64 `json:"interest_rate"`
		EMIAmount    *float64 `json:"emi_amount"`
		TenureMonths *int     `json:"tenure_months"`
		Notes        *string  `json:"notes"`
	}

	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.BadReq(w, "invalid json payload")
		return
	}

	err := h.svc.Update(r.Context(), uid, id, debt.UpdateInput{
		Name:         in.Name,
		InterestRate: in.InterestRate,
		EMIAmount:    in.EMIAmount,
		TenureMonths: in.TenureMonths,
		Notes:        in.Notes,
	})
	if err != nil {
		h.logger.Error("debt.update failed", "request_id", requestctx.RequestID(r.Context()), "uid", uid, "err", err)
		response.WriteError(w, r, err)
		return
	}

	response.OK(w)
}

func (h *DebtHandler) recordPayment(w http.ResponseWriter, r *http.Request) {
	uid, ok := mustUID(w, r)
	if !ok {
		return
	}

	id := common.DebtID(chi.URLParam(r, "id"))

	var in struct {
		PrincipalComponent float64             `json:"principal_component"`
		InterestComponent  float64             `json:"interest_component"`
		Date               shared.FlexibleTime `json:"date"`
		Notes              string              `json:"notes"`
	}

	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.BadReq(w, "invalid json payload")
		return
	}

	out, err := h.svc.RecordPayment(r.Context(), uid, id, debt.PaymentInput{
		PrincipalComponent: in.PrincipalComponent,
		InterestComponent:  in.InterestComponent,
		Date:               in.Date.Time,
		Notes:              in.Notes,
	})
	if err != nil {
		h.logger.Error("debt.record_payment failed", "request_id", requestctx.RequestID(r.Context()), "uid", uid, "debt_id", id, "err", err)
		response.WriteError(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, out)
}

func (h *DebtHandler) delete(w http.ResponseWriter, r *http.Request) {
	uid, ok := mustUID(w, r)
	if !ok {
		return
	}

	id := common.DebtID(chi.URLParam(r, "id"))
	if err := h.svc.Delete(r.Context(), uid, id); err != nil {
		h.logger.Error("debt.delete failed", "request_id", requestctx.RequestID(r.Context()), "uid", uid, "err", err)
		response.WriteError(w, r, err)
		return
	}

	response.OK(w)
}
