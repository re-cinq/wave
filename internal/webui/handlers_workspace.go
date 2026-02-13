//go:build webui

package webui

import (
	"html"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const maxWorkspaceEntries = 500
const maxWorkspaceFileSize = 512 * 1024 // 512 KB

// handleWorkspaceTree handles GET /api/runs/{id}/workspace/{step}/tree?path=
func (s *Server) handleWorkspaceTree(w http.ResponseWriter, r *http.Request) {
	runID := r.PathValue("id")
	stepID := r.PathValue("step")
	reqPath := r.URL.Query().Get("path")

	if runID == "" || stepID == "" {
		writeJSONError(w, http.StatusBadRequest, "missing run ID or step ID")
		return
	}

	// Resolve workspace root path
	wsRoot := resolveWorkspacePath(runID, stepID)
	if wsRoot == "" {
		writeJSON(w, http.StatusOK, WorkspaceTreeResponse{
			Path:  reqPath,
			Error: "workspace not found",
		})
		return
	}

	// Resolve and validate the requested path
	targetPath := filepath.Join(wsRoot, filepath.Clean("/"+reqPath))
	if !strings.HasPrefix(targetPath, wsRoot) {
		writeJSON(w, http.StatusOK, WorkspaceTreeResponse{
			Path:  reqPath,
			Error: "path traversal detected",
		})
		return
	}

	info, err := os.Stat(targetPath)
	if err != nil || !info.IsDir() {
		writeJSON(w, http.StatusOK, WorkspaceTreeResponse{
			Path:  reqPath,
			Error: "directory not found",
		})
		return
	}

	entries, err := os.ReadDir(targetPath)
	if err != nil {
		writeJSON(w, http.StatusOK, WorkspaceTreeResponse{
			Path:  reqPath,
			Error: "failed to read directory",
		})
		return
	}

	// Sort: directories first, then files
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].IsDir() != entries[j].IsDir() {
			return entries[i].IsDir()
		}
		return entries[i].Name() < entries[j].Name()
	})

	wsEntries := make([]WorkspaceEntry, 0, len(entries))
	for i, e := range entries {
		if i >= maxWorkspaceEntries {
			break
		}
		// Skip hidden files
		if strings.HasPrefix(e.Name(), ".") {
			continue
		}
		we := WorkspaceEntry{
			Name:  e.Name(),
			IsDir: e.IsDir(),
		}
		if info, err := e.Info(); err == nil {
			we.Size = info.Size()
		}
		if !e.IsDir() {
			we.Extension = strings.TrimPrefix(filepath.Ext(e.Name()), ".")
		}
		wsEntries = append(wsEntries, we)
	}

	writeJSON(w, http.StatusOK, WorkspaceTreeResponse{
		Path:    reqPath,
		Entries: wsEntries,
	})
}

// handleWorkspaceFile handles GET /api/runs/{id}/workspace/{step}/file?path=
func (s *Server) handleWorkspaceFile(w http.ResponseWriter, r *http.Request) {
	runID := r.PathValue("id")
	stepID := r.PathValue("step")
	reqPath := r.URL.Query().Get("path")

	if runID == "" || stepID == "" || reqPath == "" {
		writeJSONError(w, http.StatusBadRequest, "missing parameters")
		return
	}

	// Resolve workspace root path
	wsRoot := resolveWorkspacePath(runID, stepID)
	if wsRoot == "" {
		writeJSON(w, http.StatusOK, WorkspaceFileResponse{
			Path:  reqPath,
			Error: "workspace not found",
		})
		return
	}

	// Resolve and validate the requested path
	targetPath := filepath.Join(wsRoot, filepath.Clean("/"+reqPath))
	if !strings.HasPrefix(targetPath, wsRoot) {
		writeJSON(w, http.StatusOK, WorkspaceFileResponse{
			Path:  reqPath,
			Error: "path traversal detected",
		})
		return
	}

	info, err := os.Stat(targetPath)
	if err != nil {
		writeJSON(w, http.StatusOK, WorkspaceFileResponse{
			Path:  reqPath,
			Error: "file not found",
		})
		return
	}

	if info.IsDir() {
		writeJSON(w, http.StatusOK, WorkspaceFileResponse{
			Path:  reqPath,
			Error: "path is a directory, not a file",
		})
		return
	}

	content, err := os.ReadFile(targetPath)
	if err != nil {
		writeJSON(w, http.StatusOK, WorkspaceFileResponse{
			Path:  reqPath,
			Error: "failed to read file",
		})
		return
	}

	truncated := false
	if len(content) > maxWorkspaceFileSize {
		content = content[:maxWorkspaceFileSize]
		truncated = true
	}

	// Redact credentials and escape HTML
	redacted := RedactCredentials(string(content))
	escaped := html.EscapeString(redacted)

	mimeType := detectMimeType(filepath.Base(reqPath))

	writeJSON(w, http.StatusOK, WorkspaceFileResponse{
		Path:      reqPath,
		Content:   escaped,
		MimeType:  mimeType,
		Size:      info.Size(),
		Truncated: truncated,
	})
}

// resolveWorkspacePath tries to find the workspace directory for a run+step.
func resolveWorkspacePath(runID, stepID string) string {
	// Try common workspace path patterns
	candidates := []string{
		filepath.Join(".wave", "workspaces", runID, stepID),
		filepath.Join(".wave", "workspaces", runID),
	}

	for _, c := range candidates {
		if info, err := os.Stat(c); err == nil && info.IsDir() {
			abs, _ := filepath.Abs(c)
			return abs
		}
	}

	return ""
}
