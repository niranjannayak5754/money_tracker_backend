package handler

import (
	"net/http"

	"log/slog"

	"github.com/go-chi/chi/v5"

	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/common"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/notification"
	"github.com/niranjannayak5754/money_tracker_backend/internal/http/requestctx"
	"github.com/niranjannayak5754/money_tracker_backend/internal/http/response"
)

type NotificationHandler struct {
	svc    notification.Service
	logger *slog.Logger
}

func NewNotificationHandler(svc notification.Service, logger *slog.Logger) *NotificationHandler {
	return &NotificationHandler{svc: svc, logger: logger.With("handler", "notification")}
}

func (h *NotificationHandler) Routes() http.Handler {
	r := chi.NewRouter()
	r.Get("/", h.list)
	r.Get("/unread-count", h.unreadCount)
	r.Post("/{id}/read", h.markRead)
	r.Post("/{id}/dismiss", h.dismiss)
	return r
}

func (h *NotificationHandler) list(w http.ResponseWriter, r *http.Request) {
	uid, ok := mustUID(w, r)
	if !ok {
		return
	}

	onlyUnread := r.URL.Query().Get("unread") == "true"

	items, err := h.svc.List(r.Context(), uid, onlyUnread)
	if err != nil {
		h.logger.Error("notification.list failed", "request_id", requestctx.RequestID(r.Context()), "uid", uid, "err", err)
		response.WriteError(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, items)
}

func (h *NotificationHandler) unreadCount(w http.ResponseWriter, r *http.Request) {
	uid, ok := mustUID(w, r)
	if !ok {
		return
	}

	count, err := h.svc.UnreadCount(r.Context(), uid)
	if err != nil {
		h.logger.Error("notification.unread_count failed", "request_id", requestctx.RequestID(r.Context()), "uid", uid, "err", err)
		response.WriteError(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]int64{"unread_count": count})
}

func (h *NotificationHandler) markRead(w http.ResponseWriter, r *http.Request) {
	uid, ok := mustUID(w, r)
	if !ok {
		return
	}

	id := common.NotificationID(chi.URLParam(r, "id"))
	if err := h.svc.MarkRead(r.Context(), uid, id); err != nil {
		h.logger.Error("notification.mark_read failed", "request_id", requestctx.RequestID(r.Context()), "uid", uid, "err", err)
		response.WriteError(w, r, err)
		return
	}

	response.OK(w)
}

func (h *NotificationHandler) dismiss(w http.ResponseWriter, r *http.Request) {
	uid, ok := mustUID(w, r)
	if !ok {
		return
	}

	id := common.NotificationID(chi.URLParam(r, "id"))
	if err := h.svc.Dismiss(r.Context(), uid, id); err != nil {
		h.logger.Error("notification.dismiss failed", "request_id", requestctx.RequestID(r.Context()), "uid", uid, "err", err)
		response.WriteError(w, r, err)
		return
	}

	response.OK(w)
}
