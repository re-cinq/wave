//go:build webui

package webui

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
)

// requiresAuth returns true if the server binding requires authentication.
func (s *Server) requiresAuth() bool {
	return s.bind != "127.0.0.1" && s.bind != "localhost" && s.bind != ""
}

// GenerateToken generates a random 32-byte hex token.
func GenerateToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// securityHeaders adds security headers to responses.
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'")
		w.Header().Set("Referrer-Policy", "same-origin")
		next.ServeHTTP(w, r)
	})
}
