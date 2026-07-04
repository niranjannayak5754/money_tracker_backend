package server

import (
	"time"

	"github.com/go-chi/chi/v5"
	"golang.org/x/time/rate"

	"github.com/niranjannayak5754/money_tracker_backend/internal/http/handler"
	"github.com/niranjannayak5754/money_tracker_backend/internal/http/middleware"
)

// RegisterRoutesV2 registers routes using new http handlers
func RegisterRoutesV2(r chi.Router, c *Container) {
	// health check routes
	r.Get("/health", handler.Health)

	// swagger docs routes
	r.Get("/docs", handler.SwaggerUI)
	r.Get("/docs/json", handler.OpenAPIJSON)
	r.Get("/openapi.yaml", handler.OpenAPI)

	// throttles brute-force/credential-stuffing attempts against auth endpoints
	authRateLimit := middleware.RateLimit(rate.Every(2*time.Second), 5)

	// auth routes
	r.Route("/auth", func(r chi.Router) {
		r.With(authRateLimit).Post("/register", c.AuthH.Register)
		r.With(authRateLimit).Post("/login", c.AuthH.Login)
		r.With(authRateLimit).Post("/refresh", c.AuthH.Refresh)
		r.With(c.AuthMw.RequireAuth).Get("/me", c.AuthH.Me)
		r.With(c.AuthMw.RequireAuth).Post("/logout", c.AuthH.Logout)
		r.With(c.AuthMw.RequireAuth).Post("/reset-password", c.AuthH.ResetPassword)
	})

	// protected routes
	r.Group(func(priv chi.Router) {
		priv.Use(c.AuthMw.RequireAuth)
		priv.Mount("/categories", c.CategoryH.Routes())
		priv.Mount("/income", c.IncomeH.Routes())
		priv.Mount("/expenses", c.ExpenseH.Routes())
		priv.Mount("/investment", c.InvestmentH.Routes())
		priv.Mount("/bank-accounts", c.BankAccountH.Routes())
		priv.Mount("/debts", c.DebtH.Routes())
		priv.Mount("/recurring", c.RecurringH.Routes())
		priv.Mount("/budgets", c.BudgetH.Routes())
		priv.Mount("/goals", c.GoalH.Routes())
		priv.Mount("/notifications", c.NotificationH.Routes())
		priv.Get("/summary", c.SummaryH.Get)
		priv.Get("/summary/compare", c.SummaryH.Compare)
		priv.Get("/summary/networth-history", c.SummaryH.NetWorthHistory)
	})

}
