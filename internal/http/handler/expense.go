package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/expense"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/shared"
	"github.com/niranjannayak5754/money_tracker_backend/internal/http/response"
)

type ExpenseHandler struct {
	svc expense.Service
}

func NewExpenseHandler(svc expense.Service) *ExpenseHandler {
	return &ExpenseHandler{svc: svc}
}

// Routes() for /expenses
func (h *ExpenseHandler) Routes() http.Handler {
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
		response.BadReq(w, "invalid json")
		return
	}

	if in.Amount <= 0 {
		response.BadReq(w, "amount must be > 0")
		return
	}

	cid, err := primitive.ObjectIDFromHex(in.CategoryID)
	if err != nil {
		response.BadReq(w, "invalid category id")
		return
	}

	out, err := h.svc.Create(r.Context(), uid, expense.CreateInput{
		Amount:     in.Amount,
		Date:       shared.ChooseDate(in.Date.Time),
		CategoryID: cid,
		Merchant:   in.Merchant,
		Notes:      in.Notes,
		Tags:       in.Tags,
	})
	if err != nil {
		response.BadReq(w, err.Error())
		return
	}

	response.JSON(w, 201, out)
}

// GET /expenses?month=YYYY-MM&category=<id>
func (h *ExpenseHandler) list(w http.ResponseWriter, r *http.Request) {
	uid, ok := mustUID(w, r)
	if !ok {
		return
	}

	month := r.URL.Query().Get("month")
	category := r.URL.Query().Get("category")

	items, err := h.svc.List(r.Context(), uid, month, category)
	if err != nil {
		response.ServerErr(w, err)
		return
	}

	if items == nil {
		items = []expense.Model{}
	}

	response.JSON(w, 200, items)
}

// PUT /expenses/{id}
func (h *ExpenseHandler) update(w http.ResponseWriter, r *http.Request) {
	uid, ok := mustUID(w, r)
	if !ok {
		return
	}

	idHex := chi.URLParam(r, "id")
	expID, err := primitive.ObjectIDFromHex(idHex)
	if err != nil {
		response.BadReq(w, "bad expense id")
		return
	}

	var in struct {
		Amount     *float64   `json:"amount"`
		Date       *time.Time `json:"date"`
		CategoryID *string    `json:"category_id"`
		Merchant   *string    `json:"merchant"`
		Notes      *string    `json:"notes"`
		Tags       *[]string  `json:"tags"`
	}

	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.BadReq(w, "invalid json")
		return
	}

	var catID *primitive.ObjectID
	if in.CategoryID != nil {
		cid, err := primitive.ObjectIDFromHex(*in.CategoryID)
		if err == nil {
			catID = &cid
		}
	}

	updated, err := h.svc.Update(r.Context(), uid, expID, expense.UpdateInput{
		Amount:     in.Amount,
		Date:       in.Date,
		CategoryID: catID,
		Merchant:   in.Merchant,
		Notes:      in.Notes,
		Tags:       in.Tags,
	})
	if err != nil {
		response.ServerErr(w, err)
		return
	}

	if !updated {
		response.NotFound(w)
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

	idHex := chi.URLParam(r, "id")
	expID, err := primitive.ObjectIDFromHex(idHex)
	if err != nil {
		response.BadReq(w, "bad expense id")
		return
	}

	deleted, err := h.svc.Delete(r.Context(), uid, expID)
	if err != nil {
		response.ServerErr(w, err)
		return
	}

	if !deleted {
		response.NotFound(w)
		return
	}

	response.OK(w)
}
