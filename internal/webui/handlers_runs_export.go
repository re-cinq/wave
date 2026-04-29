package webui

import (
	"encoding/csv"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/recinq/wave/internal/state"
)

// handleExportRuns handles GET /api/runs/export - exports runs as CSV or JSON.
func (s *Server) handleExportRuns(w http.ResponseWriter, r *http.Request) {
	format := r.URL.Query().Get("format")
	if format != "csv" && format != "json" {
		writeJSONError(w, http.StatusBadRequest, "format must be 'csv' or 'json'")
		return
	}

	// Respect any active filters
	status := r.URL.Query().Get("status")
	pipeline := r.URL.Query().Get("pipeline")
	opts := state.ListRunsOptions{
		Status:       status,
		PipelineName: pipeline,
		Limit:        10000, // reasonable upper bound for export
	}

	runs, err := s.runtime.store.ListRuns(opts)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to list runs")
		return
	}

	switch format {
	case "csv":
		s.exportRunsCSV(w, runs)
	case "json":
		s.exportRunsJSON(w, runs)
	}
}

// exportRunsCSV writes runs as a CSV download.
func (s *Server) exportRunsCSV(w http.ResponseWriter, runs []state.RunRecord) {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=wave-runs.csv")

	writer := csv.NewWriter(w)
	defer writer.Flush()

	// Write header
	_ = writer.Write([]string{"run_id", "pipeline", "status", "started_at", "duration_seconds", "tokens", "branch"})

	for _, run := range runs {
		var durationSec string
		if run.CompletedAt != nil {
			dur := run.CompletedAt.Sub(run.StartedAt)
			durationSec = strconv.FormatFloat(dur.Seconds(), 'f', 1, 64)
		}
		_ = writer.Write([]string{
			run.RunID,
			run.PipelineName,
			run.Status,
			run.StartedAt.Format(time.RFC3339),
			durationSec,
			strconv.Itoa(run.TotalTokens),
			run.BranchName,
		})
	}
}

// runExportEntry is the JSON structure for a single exported run.
type runExportEntry struct {
	RunID           string   `json:"run_id"`
	Pipeline        string   `json:"pipeline"`
	Status          string   `json:"status"`
	StartedAt       string   `json:"started_at"`
	DurationSeconds *float64 `json:"duration_seconds,omitempty"`
	Tokens          int      `json:"tokens"`
	Branch          string   `json:"branch,omitempty"`
}

// exportRunsJSON writes runs as a JSON array download.
func (s *Server) exportRunsJSON(w http.ResponseWriter, runs []state.RunRecord) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=wave-runs.json")

	entries := make([]runExportEntry, len(runs))
	for i, run := range runs {
		entry := runExportEntry{
			RunID:     run.RunID,
			Pipeline:  run.PipelineName,
			Status:    run.Status,
			StartedAt: run.StartedAt.Format(time.RFC3339),
			Tokens:    run.TotalTokens,
			Branch:    run.BranchName,
		}
		if run.CompletedAt != nil {
			dur := run.CompletedAt.Sub(run.StartedAt).Seconds()
			entry.DurationSeconds = &dur
		}
		entries[i] = entry
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(entries)
}
