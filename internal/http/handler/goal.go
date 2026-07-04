package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"log/slog"

	"github.com/go-chi/chi/v5"

	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/common"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/goal"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/shared"
	"github.com/niranjannayak5754/money_tracker_backend/internal/http/requestctx"
	"github.com/niranjannayak5754/money_tracker_backend/internal/http/response"
)

// flexibleTimePtr converts an optional shared.FlexibleTime (used for JSON
// decoding) into the plain *time.Time the domain layer expects.
func flexibleTimePtr(ft *shared.FlexibleTime) *time.Time {
	if ft == nil {
		return nil
	}
	t := ft.Time
	return &t
}

type GoalHandler struct {
	svc    goal.Service
	logger *slog.Logger
}

func NewGoalHandler(svc goal.Service, logger *slog.Logger) *GoalHandler {
	return &GoalHandler{svc: svc, logger: logger.With("handler", "goal")}
}

func (h *GoalHandler) Routes() http.Handler {
	r := chi.NewRouter()
	r.Get("/", h.list)
	r.Post("/", h.create)
	r.Put("/{id}", h.update)
	r.Delete("/{id}", h.delete)
	return r
}

func (h *GoalHandler) create(w http.ResponseWriter, r *http.Request) {
	uid, ok := mustUID(w, r)
	if !ok {
		return
	}

	var in struct {
		Name         string               `json:"name"`
		TargetAmount float64              `json:"target_amount"`
		TargetDate   *shared.FlexibleTime `json:"target_date"`
		LinkedType   string               `json:"linked_type"`
		LinkedID     string               `json:"linked_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.BadReq(w, "invalid json payload")
		return
	}

	out, err := h.svc.Create(r.Context(), uid, goal.CreateInput{
		Name:         in.Name,
		TargetAmount: in.TargetAmount,
		TargetDate:   flexibleTimePtr(in.TargetDate),
		LinkedType:   goal.LinkedType(in.LinkedType),
		LinkedID:     in.LinkedID,
	})
	if err != nil {
		h.logger.Error("goal.create failed", "request_id", requestctx.RequestID(r.Context()), "uid", uid, "err", err)
		response.WriteError(w, r, err)
		return
	}

	response.JSON(w, http.StatusCreated, out)
}

func (h *GoalHandler) list(w http.ResponseWriter, r *http.Request) {
	uid, ok := mustUID(w, r)
	if !ok {
		return
	}

	items, err := h.svc.List(r.Context(), uid)
	if err != nil {
		h.logger.Error("goal.list failed", "request_id", requestctx.RequestID(r.Context()), "uid", uid, "err", err)
		response.WriteError(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, items)
}

func (h *GoalHandler) update(w http.ResponseWriter, r *http.Request) {
	uid, ok := mustUID(w, r)
	if !ok {
		return
	}

	id := common.GoalID(chi.URLParam(r, "id"))

	var in struct {
		Name         *string              `json:"name"`
		TargetAmount *float64             `json:"target_amount"`
		TargetDate   *shared.FlexibleTime `json:"target_date"`
		Status       *string              `json:"status"`
	}

	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.BadReq(w, "invalid json payload")
		return
	}

	var status *goal.Status
	if in.Status != nil {
		st := goal.Status(*in.Status)
		status = &st
	}

	err := h.svc.Update(r.Context(), uid, id, goal.UpdateInput{
		Name:         in.Name,
		TargetAmount: in.TargetAmount,
		TargetDate:   flexibleTimePtr(in.TargetDate),
		Status:       status,
	})
	if err != nil {
		h.logger.Error("goal.update failed", "request_id", requestctx.RequestID(r.Context()), "uid", uid, "err", err)
		response.WriteError(w, r, err)
		return
	}

	response.OK(w)
}

func (h *GoalHandler) delete(w http.ResponseWriter, r *http.Request) {
	uid, ok := mustUID(w, r)
	if !ok {
		return
	}

	id := common.GoalID(chi.URLParam(r, "id"))
	if err := h.svc.Delete(r.Context(), uid, id); err != nil {
		h.logger.Error("goal.delete failed", "request_id", requestctx.RequestID(r.Context()), "uid", uid, "err", err)
		response.WriteError(w, r, err)
		return
	}

	response.OK(w)
}
