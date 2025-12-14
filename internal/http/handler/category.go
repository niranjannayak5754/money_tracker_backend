package handler

import (
	"encoding/json"
	"net/http"

	"log/slog"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/category"
	"github.com/niranjannayak5754/money_tracker_backend/internal/http/response"
)

type CategoryHandler struct {
	svc    category.Service
	logger *slog.Logger
}

func NewCategoryHandler(
	svc category.Service,
	logger *slog.Logger,
) *CategoryHandler {
	return &CategoryHandler{
		svc:    svc,
		logger: logger.With("handler", "category"),
	}
}

// Routes returns a chi.Router for /categories
func (h *CategoryHandler) Routes() http.Handler {
	r := chi.NewRouter()

	r.Get("/", h.list)
	r.Post("/", h.create)
	r.Put("/{id}", h.update)

	return r
}

// GET /categories
func (h *CategoryHandler) list(w http.ResponseWriter, r *http.Request) {
	uid, ok := mustUID(w, r)
	if !ok {
		return
	}

	items, err := h.svc.List(r.Context(), uid)
	if err != nil {
		h.logger.Error(
			"list categories failed",
			"uid", uid.Hex(),
			"err", err,
		)
		response.ServerErr(w, err)
		return
	}

	if items == nil {
		items = []category.Model{}
	}

	response.JSON(w, http.StatusOK, items)
}

// POST /categories
func (h *CategoryHandler) create(w http.ResponseWriter, r *http.Request) {
	uid, ok := mustUID(w, r)
	if !ok {
		return
	}

	var in struct {
		Name string `json:"name"`
		Type string `json:"type"`
	}

	if err := json.NewDecoder(r.Body).Decode(&in); err != nil || in.Name == "" {
		h.logger.Warn(
			"invalid create category payload",
			"uid", uid.Hex(),
			"err", err,
		)
		response.BadReq(w, "invalid json")
		return
	}

	out, err := h.svc.Create(r.Context(), uid, category.CreateInput{
		Name: in.Name,
		Type: in.Type,
	})
	if err != nil {
		h.logger.Warn(
			"create category failed",
			"uid", uid.Hex(),
			"name", in.Name,
			"err", err,
		)
		response.BadReq(w, err.Error())
		return
	}

	response.JSON(w, http.StatusCreated, out)
}

// PUT /categories/{id}
func (h *CategoryHandler) update(w http.ResponseWriter, r *http.Request) {
	uid, ok := mustUID(w, r)
	if !ok {
		return
	}

	catID, err := primitive.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		response.BadReq(w, "bad category id")
		return
	}

	var in struct {
		Name     *string `json:"name"`
		Archived *bool   `json:"archived"`
	}

	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		h.logger.Warn(
			"invalid update category payload",
			"category_id", catID.Hex(),
			"uid", uid.Hex(),
			"err", err,
		)
		response.BadReq(w, "invalid json")
		return
	}

	updated, err := h.svc.Update(
		r.Context(),
		uid,
		catID,
		category.UpdateInput{
			Name:     in.Name,
			Archived: in.Archived,
		},
	)
	if err != nil {
		h.logger.Error(
			"update category failed",
			"category_id", catID.Hex(),
			"uid", uid.Hex(),
			"err", err,
		)
		response.ServerErr(w, err)
		return
	}

	if !updated {
		response.NotFound(w)
		return
	}

	response.OK(w)
}
