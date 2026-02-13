//go:build webui

package webui

import (
	"html"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const maxArtifactSize = 1024 * 1024 // 1 MB

// handleArtifact handles GET /api/runs/{id}/artifacts/{step}/{name}
func (s *Server) handleArtifact(w http.ResponseWriter, r *http.Request) {
	runID := r.PathValue("id")
	stepID := r.PathValue("step")
	name := r.PathValue("name")

	if runID == "" || stepID == "" || name == "" {
		http.Error(w, "missing parameters", http.StatusBadRequest)
		return
	}

	// Find the artifact in the database
	artifacts, err := s.store.GetArtifacts(runID, stepID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to get artifacts")
		return
	}

	var found *ArtifactSummary
	for _, a := range artifacts {
		if a.Name == name {
			as := artifactToSummary(a)
			found = &as
			break
		}
	}

	if found == nil {
		writeJSONError(w, http.StatusNotFound, "artifact not found")
		return
	}

	// Path traversal prevention
	cleanPath := filepath.Clean(found.Path)
	if strings.Contains(cleanPath, "..") {
		writeJSONError(w, http.StatusForbidden, "path traversal detected")
		return
	}

	// Read file content
	content, err := os.ReadFile(cleanPath)
	if err != nil {
		if os.IsNotExist(err) {
			writeJSONError(w, http.StatusNotFound, "artifact file not found (workspace may have been cleaned up)")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "failed to read artifact")
		return
	}

	// Check for raw download
	if r.URL.Query().Get("raw") == "true" {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("Content-Disposition", "attachment; filename=\""+name+"\"")
		w.Write(content)
		return
	}

	// Truncate if too large
	truncated := false
	if len(content) > maxArtifactSize {
		content = content[:maxArtifactSize]
		truncated = true
	}

	// Redact credentials and escape HTML
	redacted := RedactCredentials(string(content))
	escaped := html.EscapeString(redacted)

	// Determine MIME type
	mimeType := detectMimeType(name)

	resp := ArtifactContentResponse{
		Content: escaped,
		Metadata: ArtifactMetadata{
			Name:      name,
			Type:      found.Type,
			SizeBytes: found.SizeBytes,
			Truncated: truncated,
			MimeType:  mimeType,
		},
	}

	writeJSON(w, http.StatusOK, resp)
}

// detectMimeType guesses the MIME type from the file extension.
func detectMimeType(name string) string {
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".json":
		return "application/json"
	case ".yaml", ".yml":
		return "application/x-yaml"
	case ".md":
		return "text/markdown"
	case ".go":
		return "text/x-go"
	case ".py":
		return "text/x-python"
	case ".js":
		return "text/javascript"
	case ".ts":
		return "text/typescript"
	case ".html":
		return "text/html"
	case ".css":
		return "text/css"
	case ".txt":
		return "text/plain"
	case ".log":
		return "text/plain"
	default:
		return "application/octet-stream"
	}
}
