package webui

import (
	"log"
	"net/http"
	"strings"
	"time"
)

// applyMiddleware wraps the given handler with server middleware.
func (s *Server) applyMiddleware(handler http.Handler) http.Handler {
	h := handler
	if s.csrfToken != "" {
		h = s.csrfMiddleware(h)
	}
	h = securityHeaders(h)
	h = s.loggingMiddleware(h)

	// Apply auth middleware based on resolved auth mode
	switch s.authMode {
	case AuthModeBearer:
		if s.token != "" && s.requiresAuth() {
			h = s.bearerAuthMiddleware(h)
		}
	case AuthModeJWT:
		h = s.jwtAuthMiddleware(h)
	case AuthModeMTLS:
		// mTLS is handled at the TLS layer — no HTTP middleware needed
	case AuthModeNone:
		// No auth
	default:
		// Backward compatibility: use bearer auth if token is set
		if s.token != "" && s.requiresAuth() {
			h = s.bearerAuthMiddleware(h)
		}
	}

	return h
}

// authMiddleware validates the bearer token for non-static requests.
// Preserved for backward compatibility.
func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return s.bearerAuthMiddleware(next)
}

// bearerAuthMiddleware validates a bearer token for non-static requests.
func (s *Server) bearerAuthMiddleware(next http.Handler) http.Handler {
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

// jwtAuthMiddleware validates JWT tokens for non-static requests.
func (s *Server) jwtAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Allow static assets without auth
		if len(r.URL.Path) >= 8 && r.URL.Path[:8] == "/static/" {
			next.ServeHTTP(w, r)
			return
		}

		// Extract token from Authorization header
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			// Fallback: check query parameter (for SSE)
			tokenStr := r.URL.Query().Get("token")
			if tokenStr == "" {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			auth = "Bearer " + tokenStr
		}

		tokenStr := strings.TrimPrefix(auth, "Bearer ")
		if _, err := ValidateJWT(tokenStr, s.jwtSecret); err != nil {
			http.Error(w, "Unauthorized: "+err.Error(), http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// csrfMiddleware validates the X-CSRF-Token header on mutation requests
// (POST, PUT, DELETE, PATCH). GET/HEAD/OPTIONS are allowed through.
func (s *Server) csrfMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch:
			token := r.Header.Get("X-CSRF-Token")
			if token == "" || token != s.csrfToken {
				http.Error(w, "Forbidden: invalid CSRF token", http.StatusForbidden)
				return
			}
		}
		next.ServeHTTP(w, r)
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

// Flush delegates to the underlying ResponseWriter if it supports flushing.
// Required for SSE (Server-Sent Events) streaming through the logging middleware.
func (rw *responseWriter) Flush() {
	if f, ok := rw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}
