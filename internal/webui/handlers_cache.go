package webui

import (
	"log"
	"net/http"
)

// handleAPICacheRefresh handles POST /api/cache/refresh — clears all cached API responses.
func (s *Server) handleAPICacheRefresh(w http.ResponseWriter, r *http.Request) {
	s.cache.Clear()
	log.Printf("[webui] cache cleared via API request")
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "message": "cache cleared"})
}
