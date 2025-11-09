package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/category"
	"github.com/niranjannayak5754/money_tracker_backend/internal/httpx"
	"github.com/niranjannayak5754/money_tracker_backend/internal/platform/contextutils"
)

type CategoryHandlers struct {
	svc category.Service
}

func NewCategoryHandlers(svc category.Service) *CategoryHandlers {
	return &CategoryHandlers{svc: svc}
}

// Routes returns a chi.Router for /categories
func (h *CategoryHandlers) Routes() http.Handler {
	r := chi.NewRouter()

	r.Get("/", h.list)
	r.Post("/", h.create)
	r.Put("/{id}", h.update)

	return r
}

//
// HANDLERS
//

// GET /categories
func (h *CategoryHandlers) list(w http.ResponseWriter, r *http.Request) {
	uidHex := contextutils.UID(r.Context())
	uid, _ := primitive.ObjectIDFromHex(uidHex)

	items, err := h.svc.List(r.Context(), uid)
	if err != nil {
		httpx.ServerErr(w, err)
		return
	}

	httpx.JSON(w, 200, items)
}

// POST /categories
func (h *CategoryHandlers) create(w http.ResponseWriter, r *http.Request) {
	uidHex := contextutils.UID(r.Context())
	uid, _ := primitive.ObjectIDFromHex(uidHex)

	var in struct {
		Name string `json:"name"`
		Type string `json:"type"` // optional (expense|income)
	}

	if err := json.NewDecoder(r.Body).Decode(&in); err != nil || in.Name == "" {
		httpx.BadReq(w, "invalid json")
		return
	}

	out, err := h.svc.Create(r.Context(), uid, category.CreateInput{
		Name: in.Name,
		Type: in.Type,
	})
	if err != nil {
		httpx.BadReq(w, err.Error())
		return
	}

	httpx.JSON(w, 201, out)
}

// PUT /categories/{id}
func (h *CategoryHandlers) update(w http.ResponseWriter, r *http.Request) {
	uidHex := contextutils.UID(r.Context())
	uid, _ := primitive.ObjectIDFromHex(uidHex)

	idHex := chi.URLParam(r, "id")
	catID, err := primitive.ObjectIDFromHex(idHex)
	if err != nil {
		httpx.BadReq(w, "bad category id")
		return
	}

	var in struct {
		Name     *string `json:"name"`
		Archived *bool   `json:"archived"`
	}

	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		httpx.BadReq(w, "invalid json")
		return
	}

	ok, err := h.svc.Update(r.Context(), uid, catID, category.UpdateInput{
		Name:     in.Name,
		Archived: in.Archived,
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
