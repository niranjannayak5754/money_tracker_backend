package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/niranjannayak5754/money_tracker_backend/internal/config"
	"github.com/niranjannayak5754/money_tracker_backend/internal/platform/mongo"
	"github.com/niranjannayak5754/money_tracker_backend/internal/server"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	// Root context with OS signal handling
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	// Mongo connection
	mc, err := mongo.Connect(ctx, cfg.MongoURI)
	if err != nil {
		log.Fatal(err)
	}
	defer mc.Disconnect(context.Background())

	// Build DI container
	container := server.BuildContainer(cfg, mc)

	// HTTP server
	srv := server.New(cfg, container)

	// Run server (blocks until ctx is done)
	if err := srv.Run(ctx); err != nil {
		log.Printf("server stopped with error: %v", err)
	}
}
