package middleware

import (
	"net/http"
	"strings"

	"github.com/niranjannayak5754/money_tracker_backend/internal/http/requestctx"
	"github.com/niranjannayak5754/money_tracker_backend/internal/http/response"
	"github.com/niranjannayak5754/money_tracker_backend/internal/security"
)

type AuthMiddleware struct {
	jwtSecret string
}

func NewAuthMiddleware(secret string) *AuthMiddleware {
	return &AuthMiddleware{jwtSecret: secret}
}

func (a *AuthMiddleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		h := r.Header.Get("Authorization")
		if !strings.HasPrefix(h, "Bearer ") {
			response.Unauthorized(w, "missing token")
			return
		}

		token := strings.TrimPrefix(h, "Bearer ")

		var claims security.Claims
		parsed, err := security.Parse(a.jwtSecret, token, &claims)
		if err != nil || !parsed.Valid {
			response.Unauthorized(w, "invalid token")
			return
		}

		r = r.WithContext(requestctx.WithUID(r.Context(), claims.UID))
		next.ServeHTTP(w, r)
	})
}
