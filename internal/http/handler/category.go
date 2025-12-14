package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/category"
	"github.com/niranjannayak5754/money_tracker_backend/internal/http/response"
)

type CategoryHandler struct {
	svc category.Service
}

func NewCategoryHandler(svc category.Service) *CategoryHandler {
	return &CategoryHandler{svc: svc}
}

// Routes returns a chi.Router for /categories
func (h *CategoryHandler) Routes() http.Handler {
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
func (h *CategoryHandler) list(w http.ResponseWriter, r *http.Request) {
	uid, ok := mustUID(w, r)
	if !ok {
		return
	}

	items, err := h.svc.List(r.Context(), uid)
	if err != nil {
		response.ServerErr(w, err)
		return
	}

	if items == nil {
		items = []category.Model{}
	}

	response.JSON(w, 200, items)
}

// POST /categories
func (h *CategoryHandler) create(w http.ResponseWriter, r *http.Request) {
	uid, ok := mustUID(w, r)
	if !ok {
		return
	}

	var in struct {
		Name string `json:"name"`
		Type string `json:"type"` // optional (expense|income)
	}

	if err := json.NewDecoder(r.Body).Decode(&in); err != nil || in.Name == "" {
		response.BadReq(w, "invalid json")
		return
	}

	out, err := h.svc.Create(r.Context(), uid, category.CreateInput{
		Name: in.Name,
		Type: in.Type,
	})
	if err != nil {
		response.BadReq(w, err.Error())
		return
	}

	response.JSON(w, 201, out)
}

// PUT /categories/{id}
func (h *CategoryHandler) update(w http.ResponseWriter, r *http.Request) {
	uid, ok := mustUID(w, r)
	if !ok {
		return
	}

	idHex := chi.URLParam(r, "id")
	catID, err := primitive.ObjectIDFromHex(idHex)
	if err != nil {
		response.BadReq(w, "bad category id")
		return
	}

	var in struct {
		Name     *string `json:"name"`
		Archived *bool   `json:"archived"`
	}

	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.BadReq(w, "invalid json")
		return
	}

	updated, err := h.svc.Update(r.Context(), uid, catID, category.UpdateInput{
		Name:     in.Name,
		Archived: in.Archived,
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
