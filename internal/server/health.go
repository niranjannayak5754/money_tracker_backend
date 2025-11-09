package server

import (
	"net/http"

	"github.com/niranjannayak5754/money_tracker_backend/internal/httpx"
)

func Health(w http.ResponseWriter, r *http.Request) {
	httpx.JSON(w, 200, map[string]string{
		"status":  "ok",
		"message": "money-tracker api is running",
	})
}
