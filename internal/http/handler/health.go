package handler

import (
	"net/http"

	"github.com/niranjannayak5754/money_tracker_backend/internal/http/response"
)

func Health(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, 200, map[string]string{
		"status":  "ok",
		"message": "money-tracker api is running",
	})
}
