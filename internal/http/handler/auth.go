package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/niranjannayak5754/money_tracker_backend/internal/config"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/user"
	"github.com/niranjannayak5754/money_tracker_backend/internal/http/response"
	"github.com/niranjannayak5754/money_tracker_backend/internal/security"
)

type AuthHandler struct {
	svc user.Service
	cfg config.Config
}

func NewAuthHandler(svc user.Service, cfg config.Config) *AuthHandler {
	return &AuthHandler{svc: svc, cfg: cfg}
}

// POST /auth/register
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.BadReq(w, "invalid json")
		return
	}

	out, err := h.svc.Register(r.Context(), user.RegisterInput{
		Email:    in.Email,
		Password: in.Password,
	})
	if err != nil {
		response.BadReq(w, err.Error())
		return
	}

	response.JSON(w, http.StatusCreated, map[string]any{
		"id":    out.ID.Hex(),
		"email": out.Email,
	})
}

// POST /auth/login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.BadReq(w, "invalid json")
		return
	}

	u, err := h.svc.Login(r.Context(), in.Email, in.Password)
	if err != nil {
		response.Unauthorized(w, "invalid credentials")
		return
	}

	tok, err := security.Sign(
		h.cfg.JWTSecret,
		u.ID.Hex(),
		7*24*time.Hour,
	)
	if err != nil {
		response.ServerErr(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{
		"access_token": tok,
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
		response.NotFound(w)
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
	response.JSON(w, http.StatusOK, map[string]string{
		"message": "logged out successfully",
	})
}
