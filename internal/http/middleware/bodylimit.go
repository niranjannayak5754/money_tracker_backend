package middleware

import "net/http"

// maxBodyBytes caps request body size to blunt large-payload DoS attempts,
// including on unauthenticated routes like /auth/login.
const maxBodyBytes = 1 << 20 // 1MB

func BodyLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
		next.ServeHTTP(w, r)
	})
}
