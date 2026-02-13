//go:build webui

package webui

import (
	"log"
	"net/http"
	"time"
)

// applyMiddleware wraps the given handler with server middleware.
func (s *Server) applyMiddleware(handler http.Handler) http.Handler {
	h := handler
	h = securityHeaders(h)
	h = s.loggingMiddleware(h)
	if s.token != "" && s.requiresAuth() {
		h = s.authMiddleware(h)
	}
	return h
}

// authMiddleware validates the bearer token for non-static requests.
func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Allow static assets without auth
		if len(r.URL.Path) >= 8 && r.URL.Path[:8] == "/static/" {
			next.ServeHTTP(w, r)
			return
		}

		// Check Authorization header
		auth := r.Header.Get("Authorization")
		if auth == "Bearer "+s.token {
			next.ServeHTTP(w, r)
			return
		}

		// Check query parameter as fallback (for SSE and browser access)
		if r.URL.Query().Get("token") == s.token {
			next.ServeHTTP(w, r)
			return
		}

		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	})
}

// loggingMiddleware logs HTTP requests.
func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(wrapped, r)
		log.Printf("%s %s %d %s", r.Method, r.URL.Path, wrapped.statusCode, time.Since(start))
	})
}

// responseWriter wraps http.ResponseWriter to capture the status code.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
