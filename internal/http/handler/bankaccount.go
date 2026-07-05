package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"log/slog"

	"github.com/go-chi/chi/v5"

	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/bankaccount"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/common"
	"github.com/niranjannayak5754/money_tracker_backend/internal/http/requestctx"
	"github.com/niranjannayak5754/money_tracker_backend/internal/http/response"
)

type BankAccountHandler struct {
	svc    bankaccount.Service
	logger *slog.Logger
}

func NewBankAccountHandler(svc bankaccount.Service, logger *slog.Logger) *BankAccountHandler {
	return &BankAccountHandler{svc: svc, logger: logger.With("handler", "bankaccount")}
}

func (h *BankAccountHandler) Routes() http.Handler {
	r := chi.NewRouter()
	r.Get("/", h.list)
	r.Post("/", h.create)
	r.Delete("/{id}", h.delete)
	r.Post("/{id}/adjust", h.adjust)
	r.Get("/{id}/ledger", h.ledger)
	r.Get("/{id}/balance-history", h.balanceHistory)
	return r
}

func (h *BankAccountHandler) create(w http.ResponseWriter, r *http.Request) {
	uid, ok := mustUID(w, r)
	if !ok {
		return
	}

	var in struct {
		Name         string   `json:"name"`
		Balance      float64  `json:"balance"`
		InterestRate *float64 `json:"interest_rate"`
	}

	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.BadReq(w, "invalid json payload")
		return
	}

	out, err := h.svc.Create(r.Context(), uid, bankaccount.CreateInput{
		Name:         in.Name,
		Balance:      in.Balance,
		InterestRate: in.InterestRate,
	})
	if err != nil {
		h.logger.Error("bankaccount.create failed", "request_id", requestctx.RequestID(r.Context()), "uid", uid, "err", err)
		response.WriteError(w, r, err)
		return
	}

	response.JSON(w, http.StatusCreated, out)
}

func (h *BankAccountHandler) list(w http.ResponseWriter, r *http.Request) {
	uid, ok := mustUID(w, r)
	if !ok {
		return
	}

	items, err := h.svc.List(r.Context(), uid)
	if err != nil {
		h.logger.Error("bankaccount.list failed", "request_id", requestctx.RequestID(r.Context()), "uid", uid, "err", err)
		response.WriteError(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, items)
}

func (h *BankAccountHandler) ledger(w http.ResponseWriter, r *http.Request) {
	uid, ok := mustUID(w, r)
	if !ok {
		return
	}

	id := common.BankAccountID(chi.URLParam(r, "id"))
	month := r.URL.Query().Get("month")

	entries, err := h.svc.ListLedger(r.Context(), uid, id, month)
	if err != nil {
		h.logger.Error("bankaccount.ledger failed", "request_id", requestctx.RequestID(r.Context()), "uid", uid, "err", err)
		response.WriteError(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, entries)
}

func (h *BankAccountHandler) balanceHistory(w http.ResponseWriter, r *http.Request) {
	uid, ok := mustUID(w, r)
	if !ok {
		return
	}

	id := common.BankAccountID(chi.URLParam(r, "id"))
	months := 3
	if qs := r.URL.Query().Get("months"); qs != "" {
		if v, err := strconv.Atoi(qs); err == nil {
			months = v
		}
	}

	out, err := h.svc.BalanceHistory(r.Context(), uid, id, months)
	if err != nil {
		h.logger.Error("bankaccount.balance_history failed", "request_id", requestctx.RequestID(r.Context()), "uid", uid, "err", err)
		response.WriteError(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, out)
}

func (h *BankAccountHandler) adjust(w http.ResponseWriter, r *http.Request) {
	uid, ok := mustUID(w, r)
	if !ok {
		return
	}

	id := common.BankAccountID(chi.URLParam(r, "id"))

	var in struct {
		Direction string  `json:"direction"`
		Amount    float64 `json:"amount"`
		Note      string  `json:"note"`
	}

	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.BadReq(w, "invalid json payload")
		return
	}

	out, err := h.svc.Adjust(r.Context(), uid, id, bankaccount.AdjustInput{
		Direction: bankaccount.AdjustDirection(in.Direction),
		Amount:    in.Amount,
		Note:      in.Note,
	})
	if err != nil {
		h.logger.Error("bankaccount.adjust failed", "request_id", requestctx.RequestID(r.Context()), "uid", uid, "err", err)
		response.WriteError(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, out)
}

func (h *BankAccountHandler) delete(w http.ResponseWriter, r *http.Request) {
	uid, ok := mustUID(w, r)
	if !ok {
		return
	}

	id := common.BankAccountID(chi.URLParam(r, "id"))
	if err := h.svc.Delete(r.Context(), uid, id); err != nil {
		h.logger.Error("bankaccount.delete failed", "request_id", requestctx.RequestID(r.Context()), "uid", uid, "err", err)
		response.WriteError(w, r, err)
		return
	}

	response.OK(w)
}
