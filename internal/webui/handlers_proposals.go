package webui

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/recinq/wave/internal/proposals"
	"github.com/recinq/wave/internal/state"
)

// proposalListView is the template payload for the proposals list page.
type proposalListView struct {
	ActivePage string
	Filter     string
	Counts     map[string]int
	Proposals  []proposalRow
}

// proposalRow is one entry in the proposals list table.
type proposalRow struct {
	ID            int64
	PipelineName  string
	VersionBefore int
	VersionAfter  int
	Reason        string
	Status        string
	ProposedAt    time.Time
	DecidedAt     *time.Time
	DecidedBy     string
}

// proposalDetailView is the template payload for the proposal detail page.
type proposalDetailView struct {
	ActivePage    string
	Proposal      proposalRow
	DiffLines     []diffLine
	DiffMissing   bool
	DiffError     string
	SignalSummary string
}

// diffLine is one row of a unified diff with a class for line type.
type diffLine struct {
	Class string // diff-line-add | diff-line-del | diff-line-ctx
	Text  string
}

// proposalDecisionResponse is the JSON body returned by approve/reject.
type proposalDecisionResponse struct {
	ID          int64  `json:"id"`
	Status      string `json:"status"`
	NewVersion  int    `json:"new_version,omitempty"`
	NewYAMLPath string `json:"new_yaml_path,omitempty"`
	NewSHA256   string `json:"new_sha256,omitempty"`
	DecidedBy   string `json:"decided_by,omitempty"`
	Activated   bool   `json:"activated,omitempty"`
}

// handleProposalsPage handles GET /proposals.
func (s *Server) handleProposalsPage(w http.ResponseWriter, r *http.Request) {
	store := s.proposalStore()
	if store == nil {
		http.Error(w, "evolution store unavailable", http.StatusInternalServerError)
		return
	}

	filter := strings.TrimSpace(r.URL.Query().Get("status"))
	statuses := []state.EvolutionProposalStatus{
		state.ProposalProposed,
		state.ProposalApproved,
		state.ProposalRejected,
		state.ProposalSuperseded,
	}

	counts := map[string]int{}
	for _, st := range statuses {
		recs, err := store.ListProposalsByStatus(st, 0)
		if err != nil {
			http.Error(w, "list proposals: "+err.Error(), http.StatusInternalServerError)
			return
		}
		counts[string(st)] = len(recs)
	}

	target := state.ProposalProposed
	if filter != "" {
		target = state.EvolutionProposalStatus(filter)
	}
	recs, err := store.ListProposalsByStatus(target, 200)
	if err != nil {
		http.Error(w, "list proposals: "+err.Error(), http.StatusInternalServerError)
		return
	}

	rows := make([]proposalRow, 0, len(recs))
	for _, p := range recs {
		rows = append(rows, recordToRow(p))
	}

	view := proposalListView{
		ActivePage: "proposals",
		Filter:     string(target),
		Counts:     counts,
		Proposals:  rows,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl := s.assets.templates["templates/proposals/list.html"]
	if tmpl == nil {
		http.Error(w, "template missing: proposals/list.html", http.StatusInternalServerError)
		return
	}
	if err := tmpl.Execute(w, view); err != nil {
		http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
	}
}

// handleProposalDetailPage handles GET /proposals/{id}.
func (s *Server) handleProposalDetailPage(w http.ResponseWriter, r *http.Request) {
	store := s.proposalStore()
	if store == nil {
		http.Error(w, "evolution store unavailable", http.StatusInternalServerError)
		return
	}

	id, err := parseProposalID(r.PathValue("id"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	rec, err := store.GetProposal(id)
	if err != nil {
		http.Error(w, "get proposal: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if rec == nil {
		http.Error(w, "proposal not found", http.StatusNotFound)
		return
	}

	view := proposalDetailView{
		ActivePage:    "proposals",
		Proposal:      recordToRow(*rec),
		SignalSummary: rec.SignalSummary,
	}

	if rec.DiffPath == "" {
		view.DiffMissing = true
	} else {
		raw, readErr := os.ReadFile(rec.DiffPath)
		if readErr != nil {
			view.DiffError = fmt.Sprintf("diff unavailable: %s", readErr.Error())
		} else {
			view.DiffLines = parseDiffLines(string(raw))
		}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl := s.assets.templates["templates/proposals/detail.html"]
	if tmpl == nil {
		http.Error(w, "template missing: proposals/detail.html", http.StatusInternalServerError)
		return
	}
	if err := tmpl.Execute(w, view); err != nil {
		http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
	}
}

// handleProposalApprove handles POST /proposals/{id}/approve.
func (s *Server) handleProposalApprove(w http.ResponseWriter, r *http.Request) {
	store := s.proposalStore()
	if store == nil {
		writeJSONError(w, http.StatusInternalServerError, "evolution store unavailable")
		return
	}

	id, err := parseProposalID(r.PathValue("id"))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	rec, err := store.GetProposal(id)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "get proposal: "+err.Error())
		return
	}
	if rec == nil {
		writeJSONError(w, http.StatusNotFound, "proposal not found")
		return
	}

	decidedBy := strings.TrimSpace(r.Header.Get("X-Wave-User"))
	if decidedBy == "" {
		decidedBy = "webui"
	}

	result, err := proposals.Approve(store, rec, decidedBy)
	if err != nil {
		switch {
		case errors.Is(err, proposals.ErrAlreadyDecided):
			writeJSONError(w, http.StatusConflict, err.Error())
		case errors.Is(err, proposals.ErrVersionConflict):
			writeJSONError(w, http.StatusConflict, err.Error())
		case errors.Is(err, proposals.ErrAfterYAMLMissing):
			writeJSONError(w, http.StatusUnprocessableEntity, err.Error())
		default:
			writeJSONError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	writeJSON(w, http.StatusOK, proposalDecisionResponse{
		ID:          rec.ID,
		Status:      string(state.ProposalApproved),
		NewVersion:  result.NewVersion,
		NewYAMLPath: result.YAMLPath,
		NewSHA256:   result.SHA256,
		DecidedBy:   decidedBy,
		Activated:   true,
	})
}

// handleProposalReject handles POST /proposals/{id}/reject.
func (s *Server) handleProposalReject(w http.ResponseWriter, r *http.Request) {
	store := s.proposalStore()
	if store == nil {
		writeJSONError(w, http.StatusInternalServerError, "evolution store unavailable")
		return
	}

	id, err := parseProposalID(r.PathValue("id"))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	decidedBy := strings.TrimSpace(r.Header.Get("X-Wave-User"))
	if decidedBy == "" {
		decidedBy = "webui"
	}

	if err := store.DecideProposal(id, state.ProposalRejected, decidedBy); err != nil {
		// Record may already be decided or missing.
		writeJSONError(w, http.StatusConflict, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, proposalDecisionResponse{
		ID:        id,
		Status:    string(state.ProposalRejected),
		DecidedBy: decidedBy,
	})
}

// handleProposalRollback handles POST /pipelines/{pipelineName}/rollback.
// Flips activation to the prior pipeline_version, an emergency rollback
// for an approved evolution that misbehaves in production.
func (s *Server) handleProposalRollback(w http.ResponseWriter, r *http.Request) {
	store := s.proposalStore()
	if store == nil {
		writeJSONError(w, http.StatusInternalServerError, "evolution store unavailable")
		return
	}
	pipelineName := strings.TrimSpace(r.PathValue("pipelineName"))
	if pipelineName == "" {
		writeJSONError(w, http.StatusBadRequest, "missing pipeline name")
		return
	}

	prior, current, err := proposals.PriorVersion(store, pipelineName)
	if err != nil {
		switch {
		case errors.Is(err, proposals.ErrNoActiveVersion):
			writeJSONError(w, http.StatusNotFound, err.Error())
		case errors.Is(err, proposals.ErrNoPriorVersion):
			writeJSONError(w, http.StatusConflict, err.Error())
		default:
			writeJSONError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}
	if err := store.ActivateVersion(pipelineName, prior.Version); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "activate prior: "+err.Error())
		return
	}

	decidedBy := strings.TrimSpace(r.Header.Get("X-Wave-User"))
	if decidedBy == "" {
		decidedBy = "webui"
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"pipeline_name":    pipelineName,
		"rolled_back_from": current.Version,
		"now_active":       prior.Version,
		"yaml_path":        prior.YAMLPath,
		"decided_by":       decidedBy,
	})
}

// proposalStore returns the read-write evolution store, preferring the rwStore
// (which permits writes) over the read-only store. Tests that wire only an
// rwStore still work because both fields hold the same handle in production.
func (s *Server) proposalStore() state.EvolutionStore {
	if s.runtime.rwStore != nil {
		return s.runtime.rwStore
	}
	return s.runtime.store
}

// parseDiffLines converts unified-diff text into class-tagged lines. Empty
// input yields a single context line so the template renders an empty pre.
func parseDiffLines(diff string) []diffLine {
	if diff == "" {
		return nil
	}
	lines := strings.Split(diff, "\n")
	out := make([]diffLine, 0, len(lines))
	for i, ln := range lines {
		// Trim a trailing empty line introduced by the final newline.
		if i == len(lines)-1 && ln == "" {
			continue
		}
		switch {
		case strings.HasPrefix(ln, "+++") || strings.HasPrefix(ln, "---"):
			out = append(out, diffLine{Class: "diff-line-meta", Text: ln})
		case strings.HasPrefix(ln, "@@"):
			out = append(out, diffLine{Class: "diff-line-hunk", Text: ln})
		case strings.HasPrefix(ln, "+"):
			out = append(out, diffLine{Class: "diff-line-add", Text: ln})
		case strings.HasPrefix(ln, "-"):
			out = append(out, diffLine{Class: "diff-line-del", Text: ln})
		default:
			out = append(out, diffLine{Class: "diff-line-ctx", Text: ln})
		}
	}
	return out
}

func parseProposalID(raw string) (int64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, errors.New("missing proposal id")
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return 0, errors.New("invalid proposal id")
	}
	return id, nil
}

func recordToRow(p state.EvolutionProposalRecord) proposalRow {
	return proposalRow{
		ID:            p.ID,
		PipelineName:  p.PipelineName,
		VersionBefore: p.VersionBefore,
		VersionAfter:  p.VersionAfter,
		Reason:        p.Reason,
		Status:        string(p.Status),
		ProposedAt:    p.ProposedAt,
		DecidedAt:     p.DecidedAt,
		DecidedBy:     p.DecidedBy,
	}
}

// proposalStatusBadgeClass picks a badge css class for a proposal status.
// Exposed via a template func so list/detail can render colored pills.
func proposalStatusBadgeClass(status string) string {
	switch status {
	case "proposed":
		return "badge-yellow"
	case "approved":
		return "badge-green"
	case "rejected":
		return "badge-red"
	case "superseded":
		return "badge-gray"
	default:
		return "badge-neutral"
	}
}
