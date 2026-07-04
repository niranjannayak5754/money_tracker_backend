package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"log/slog"

	"github.com/go-chi/chi/v5"

	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/common"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/recurring"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/shared"
	"github.com/niranjannayak5754/money_tracker_backend/internal/http/requestctx"
	"github.com/niranjannayak5754/money_tracker_backend/internal/http/response"
)

type RecurringHandler struct {
	svc    recurring.Service
	logger *slog.Logger
}

func NewRecurringHandler(svc recurring.Service, logger *slog.Logger) *RecurringHandler {
	return &RecurringHandler{svc: svc, logger: logger.With("handler", "recurring")}
}

func (h *RecurringHandler) Routes() http.Handler {
	r := chi.NewRouter()
	r.Get("/", h.list)
	r.Post("/", h.create)
	r.Put("/{id}", h.update)
	r.Delete("/{id}", h.delete)
	return r
}

func (h *RecurringHandler) create(w http.ResponseWriter, r *http.Request) {
	uid, ok := mustUID(w, r)
	if !ok {
		return
	}

	var in struct {
		EntityType string               `json:"entity_type"`
		Frequency  string               `json:"frequency"`
		DayOfMonth int                  `json:"day_of_month"`
		Payload    map[string]any       `json:"payload"`
		StartDate  shared.FlexibleTime  `json:"start_date"`
		EndDate    *shared.FlexibleTime `json:"end_date"`
	}

	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.BadReq(w, "invalid json payload")
		return
	}

	var endDate *time.Time
	if in.EndDate != nil {
		d := in.EndDate.Time
		endDate = &d
	}

	out, err := h.svc.Create(r.Context(), uid, recurring.CreateInput{
		EntityType: recurring.EntityType(in.EntityType),
		Frequency:  recurring.Frequency(in.Frequency),
		DayOfMonth: in.DayOfMonth,
		Payload:    in.Payload,
		StartDate:  in.StartDate.Time,
		EndDate:    endDate,
	})
	if err != nil {
		h.logger.Error("recurring.create failed", "request_id", requestctx.RequestID(r.Context()), "uid", uid, "err", err)
		response.WriteError(w, r, err)
		return
	}

	response.JSON(w, http.StatusCreated, out)
}

func (h *RecurringHandler) list(w http.ResponseWriter, r *http.Request) {
	uid, ok := mustUID(w, r)
	if !ok {
		return
	}

	items, err := h.svc.List(r.Context(), uid)
	if err != nil {
		h.logger.Error("recurring.list failed", "request_id", requestctx.RequestID(r.Context()), "uid", uid, "err", err)
		response.WriteError(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, items)
}

func (h *RecurringHandler) update(w http.ResponseWriter, r *http.Request) {
	uid, ok := mustUID(w, r)
	if !ok {
		return
	}

	id := common.RecurringID(chi.URLParam(r, "id"))

	var in struct {
		DayOfMonth *int                 `json:"day_of_month"`
		Payload    map[string]any       `json:"payload"`
		EndDate    *shared.FlexibleTime `json:"end_date"`
		Active     *bool                `json:"active"`
	}

	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.BadReq(w, "invalid json payload")
		return
	}

	var endDate *time.Time
	if in.EndDate != nil {
		d := in.EndDate.Time
		endDate = &d
	}

	err := h.svc.Update(r.Context(), uid, id, recurring.UpdateInput{
		DayOfMonth: in.DayOfMonth,
		Payload:    in.Payload,
		EndDate:    endDate,
		Active:     in.Active,
	})
	if err != nil {
		h.logger.Error("recurring.update failed", "request_id", requestctx.RequestID(r.Context()), "uid", uid, "err", err)
		response.WriteError(w, r, err)
		return
	}

	response.OK(w)
}

func (h *RecurringHandler) delete(w http.ResponseWriter, r *http.Request) {
	uid, ok := mustUID(w, r)
	if !ok {
		return
	}

	id := common.RecurringID(chi.URLParam(r, "id"))
	if err := h.svc.Delete(r.Context(), uid, id); err != nil {
		h.logger.Error("recurring.delete failed", "request_id", requestctx.RequestID(r.Context()), "uid", uid, "err", err)
		response.WriteError(w, r, err)
		return
	}

	response.OK(w)
}
