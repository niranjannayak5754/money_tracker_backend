package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"log/slog"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/niranjannayak5754/money_tracker_backend/internal/apperr"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/income"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/shared"
	"github.com/niranjannayak5754/money_tracker_backend/internal/http/requestctx"
	"github.com/niranjannayak5754/money_tracker_backend/internal/http/response"
)

type IncomeHandler struct {
	svc    income.Service
	logger *slog.Logger
}

func NewIncomeHandler(
	svc income.Service,
	logger *slog.Logger,
) *IncomeHandler {
	return &IncomeHandler{
		svc:    svc,
		logger: logger.With("handler", "income"),
	}
}

// Routes for /income
func (h *IncomeHandler) Routes() http.Handler {
	r := chi.NewRouter()

	r.Get("/", h.list)
	r.Post("/", h.create)
	r.Put("/{id}", h.update)
	r.Delete("/{id}", h.delete)

	return r
}

// POST /income
func (h *IncomeHandler) create(w http.ResponseWriter, r *http.Request) {
	uid, ok := mustUID(w, r)
	if !ok {
		return
	}

	var in struct {
		Amount float64             `json:"amount"`
		Date   shared.FlexibleTime `json:"date"`
		Source string              `json:"source"`
		Notes  string              `json:"notes"`
	}

	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		appErr := apperr.ValidationErr("invalid json payload")

		h.logger.Warn(
			"income.create invalid payload",
			"request_id", requestctx.UID(r.Context()),
			"user_id", uid.Hex(),
			"err", err,
		)

		response.WriteError(w, r, appErr)
		return
	}

	if in.Amount <= 0 {
		appErr := apperr.ValidationErr("amount must be > 0")

		h.logger.Warn(
			"income.create invalid amount",
			"request_id", requestctx.UID(r.Context()),
			"user_id", uid.Hex(),
		)

		response.WriteError(w, r, appErr)
		return
	}

	rec, err := h.svc.Create(
		r.Context(),
		uid,
		income.CreateInput{
			Amount: in.Amount,
			Date:   shared.ChooseDate(in.Date.Time),
			Source: in.Source,
			Notes:  in.Notes,
		},
	)
	if err != nil {
		h.logger.Error(
			"income.create failed",
			"request_id", requestctx.UID(r.Context()),
			"user_id", uid.Hex(),
			"err", err,
		)

		response.WriteError(w, r, err)
		return
	}

	response.JSON(w, http.StatusCreated, rec)
}

// GET /income?month=YYYY-MM
func (h *IncomeHandler) list(w http.ResponseWriter, r *http.Request) {
	uid, ok := mustUID(w, r)
	if !ok {
		return
	}

	month := r.URL.Query().Get("month")

	items, err := h.svc.List(r.Context(), uid, month)
	if err != nil {
		h.logger.Error(
			"income.list failed",
			"request_id", requestctx.UID(r.Context()),
			"user_id", uid.Hex(),
			"month", month,
			"err", err,
		)

		response.WriteError(w, r, err)
		return
	}

	if items == nil {
		items = []income.Model{}
	}

	response.JSON(w, http.StatusOK, items)
}

// PUT /income/{id}
func (h *IncomeHandler) update(w http.ResponseWriter, r *http.Request) {
	uid, ok := mustUID(w, r)
	if !ok {
		return
	}

	incomeID, err := primitive.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		appErr := apperr.ValidationErr("invalid income id")

		h.logger.Warn(
			"income.update invalid id",
			"request_id", requestctx.UID(r.Context()),
			"user_id", uid.Hex(),
			"err", err,
		)

		response.WriteError(w, r, appErr)
		return
	}

	var in struct {
		Amount *float64   `json:"amount"`
		Date   *time.Time `json:"date"`
		Source *string    `json:"source"`
		Notes  *string    `json:"notes"`
	}

	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		appErr := apperr.ValidationErr("invalid json payload")

		h.logger.Warn(
			"income.update invalid payload",
			"request_id", requestctx.UID(r.Context()),
			"user_id", uid.Hex(),
			"income_id", incomeID.Hex(),
			"err", err,
		)

		response.WriteError(w, r, appErr)
		return
	}

	err = h.svc.Update(
		r.Context(),
		uid,
		incomeID,
		income.UpdateInput{
			Amount: in.Amount,
			Date:   in.Date,
			Source: in.Source,
			Notes:  in.Notes,
		},
	)

	if err != nil {
		h.logger.Error(
			"income.update failed",
			"request_id", requestctx.UID(r.Context()),
			"user_id", uid.Hex(),
			"income_id", incomeID.Hex(),
			"err", err,
		)

		response.WriteError(w, r, err)
		return
	}

	response.OK(w)
}

// DELETE /income/{id}
func (h *IncomeHandler) delete(w http.ResponseWriter, r *http.Request) {
	uid, ok := mustUID(w, r)
	if !ok {
		return
	}

	incomeID, err := primitive.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		appErr := apperr.ValidationErr("invalid income id")

		h.logger.Warn(
			"income.delete invalid id",
			"request_id", requestctx.UID(r.Context()),
			"user_id", uid.Hex(),
			"err", err,
		)

		response.WriteError(w, r, appErr)
		return
	}

	err = h.svc.Delete(r.Context(), uid, incomeID)
	if err != nil {
		h.logger.Error(
			"income.delete failed",
			"request_id", requestctx.UID(r.Context()),
			"user_id", uid.Hex(),
			"income_id", incomeID.Hex(),
			"err", err,
		)

		response.WriteError(w, r, err)
		return
	}

	response.OK(w)
}
