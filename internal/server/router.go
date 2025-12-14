package server

import (
	"github.com/go-chi/chi/v5"

	"github.com/niranjannayak5754/money_tracker_backend/internal/http/handler"
)

// RegisterRoutesV2 registers routes using new http handlers
func RegisterRoutesV2(r chi.Router, c *Container) {
	// health check routes
	r.Get("/health", handler.Health)

	// swagger docs routes
	r.Get("/docs", handler.SwaggerUI)
	r.Get("/docs/json", handler.OpenAPIJSON)
	r.Get("/openapi.yaml", handler.OpenAPI)

	// auth routes
	r.Route("/auth", func(r chi.Router) {
		r.Post("/register", c.AuthH.Register)
		r.Post("/login", c.AuthH.Login)
		r.With(c.AuthMw.RequireAuth).Get("/me", c.AuthH.Me)
		r.With(c.AuthMw.RequireAuth).Post("/logout", c.AuthH.Logout)
	})

	// protected routes
	r.Group(func(priv chi.Router) {
		priv.Use(c.AuthMw.RequireAuth)
		priv.Get("/summary", c.SummaryH.Get)

		priv.Mount("/categories", c.CategoryH.Routes())
		priv.Mount("/income", c.IncomeH.Routes())
		priv.Mount("/expenses", c.ExpenseH.Routes())
	})

}
