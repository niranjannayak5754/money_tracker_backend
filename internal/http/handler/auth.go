package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"log/slog"

	"github.com/niranjannayak5754/money_tracker_backend/internal/config"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/user"
	"github.com/niranjannayak5754/money_tracker_backend/internal/http/requestctx"
	"github.com/niranjannayak5754/money_tracker_backend/internal/http/response"
	"github.com/niranjannayak5754/money_tracker_backend/internal/security"
)

type AuthHandler struct {
	svc    user.Service
	cfg    config.Config
	logger *slog.Logger
}

func NewAuthHandler(
	svc user.Service,
	cfg config.Config,
	logger *slog.Logger,
) *AuthHandler {
	return &AuthHandler{
		svc:    svc,
		cfg:    cfg,
		logger: logger.With("handler", "auth"),
	}
}

// POST /auth/register
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.BadReq(w, `invalid json payload`)
		return
	}

	u, err := h.svc.Register(r.Context(), user.RegisterInput{
		Email:    in.Email,
		Password: in.Password,
	})
	if err != nil {
		h.logger.Warn(
			"auth.register failed",
			"request_id", requestctx.RequestID(r.Context()),
			"email", in.Email,
			"err", err,
		)

		response.WriteError(w, r, err)
		return
	}

	response.JSON(w, http.StatusCreated, map[string]any{
		"id":    u.ID.Hex(),
		"email": u.Email,
	})
}

// POST /auth/login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.BadReq(w, `invalid json payload`)
		return
	}

	u, err := h.svc.Login(r.Context(), in.Email, in.Password)
	if err != nil {
		h.logger.Warn(
			"auth.login failed",
			"request_id", requestctx.RequestID(r.Context()),
			"email", in.Email,
			"err", err,
		)

		response.WriteError(w, r, err)
		return
	}

	token, err := security.Sign(
		h.cfg.JWTSecret,
		u.ID.Hex(),
		config.SEVEN_DAYS_IN_HOUR*time.Hour,
	)
	if err != nil {
		h.logger.Error(
			"auth.login jwt signing failed",
			"request_id", requestctx.RequestID(r.Context()),
			"uid", u.ID.Hex(),
			"err", err,
		)

		response.WriteError(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{
		"access_token": token,
	})
}

// GET /auth/me
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	uid, ok := mustUID(w, r)
	if !ok {
		return
	}

	u, err := h.svc.GetByID(r.Context(), uid)
	if err != nil {
		h.logger.Warn(
			"auth.me user not found",
			"request_id", requestctx.RequestID(r.Context()),
			"uid", uid.Hex(),
			"err", err,
		)

		response.WriteError(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]any{
		"id":         u.ID.Hex(),
		"email":      u.Email,
		"created_at": u.CreatedAt,
	})
}

// POST /auth/logout
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	// Stateless JWT logout — client just discards token
	response.JSON(w, http.StatusOK, map[string]string{
		"message": "logged out successfully",
	})
}

// POST /auth/reset-password
func (h *AuthHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	uid, ok := mustUID(w, r)
	if !ok {
		return
	}

	var in struct {
		NewPassword string `json:"new_password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.BadReq(w, "invalid json payload")
		return
	}

	if err := h.svc.ResetPassword(r.Context(), uid, in.NewPassword); err != nil {
		h.logger.Warn(
			"auth.reset_password failed",
			"request_id", requestctx.RequestID(r.Context()),
			"uid", uid.Hex(),
			"err", err,
		)

		response.WriteError(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "password changed successfully"})
}
