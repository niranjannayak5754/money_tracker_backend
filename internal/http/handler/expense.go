package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"log/slog"

	"github.com/go-chi/chi/v5"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/common"

	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/expense"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/shared"
	"github.com/niranjannayak5754/money_tracker_backend/internal/http/requestctx"
	"github.com/niranjannayak5754/money_tracker_backend/internal/http/response"
)

type ExpenseHandler struct {
	svc    expense.Service
	logger *slog.Logger
}

func NewExpenseHandler(
	svc expense.Service,
	logger *slog.Logger,
) *ExpenseHandler {
	return &ExpenseHandler{
		svc:    svc,
		logger: logger.With("handler", "expense"),
	}
}

// Routes for /expenses
func (h *ExpenseHandler) Routes() http.Handler {
	r := chi.NewRouter()

	r.Get("/", h.list)
	r.Post("/", h.create)
	r.Put("/{id}", h.update)
	r.Delete("/{id}", h.delete)

	return r
}

// POST /expenses
func (h *ExpenseHandler) create(w http.ResponseWriter, r *http.Request) {
	uid, ok := mustUID(w, r)
	if !ok {
		return
	}

	var in struct {
		Amount     float64             `json:"amount"`
		Date       shared.FlexibleTime `json:"date"`
		CategoryID string              `json:"category_id"`
		Merchant   string              `json:"merchant"`
		Notes      string              `json:"notes"`
		Tags       []string            `json:"tags"`
	}

	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.BadReq(w, `invalid json payload`)
		return
	}

	if in.Amount <= 0 {
		response.BadReq(w, "amount must be greater than zero")
		return
	}

	if in.CategoryID == "" {
		response.BadReq(w, `invalid category_id`)
		return
	}

	out, err := h.svc.Create(
		r.Context(),
		uid,
		expense.CreateInput{
			Amount:     in.Amount,
			Date:       shared.ChooseDate(in.Date.Time),
			CategoryID: common.CategoryID(in.CategoryID),
			Merchant:   in.Merchant,
			Notes:      in.Notes,
			Tags:       in.Tags,
		},
	)
	if err != nil {
		h.logger.Warn(
			"expense.create failed",
			"request_id", requestctx.RequestID(r.Context()),
			"uid", uid,
			"category_id", in.CategoryID,
			"err", err,
		)

		response.WriteError(w, r, err)
		return
	}

	response.JSON(w, http.StatusCreated, out)
}

// GET /expenses
func (h *ExpenseHandler) list(w http.ResponseWriter, r *http.Request) {
	uid, ok := mustUID(w, r)
	if !ok {
		return
	}

	month := r.URL.Query().Get("month")
	category := r.URL.Query().Get("category")

	items, err := h.svc.List(r.Context(), uid, month, category)
	if err != nil {
		h.logger.Error(
			"expense.list failed",
			"request_id", requestctx.RequestID(r.Context()),
			"uid", uid,
			"month", month,
			"category", category,
			"err", err,
		)

		response.WriteError(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, items)
}

// PUT /expenses/{id}
func (h *ExpenseHandler) update(w http.ResponseWriter, r *http.Request) {
	uid, ok := mustUID(w, r)
	if !ok {
		return
	}

	expenseID := common.ExpenseID(chi.URLParam(r, "id"))

	var in struct {
		Amount     *float64   `json:"amount"`
		Date       *time.Time `json:"date"`
		CategoryID *string    `json:"category_id"`
		Merchant   *string    `json:"merchant"`
		Notes      *string    `json:"notes"`
		Tags       *[]string  `json:"tags"`
	}

	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.BadReq(w, `invalid json payload`)
		return
	}

	var categoryID *common.CategoryID
	if in.CategoryID != nil {
		cid := common.CategoryID(*in.CategoryID)
		categoryID = &cid
	}

	err := h.svc.Update(
		r.Context(),
		uid,
		expenseID,
		expense.UpdateInput{
			Amount:     in.Amount,
			Date:       in.Date,
			CategoryID: categoryID,
			Merchant:   in.Merchant,
			Notes:      in.Notes,
			Tags:       in.Tags,
		},
	)

	if err != nil {
		h.logger.Error(
			"expense.update failed",
			"request_id", requestctx.RequestID(r.Context()),
			"uid", uid,
			"expense_id", expenseID,
			"err", err,
		)

		response.WriteError(w, r, err)
		return
	}

	response.OK(w)
}

// DELETE /expenses/{id}
func (h *ExpenseHandler) delete(w http.ResponseWriter, r *http.Request) {
	uid, ok := mustUID(w, r)
	if !ok {
		return
	}
	expenseID := common.ExpenseID(chi.URLParam(r, "id"))

	err := h.svc.Delete(r.Context(), uid, expenseID)
	if err != nil {
		h.logger.Error(
			"expense.delete failed",
			"request_id", requestctx.RequestID(r.Context()),
			"uid", uid,
			"expense_id", expenseID,
			"err", err,
		)

		response.WriteError(w, r, err)
		return
	}

	response.OK(w)
}
