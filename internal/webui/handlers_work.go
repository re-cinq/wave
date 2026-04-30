package webui

import (
	"context"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/recinq/wave/internal/forge"
	"github.com/recinq/wave/internal/state"
	"github.com/recinq/wave/internal/worksource"
)

// WorkBindingRow is the flat view-model rendered by the /work board for a
// single worksource binding. It deliberately decouples the template from
// worksource.BindingRecord so trigger/label maps can be precomputed without
// pushing template logic into the package.
type WorkBindingRow struct {
	ID           int64
	Forge        string
	RepoPattern  string
	PipelineName string
	Trigger      worksource.Trigger
	TriggerLabel string
	Active       bool
	StatusLabel  string
	LabelFilter  []string
	State        string
	Kinds        []string
	CreatedAt    time.Time
}

// WorkBoardData backs templates/work/board.html.
type WorkBoardData struct {
	ActivePage  string
	Bindings    []WorkBindingRow
	RecentRuns  []RunSummary
	HasBindings bool
}

// WorkItemDetailData backs templates/work/detail.html.
type WorkItemDetailData struct {
	ActivePage      string
	Forge           string
	Owner           string
	Repo            string
	RepoSlug        string
	Number          int
	NumberStr       string
	Kind            string // "issue" — work items here always come from the issue path. PRs use /work/.../pulls/... in a future iteration.
	Title           string
	Body            string
	State           string
	Author          string
	Labels          []string
	URL             string
	ItemAvailable   bool
	ForgeUnavailable bool
	MatchedBindings []WorkBindingRow
	RecentRuns      []RunSummary
}

// handleWorkBoard serves GET /work — the unified board listing all
// worksource bindings plus a best-effort "recent matches" list of recent
// pipeline runs whose pipeline name belongs to a binding.
func (s *Server) handleWorkBoard(w http.ResponseWriter, r *http.Request) {
	tmpl, ok := s.assets.templates["templates/work/board.html"]
	if !ok || tmpl == nil {
		http.Error(w, "work board template missing", http.StatusInternalServerError)
		return
	}

	ctx := r.Context()
	var rows []WorkBindingRow
	if s.runtime.worksource != nil {
		records, err := s.runtime.worksource.ListBindings(ctx, worksource.BindingFilter{})
		if err != nil {
			log.Printf("[webui] /work list bindings: %v", err)
			http.Error(w, "failed to list bindings", http.StatusInternalServerError)
			return
		}
		rows = make([]WorkBindingRow, 0, len(records))
		for _, rec := range records {
			rows = append(rows, bindingRecordToRow(rec))
		}
	}

	pipelineNames := bindingPipelineSet(rows)
	recent := s.recentRunsForPipelines(pipelineNames, 20)

	data := WorkBoardData{
		ActivePage:  "work",
		Bindings:    rows,
		RecentRuns:  recent,
		HasBindings: len(rows) > 0,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.Execute(w, data); err != nil {
		log.Printf("[webui] /work render: %v", err)
		http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
	}
}

// handleWorkItemDetail serves GET /work/{forge}/{owner}/{repo}/{number}.
// It parses path values, builds a worksource.WorkItemRef, queries
// MatchBindings, optionally fetches the live work item from the forge
// client when available, and renders templates/work/detail.html.
func (s *Server) handleWorkItemDetail(w http.ResponseWriter, r *http.Request) {
	tmpl, ok := s.assets.templates["templates/work/detail.html"]
	if !ok || tmpl == nil {
		http.Error(w, "work detail template missing", http.StatusInternalServerError)
		return
	}

	forgeName := r.PathValue("forge")
	owner := r.PathValue("owner")
	repo := r.PathValue("repo")
	numberStr := r.PathValue("number")

	number, err := strconv.Atoi(numberStr)
	if err != nil || number <= 0 {
		http.Error(w, "invalid work item number", http.StatusBadRequest)
		return
	}
	if forgeName == "" || owner == "" || repo == "" {
		http.Error(w, "incomplete work item path", http.StatusBadRequest)
		return
	}

	repoSlug := owner + "/" + repo
	ref := worksource.WorkItemRef{
		Forge: forgeName,
		Repo:  repoSlug,
		Kind:  "issue",
		ID:    numberStr,
	}

	ctx := r.Context()
	var matched []WorkBindingRow
	if s.runtime.worksource != nil {
		recs, err := s.runtime.worksource.MatchBindings(ctx, ref)
		if err != nil {
			log.Printf("[webui] /work detail match: %v", err)
			http.Error(w, "failed to match bindings", http.StatusInternalServerError)
			return
		}
		matched = make([]WorkBindingRow, 0, len(recs))
		for _, rec := range recs {
			matched = append(matched, bindingRecordToRow(rec))
		}
	}

	data := WorkItemDetailData{
		ActivePage:       "work",
		Forge:            forgeName,
		Owner:            owner,
		Repo:             repo,
		RepoSlug:         repoSlug,
		Number:           number,
		NumberStr:        numberStr,
		Kind:             "issue",
		MatchedBindings:  matched,
		ForgeUnavailable: s.runtime.forgeClient == nil,
	}

	if s.runtime.forgeClient != nil {
		issue, err := fetchIssueBestEffort(ctx, s.runtime.forgeClient, owner, repo, number)
		if err == nil && issue != nil {
			data.Title = issue.Title
			data.Body = issue.Body
			data.State = issue.State
			data.Author = issue.Author
			data.URL = issue.HTMLURL
			data.Labels = labelNames(issue.Labels)
			data.ItemAvailable = true
			// Re-run match with full label set so label-gated bindings show up.
			if s.runtime.worksource != nil && len(data.Labels) > 0 {
				ref.Labels = data.Labels
				ref.State = issue.State
				ref.Title = issue.Title
				ref.URL = issue.HTMLURL
				if recs, mErr := s.runtime.worksource.MatchBindings(ctx, ref); mErr == nil {
					rich := make([]WorkBindingRow, 0, len(recs))
					for _, rec := range recs {
						rich = append(rich, bindingRecordToRow(rec))
					}
					data.MatchedBindings = rich
				}
			}
		}
	}

	pipelineNames := bindingPipelineSet(data.MatchedBindings)
	data.RecentRuns = s.recentRunsForPipelines(pipelineNames, 20)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.Execute(w, data); err != nil {
		log.Printf("[webui] /work detail render: %v", err)
		http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
	}
}

// bindingRecordToRow flattens a worksource.BindingRecord for template
// consumption. Labels for trigger/active are computed once here so the
// templates remain logic-light.
func bindingRecordToRow(rec worksource.BindingRecord) WorkBindingRow {
	status := "inactive"
	if rec.Active {
		status = "active"
	}
	return WorkBindingRow{
		ID:           int64(rec.ID),
		Forge:        rec.Forge,
		RepoPattern:  rec.RepoPattern,
		PipelineName: rec.PipelineName,
		Trigger:      rec.Trigger,
		TriggerLabel: triggerDisplay(rec.Trigger),
		Active:       rec.Active,
		StatusLabel:  status,
		LabelFilter:  rec.LabelFilter,
		State:        rec.State,
		Kinds:        rec.Kinds,
		CreatedAt:    rec.CreatedAt,
	}
}

// triggerDisplay returns a short human label for the binding trigger enum.
func triggerDisplay(t worksource.Trigger) string {
	switch t {
	case worksource.TriggerOnDemand:
		return "On demand"
	case worksource.TriggerOnLabel:
		return "On label"
	case worksource.TriggerOnOpen:
		return "On open"
	case worksource.TriggerScheduled:
		return "Scheduled"
	default:
		return string(t)
	}
}

// bindingPipelineSet returns the unique pipeline names referenced by the
// given binding rows.
func bindingPipelineSet(rows []WorkBindingRow) map[string]struct{} {
	out := make(map[string]struct{}, len(rows))
	for _, r := range rows {
		if r.PipelineName != "" {
			out[r.PipelineName] = struct{}{}
		}
	}
	return out
}

// recentRunsForPipelines pulls up to limit recent top-level runs whose
// pipeline name appears in want. Returns nil if want is empty or the store
// is missing. This is the placeholder run-history surface until #2.4 wires
// real work_item_ref → run linkage.
func (s *Server) recentRunsForPipelines(want map[string]struct{}, limit int) []RunSummary {
	if len(want) == 0 || s.runtime.store == nil || limit <= 0 {
		return nil
	}
	// Pull a generous slice and filter in-memory; the binding pipeline set is
	// typically tiny so a single-pass scan is cheaper than per-pipeline queries.
	pageSize := limit * 4
	if pageSize < 50 {
		pageSize = 50
	}
	runs, err := s.runtime.store.ListRuns(state.ListRunsOptions{
		Limit:        pageSize,
		TopLevelOnly: true,
	})
	if err != nil {
		log.Printf("[webui] /work recent runs: %v", err)
		return nil
	}
	out := make([]RunSummary, 0, limit)
	for _, run := range runs {
		if _, ok := want[run.PipelineName]; !ok {
			continue
		}
		out = append(out, runToSummary(run))
		if len(out) >= limit {
			break
		}
	}
	return out
}

// fetchIssueBestEffort returns the issue or nil. It applies a short timeout
// so a slow forge does not hold the dashboard render. Errors are logged at
// the call site; here we return them so callers can detect "unavailable".
func fetchIssueBestEffort(ctx context.Context, client forge.Client, owner, repo string, number int) (*forge.Issue, error) {
	if client == nil {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	issue, err := client.GetIssue(ctx, owner, repo, number)
	if err != nil {
		log.Printf("[webui] /work detail forge fetch %s/%s#%d: %v", owner, repo, number, err)
		return nil, err
	}
	return issue, nil
}

// labelNames extracts the Name strings from a forge label slice.
func labelNames(labels []forge.Label) []string {
	if len(labels) == 0 {
		return nil
	}
	out := make([]string, 0, len(labels))
	for _, l := range labels {
		if l.Name != "" {
			out = append(out, l.Name)
		}
	}
	return out
}
