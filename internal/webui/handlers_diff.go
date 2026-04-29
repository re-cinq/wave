package webui

import (
	"net/http"
	"path/filepath"
	"strings"
)

// handleAPIDiffSummary handles GET /api/runs/{id}/diff — returns changed files summary.
func (s *Server) handleAPIDiffSummary(w http.ResponseWriter, r *http.Request) {
	runID := r.PathValue("id")
	if runID == "" {
		writeJSONError(w, http.StatusBadRequest, "missing run ID")
		return
	}

	run, err := s.runtime.store.GetRun(runID)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "run not found")
		return
	}

	// FR-006: validate BranchName is populated
	if run.BranchName == "" {
		writeJSON(w, http.StatusOK, &DiffSummary{
			Available: false,
			Message:   "No branch associated with this run",
			Files:     []FileSummary{},
		})
		return
	}

	baseBranch, err := resolveBaseBranch(r.Context(), s.runtime.repoDir)
	if err != nil {
		writeJSON(w, http.StatusOK, &DiffSummary{
			Available: false,
			Message:   err.Error(),
			Files:     []FileSummary{},
		})
		return
	}

	summary := computeDiffSummary(r.Context(), s.runtime.repoDir, baseBranch, run.BranchName)
	writeJSON(w, http.StatusOK, summary)
}

// handleAPIDiffFile handles GET /api/runs/{id}/diff/{path...} — returns single file diff.
func (s *Server) handleAPIDiffFile(w http.ResponseWriter, r *http.Request) {
	runID := r.PathValue("id")
	filePath := r.PathValue("path")

	if runID == "" {
		writeJSONError(w, http.StatusBadRequest, "missing run ID")
		return
	}
	if filePath == "" {
		writeJSONError(w, http.StatusBadRequest, "missing file path")
		return
	}

	// FR-013: sanitize path — reject traversal and absolute paths
	cleanPath := filepath.Clean(filePath)
	if strings.Contains(cleanPath, "..") || strings.HasPrefix(cleanPath, "/") {
		writeJSONError(w, http.StatusBadRequest, "invalid file path")
		return
	}

	run, err := s.runtime.store.GetRun(runID)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "run not found")
		return
	}

	if run.BranchName == "" {
		writeJSONError(w, http.StatusBadRequest, "no branch associated with this run")
		return
	}

	baseBranch, err := resolveBaseBranch(r.Context(), s.runtime.repoDir)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	fileDiff, err := computeFileDiff(r.Context(), s.runtime.repoDir, baseBranch, run.BranchName, cleanPath)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, fileDiff)
}
