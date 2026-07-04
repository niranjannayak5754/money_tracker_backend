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
)

type Server struct {
	httpServer *http.Server
	logger     *slog.Logger
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
		logger: logger,
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

	// FUTURE: background workers
	// go s.runCronJobs(ctx)
	// go s.runAsyncConsumers(ctx)

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
