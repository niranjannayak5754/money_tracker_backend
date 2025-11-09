package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/expense"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/shared"
	"github.com/niranjannayak5754/money_tracker_backend/internal/httpx"
	"github.com/niranjannayak5754/money_tracker_backend/internal/platform/contextutils"
)

type ExpenseHandlers struct {
	svc expense.Service
}

func NewExpenseHandlers(svc expense.Service) *ExpenseHandlers {
	return &ExpenseHandlers{svc: svc}
}

// Routes() for /expenses
func (h *ExpenseHandlers) Routes() http.Handler {
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
func (h *ExpenseHandlers) create(w http.ResponseWriter, r *http.Request) {
	uidHex := contextutils.UID(r.Context())
	uid, _ := primitive.ObjectIDFromHex(uidHex)

	var in struct {
		Amount     float64   `json:"amount"`
		Date       time.Time `json:"date"`
		CategoryID string    `json:"category_id"`
		Merchant   string    `json:"merchant"`
		Notes      string    `json:"notes"`
		Tags       []string  `json:"tags"`
	}

	if err := json.NewDecoder(r.Body).Decode(&in); err != nil || in.Amount <= 0 {
		httpx.BadReq(w, "invalid json/amount")
		return
	}

	cid, err := primitive.ObjectIDFromHex(in.CategoryID)
	if err != nil {
		httpx.BadReq(w, "invalid category id")
		return
	}

	out, err := h.svc.Create(r.Context(), uid, expense.CreateInput{
		Amount:     in.Amount,
		Date:       shared.ChooseDate(in.Date),
		CategoryID: cid,
		Merchant:   in.Merchant,
		Notes:      in.Notes,
		Tags:       in.Tags,
	})
	if err != nil {
		httpx.BadReq(w, err.Error())
		return
	}

	httpx.JSON(w, 201, out)
}

// GET /expenses?month=YYYY-MM&category=<id>
func (h *ExpenseHandlers) list(w http.ResponseWriter, r *http.Request) {
	uidHex := contextutils.UID(r.Context())
	uid, _ := primitive.ObjectIDFromHex(uidHex)

	month := r.URL.Query().Get("month")
	category := r.URL.Query().Get("category")

	items, err := h.svc.List(r.Context(), uid, month, category)
	if err != nil {
		httpx.ServerErr(w, err)
		return
	}

	httpx.JSON(w, 200, items)
}

// PUT /expenses/{id}
func (h *ExpenseHandlers) update(w http.ResponseWriter, r *http.Request) {
	uidHex := contextutils.UID(r.Context())
	uid, _ := primitive.ObjectIDFromHex(uidHex)

	idHex := chi.URLParam(r, "id")
	expID, err := primitive.ObjectIDFromHex(idHex)
	if err != nil {
		httpx.BadReq(w, "bad expense id")
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
		httpx.BadReq(w, "invalid json")
		return
	}

	var catID *primitive.ObjectID
	if in.CategoryID != nil {
		cid, err := primitive.ObjectIDFromHex(*in.CategoryID)
		if err == nil {
			catID = &cid
		}
	}

	ok, err := h.svc.Update(r.Context(), uid, expID, expense.UpdateInput{
		Amount:     in.Amount,
		Date:       in.Date,
		CategoryID: catID,
		Merchant:   in.Merchant,
		Notes:      in.Notes,
		Tags:       in.Tags,
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

// DELETE /expenses/{id}
func (h *ExpenseHandlers) delete(w http.ResponseWriter, r *http.Request) {
	uidHex := contextutils.UID(r.Context())
	uid, _ := primitive.ObjectIDFromHex(uidHex)

	idHex := chi.URLParam(r, "id")
	expID, err := primitive.ObjectIDFromHex(idHex)
	if err != nil {
		httpx.BadReq(w, "bad expense id")
		return
	}

	ok, err := h.svc.Delete(r.Context(), uid, expID)
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
