package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/niranjannayak5754/money_tracker_backend/internal/config"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/shared"
	"github.com/niranjannayak5754/money_tracker_backend/internal/logging"
	"github.com/niranjannayak5754/money_tracker_backend/internal/platform/mongo"
	"github.com/niranjannayak5754/money_tracker_backend/internal/server"
)

func main() {
	logger := logging.New()

	cfg, err := config.Load()
	if err != nil {
		logger.Error("config load failed", "err", err)
		os.Exit(1)
	}

	shared.SetAppTimezone(cfg.AppTimezone)

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	mc, err := mongo.Connect(ctx, cfg.MongoURI)
	if err != nil {
		logger.Error("mongo connection failed", "err", err)
		os.Exit(1)
	}
	defer mc.Disconnect(context.Background())

	container := server.BuildContainer(cfg, mc, logger)

	srv := server.New(cfg, container, logger)

	if err := srv.Run(ctx); err != nil {
		logger.Error("server stopped with error", "err", err)
	}
}
