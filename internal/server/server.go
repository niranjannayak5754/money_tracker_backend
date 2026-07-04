package server

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"github.com/niranjannayak5754/money_tracker_backend/internal/config"
	"github.com/niranjannayak5754/money_tracker_backend/internal/http/middleware"
	"github.com/niranjannayak5754/money_tracker_backend/internal/scheduler"
)

// recurringCheckInterval is how often the background scheduler checks for
// due recurring templates. Hourly is frequent enough that a missed period
// (e.g. after downtime) catches up within an hour, without polling Mongo
// unnecessarily often for a personal-scale app.
const recurringCheckInterval = time.Hour

type Server struct {
	httpServer      *http.Server
	recurringRunner *scheduler.RecurringRunner
	logger          *slog.Logger
}

// constructor
func New(cfg config.Config, c *Container, logger *slog.Logger) *Server {
	r := chi.NewRouter()

	// global middleware
	r.Use(
		middleware.RequestID,
		chimw.RealIP,
		chimw.Recoverer,
		middleware.SecurityHeaders,
		middleware.BodyLimit,
	)
	r.Use(middleware.CORS(cfg.AllowedOrigins))

	// routes
	RegisterRoutesV2(r, c)

	return &Server{
		httpServer: &http.Server{
			Addr:         cfg.HTTPAddr,
			Handler:      r,
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
		recurringRunner: scheduler.NewRecurringRunner(c.Recurring, c.Expenses, c.Income, c.Investment, logger),
		logger:          logger,
	}
}

func (s *Server) Run(ctx context.Context) error {

	// Start HTTP server in goroutine
	go func() {
		s.logger.Info(
			"http server started",
			"addr", s.httpServer.Addr,
		)

		if err := s.httpServer.ListenAndServe(); err != nil &&
			err != http.ErrServerClosed {

			s.logger.Error(
				"http server error",
				"err", err,
			)
		}
	}()

	// background workers
	go s.recurringRunner.Run(ctx, recurringCheckInterval)

	// Wait for shutdown signal
	<-ctx.Done()

	s.logger.Info("shutdown signal received")

	shutdownCtx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()

	return s.httpServer.Shutdown(shutdownCtx)
}
