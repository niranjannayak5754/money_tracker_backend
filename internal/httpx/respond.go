package httpx

import (
	"encoding/json"
	"net/http"
)

// ErrorResponse standardizes all error outputs.
type ErrorResponse struct {
	Message string `json:"message"`
}

// JSON serializes response data and sets HTTP status code.
func JSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)

	// Safe encoding
	if err := json.NewEncoder(w).Encode(v); err != nil {
		// Fallback: avoid recursion and double writes
		http.Error(w, `{"message":"internal json error"}`, http.StatusInternalServerError)
	}
}

//
// Common response helpers
//

func OK(w http.ResponseWriter) {
	JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func Created(w http.ResponseWriter, v any) {
	JSON(w, http.StatusCreated, v)
}

func BadReq(w http.ResponseWriter, message string) {
	JSON(w, http.StatusBadRequest, ErrorResponse{Message: message})
}

func Unauthorized(w http.ResponseWriter, message string) {
	JSON(w, http.StatusUnauthorized, ErrorResponse{Message: message})
}

func Forbidden(w http.ResponseWriter, message string) {
	JSON(w, http.StatusForbidden, ErrorResponse{Message: message})
}

func NotFound(w http.ResponseWriter) {
	JSON(w, http.StatusNotFound, ErrorResponse{Message: "not found"})
}

// ServerErr returns sanitized internal error message
// (NEVER leak raw error messages in production)
func ServerErr(w http.ResponseWriter, err error) {
	JSON(w, http.StatusInternalServerError, ErrorResponse{
		Message: "internal server error",
	})
}
