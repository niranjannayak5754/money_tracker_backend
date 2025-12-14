package server

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"github.com/niranjannayak5754/money_tracker_backend/internal/config"
	"github.com/niranjannayak5754/money_tracker_backend/internal/http/middleware"
)

type Server struct {
	httpServer *http.Server
}

// constructor
func New(cfg config.Config, c *Container) *Server {
	r := chi.NewRouter()

	// global middleware
	r.Use(
		chimw.RequestID,
		chimw.RealIP,
		chimw.Recoverer,
	)
	r.Use(middleware.CORSSimple())

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
	}
}

func (s *Server) Run(ctx context.Context) error {

	// Start HTTP server in goroutine
	go func() {
		log.Printf("HTTP server listening on %s", s.httpServer.Addr)

		if err := s.httpServer.ListenAndServe(); err != nil &&
			err != http.ErrServerClosed {
			log.Fatalf("http server error: %v", err)
		}
	}()

	// FUTURE: to add background workers
	// go s.runCronJobs(ctx)
	// go s.runAsyncConsumers(ctx)

	// Wait for shutdown signal
	<-ctx.Done()
	log.Println("shutdown signal received")

	shutdownCtx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()

	return s.httpServer.Shutdown(shutdownCtx)
}
