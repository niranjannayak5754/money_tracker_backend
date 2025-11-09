package server

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/niranjannayak5754/money_tracker_backend/internal/httpx"
	"github.com/niranjannayak5754/money_tracker_backend/internal/platform/contextutils"
	"github.com/niranjannayak5754/money_tracker_backend/internal/security"
	"github.com/niranjannayak5754/money_tracker_backend/internal/server/handlers"
)

// RegisterRoutes
func RegisterRoutes(r *chi.Mux, c *Container) {
	// Public health check
	r.Get("/health", Health)
	r.Get("/docs", SwaggerUI)
	r.Get("/docs/json", OpenAPIJSON)
	r.Get("/openapi.yaml", c.OpenAPI)

	// Instantiate handlers
	userH := handlers.NewUserHandlers(c.Users, c.Cfg)
	catH := handlers.NewCategoryHandlers(c.Categories)
	incH := handlers.NewIncomeHandlers(c.Income)
	expH := handlers.NewExpenseHandlers(c.Expenses)

	// Auth routes
	r.Route("/auth", func(r chi.Router) {
		r.Post("/register", userH.Register)
		r.Post("/login", userH.Login)
		r.With(c.authMw).Get("/me", userH.Me)
	})

	// Private routes
	r.Group(func(priv chi.Router) {
		priv.Use(c.authMw)

		priv.Mount("/categories", catH.Routes())
		priv.Mount("/income", incH.Routes())
		priv.Mount("/expenses", expH.Routes())

		priv.Get("/summary", c.Summary)
	})
}

// Auth Middleware
func (c *Container) authMw(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		h := r.Header.Get("Authorization")
		if !strings.HasPrefix(h, "Bearer ") {
			httpx.Unauthorized(w, "missing token")
			return
		}

		token := strings.TrimPrefix(h, "Bearer ")

		var claims security.Claims
		parsed, err := security.Parse(c.Cfg.JWTSecret, token, &claims)
		if err != nil || !parsed.Valid {
			httpx.Unauthorized(w, "invalid token")
			return
		}

		// Inject UID into context
		r = r.WithContext(contextutils.WithUID(r.Context(), claims.UID))
		next.ServeHTTP(w, r)
	})
}
