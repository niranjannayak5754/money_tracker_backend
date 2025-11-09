package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/niranjannayak5754/money_tracker_backend/internal/config"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/user"
	"github.com/niranjannayak5754/money_tracker_backend/internal/httpx"
	"github.com/niranjannayak5754/money_tracker_backend/internal/platform/contextutils"
	"github.com/niranjannayak5754/money_tracker_backend/internal/security"
)

type UserHandlers struct {
	svc user.Service
	cfg config.Config
}

func NewUserHandlers(svc user.Service, cfg config.Config) *UserHandlers {
	return &UserHandlers{svc: svc, cfg: cfg}
}

//
// AUTH HANDLERS
//

// POST /auth/register
func (h *UserHandlers) Register(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		httpx.BadReq(w, "invalid json")
		return
	}

	out, err := h.svc.Register(r.Context(), user.RegisterInput{
		Email:    in.Email,
		Password: in.Password,
	})

	if err != nil {
		httpx.BadReq(w, err.Error())
		return
	}

	httpx.JSON(w, 201, map[string]any{
		"id":    out.ID.Hex(),
		"email": out.Email,
	})
}

// POST /auth/login
func (h *UserHandlers) Login(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		httpx.BadReq(w, "invalid json")
		return
	}

	u, err := h.svc.Login(r.Context(), in.Email, in.Password)
	if err != nil {
		httpx.Unauthorized(w, "invalid credentials")
		return
	}

	tok, err := security.Sign(h.cfg.JWTSecret, u.ID.Hex(), 7*24*time.Hour)
	if err != nil {
		httpx.ServerErr(w, err)
		return
	}

	httpx.JSON(w, 200, map[string]string{
		"access_token": tok,
	})
}

// GET /auth/me
func (h *UserHandlers) Me(w http.ResponseWriter, r *http.Request) {
	uidHex := contextutils.UID(r.Context())
	if uidHex == "" {
		httpx.Unauthorized(w, "missing uid")
		return
	}

	id, _ := primitive.ObjectIDFromHex(uidHex)

	u, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		httpx.NotFound(w)
		return
	}

	httpx.JSON(w, 200, map[string]any{
		"id":         u.ID.Hex(),
		"email":      u.Email,
		"created_at": u.CreatedAt,
	})
}
