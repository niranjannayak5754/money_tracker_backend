package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/niranjannayak5754/money_tracker_backend/internal/config"
	"github.com/niranjannayak5754/money_tracker_backend/internal/httpx"
	"github.com/niranjannayak5754/money_tracker_backend/internal/platform/mongo"
	"github.com/niranjannayak5754/money_tracker_backend/internal/server"
)

func main() {
	cfg := config.Load()
	mc, err := mongo.Connect(cfg.MongoURI)
	if err != nil {
		log.Fatal(err)
	}
	defer mc.Disconnect()

	container := server.BuildContainer(cfg, mc)

	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.RealIP, middleware.Recoverer)
	r.Use(httpx.CORSSimple())
	server.RegisterRoutes(r, container)

	log.Printf("moneyflow api listening on %s", cfg.HTTPAddr)
	log.Fatal(http.ListenAndServe(cfg.HTTPAddr, r))
}
