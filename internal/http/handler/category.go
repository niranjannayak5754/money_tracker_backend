package handler

import (
	"encoding/json"
	"net/http"

	"log/slog"

	"github.com/go-chi/chi/v5"

	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/category"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/common"
	"github.com/niranjannayak5754/money_tracker_backend/internal/http/requestctx"
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

	ctx := r.Context()
	archived := r.URL.Query().Get("archived")
	var (
		items []category.Model
		err   error
	)

	switch archived {
	case "":
		items, err = h.svc.List(ctx, uid)
	case "false":
		items, err = h.svc.ListActive(ctx, uid)
	case "true":
		items, err = h.svc.ListArchived(ctx, uid)
	default:
		response.BadReq(w, "archived must be true or false")
		return
	}

	if err != nil {
		h.logger.Error(
			"category.list failed",
			"request_id", requestctx.RequestID(r.Context()),
			"uid", uid,
			"archived", archived,
			"err", err,
		)
		response.WriteError(w, r, err)
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

	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.BadReq(w, "invalid json payload")
		return
	}

	out, err := h.svc.Create(r.Context(), uid, category.CreateInput{
		Name: in.Name,
		Type: in.Type,
	})
	if err != nil {
		h.logger.Error(
			"category.create failed",
			"request_id", requestctx.RequestID(r.Context()),
			"uid", uid,
			"err", err,
		)
		response.WriteError(w, r, err)
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

	catID := common.CategoryID(chi.URLParam(r, "id"))
	if catID == "" {
		response.BadReq(w, "bad category id")
		return
	}

	var in struct {
		Name     *string `json:"name"`
		Archived *bool   `json:"archived"`
	}

	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.BadReq(w, "invalid json payload")
		return
	}

	err := h.svc.Update(
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
			"category.update failed",
			"request_id", requestctx.RequestID(r.Context()),
			"uid", uid,
			"err", err,
		)
		response.WriteError(w, r, err)
		return
	}

	response.OK(w)
}
