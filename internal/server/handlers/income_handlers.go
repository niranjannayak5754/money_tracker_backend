package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/income"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/shared"
	"github.com/niranjannayak5754/money_tracker_backend/internal/httpx"
	"github.com/niranjannayak5754/money_tracker_backend/internal/platform/contextutils"
)

type IncomeHandlers struct {
	svc income.Service
}

func NewIncomeHandlers(svc income.Service) *IncomeHandlers {
	return &IncomeHandlers{svc: svc}
}

// Public Routes for /income
func (h *IncomeHandlers) Routes() http.Handler {
	r := chi.NewRouter()

	r.Get("/", h.list)
	r.Post("/", h.create)
	r.Put("/{id}", h.update)
	r.Delete("/{id}", h.delete)

	return r
}

//
// HANDLERS
//

// POST /income
func (h *IncomeHandlers) create(w http.ResponseWriter, r *http.Request) {
	uidHex := contextutils.UID(r.Context())
	uid, _ := primitive.ObjectIDFromHex(uidHex)

	var in struct {
		Amount float64   `json:"amount"`
		Date   time.Time `json:"date"`
		Source string    `json:"source"`
		Notes  string    `json:"notes"`
	}

	if err := json.NewDecoder(r.Body).Decode(&in); err != nil || in.Amount <= 0 {
		httpx.BadReq(w, "invalid json/amount")
		return
	}

	rec, err := h.svc.Create(r.Context(), uid, income.CreateInput{
		Amount: in.Amount,
		Date:   shared.ChooseDate(in.Date),
		Source: in.Source,
		Notes:  in.Notes,
	})
	if err != nil {
		httpx.ServerErr(w, err)
		return
	}

	httpx.JSON(w, 201, rec)
}

// GET /income?month=YYYY-MM
func (h *IncomeHandlers) list(w http.ResponseWriter, r *http.Request) {
	uidHex := contextutils.UID(r.Context())
	uid, _ := primitive.ObjectIDFromHex(uidHex)

	month := r.URL.Query().Get("month")

	items, err := h.svc.List(r.Context(), uid, month)
	if err != nil {
		httpx.ServerErr(w, err)
		return
	}

	httpx.JSON(w, 200, items)
}

// PUT /income/{id}
func (h *IncomeHandlers) update(w http.ResponseWriter, r *http.Request) {
	uidHex := contextutils.UID(r.Context())
	uid, _ := primitive.ObjectIDFromHex(uidHex)

	id, err := primitive.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		httpx.BadReq(w, "bad id")
		return
	}

	var in struct {
		Amount *float64   `json:"amount"`
		Date   *time.Time `json:"date"`
		Source *string    `json:"source"`
		Notes  *string    `json:"notes"`
	}

	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		httpx.BadReq(w, "invalid json")
		return
	}

	ok, err := h.svc.Update(r.Context(), uid, id, income.UpdateInput{
		Amount: in.Amount,
		Date:   in.Date,
		Source: in.Source,
		Notes:  in.Notes,
	})
	if err != nil {
		httpx.ServerErr(w, err)
		return
	}

	if !ok {
		httpx.NotFound(w)
		return
	}

	httpx.OK(w)
}

// DELETE /income/{id}
func (h *IncomeHandlers) delete(w http.ResponseWriter, r *http.Request) {
	uidHex := contextutils.UID(r.Context())
	uid, _ := primitive.ObjectIDFromHex(uidHex)

	id, err := primitive.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		httpx.BadReq(w, "bad id")
		return
	}

	ok, err := h.svc.Delete(r.Context(), uid, id)
	if err != nil {
		httpx.ServerErr(w, err)
		return
	}

	if !ok {
		httpx.NotFound(w)
		return
	}

	httpx.OK(w)
}
