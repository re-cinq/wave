package webui

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/recinq/wave/internal/forge"
	"github.com/recinq/wave/internal/timeouts"
	"github.com/recinq/wave/internal/worksource"
)

// handleWorkDispatch handles POST /work/{forge}/{owner}/{repo}/{number}/dispatch.
//
// It resolves matching worksource bindings for the work item, picks a pipeline,
// serializes a shared `work_item_ref` schema document as the run input, and
// launches the pipeline through the existing s.launchPipelineExecution path.
//
// Response codes:
//   - 302 Found → /runs/{runID} on a successful launch.
//   - 400 Bad Request when the path number is malformed, when multiple bindings
//     match and no `pipeline` form/query param disambiguates, or when the
//     supplied `pipeline` value is not in the match set.
//   - 409 Conflict when no active binding matches the work item.
func (s *Server) handleWorkDispatch(w http.ResponseWriter, r *http.Request) {
	if s.runtime.worksource == nil {
		http.Error(w, "worksource service not configured", http.StatusServiceUnavailable)
		return
	}

	forgeName := strings.ToLower(r.PathValue("forge"))
	owner := r.PathValue("owner")
	repo := r.PathValue("repo")
	numberStr := r.PathValue("number")
	if forgeName == "" || owner == "" || repo == "" || numberStr == "" {
		http.Error(w, "missing path parameter", http.StatusBadRequest)
		return
	}
	number, err := strconv.Atoi(numberStr)
	if err != nil || number <= 0 {
		http.Error(w, "invalid issue number", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), timeouts.ForgeAPI)
	defer cancel()

	ref := s.buildWorkItemRef(ctx, forgeName, owner, repo, number)

	matches, err := s.runtime.worksource.MatchBindings(ctx, ref)
	if err != nil {
		http.Error(w, "match bindings: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form body", http.StatusBadRequest)
		return
	}
	requested := strings.TrimSpace(r.FormValue("pipeline"))

	pipelineName, status, msg := selectBindingPipeline(matches, requested)
	if status != http.StatusOK {
		http.Error(w, msg, status)
		return
	}

	hostOverride := ""
	if s.runtime.forgeClient != nil && string(s.runtime.forgeClient.ForgeType()) == forgeName {
		hostOverride = s.runtime.forgeHost
	}
	input, err := serializeWorkItemRef(ref, hostOverride)
	if err != nil {
		http.Error(w, "serialize work_item_ref: "+err.Error(), http.StatusInternalServerError)
		return
	}

	runID, err := s.runtime.rwStore.CreateRun(pipelineName, input)
	if err != nil {
		http.Error(w, "create run: "+err.Error(), http.StatusInternalServerError)
		return
	}

	s.launchPipelineExecution(runID, pipelineName, input, RunOptions{})

	http.Redirect(w, r, "/runs/"+runID, http.StatusFound)
}

// buildWorkItemRef constructs a worksource.WorkItemRef for the given path
// coordinates. When a configured forge client matches the path forge, it
// enriches the ref with title/state/labels/url from the live issue. Otherwise
// the ref carries only the structural fields, which is enough for repo- and
// forge-pattern matches but not for label filters.
func (s *Server) buildWorkItemRef(ctx context.Context, forgeName, owner, repo string, number int) worksource.WorkItemRef {
	ref := worksource.WorkItemRef{
		Forge: forgeName,
		Repo:  owner + "/" + repo,
		Kind:  "issue",
		ID:    strconv.Itoa(number),
		State: "open",
	}

	if s.runtime.forgeClient == nil {
		return ref
	}
	if string(s.runtime.forgeClient.ForgeType()) != forgeName {
		return ref
	}

	issue, err := s.runtime.forgeClient.GetIssue(ctx, owner, repo, number)
	if err != nil || issue == nil {
		return ref
	}

	ref.Title = issue.Title
	ref.URL = issue.HTMLURL
	if issue.State != "" {
		ref.State = issue.State
	}
	if len(issue.Labels) > 0 {
		ref.Labels = make([]string, 0, len(issue.Labels))
		for _, l := range issue.Labels {
			ref.Labels = append(ref.Labels, l.Name)
		}
	}
	return ref
}

// selectBindingPipeline picks the pipeline name to launch from the match set
// and an optionally-supplied disambiguation parameter. Returns the chosen
// pipeline, an HTTP status (200 on success), and an error message for the
// non-success path. Behaviour:
//   - 0 matches → 409 Conflict.
//   - 1 match  → that binding's pipeline.
//   - >1 match, no requested → 400 listing the available pipelines.
//   - >1 match, requested in set → that pipeline.
//   - >1 match, requested not in set → 400 listing the available pipelines.
func selectBindingPipeline(matches []worksource.BindingRecord, requested string) (string, int, string) {
	switch len(matches) {
	case 0:
		return "", http.StatusConflict, "no worksource binding matches this work item"
	case 1:
		if requested != "" && requested != matches[0].PipelineName {
			return "", http.StatusBadRequest, fmt.Sprintf("requested pipeline %q does not match binding pipeline %q", requested, matches[0].PipelineName)
		}
		return matches[0].PipelineName, http.StatusOK, ""
	}

	allowed := make([]string, 0, len(matches))
	seen := make(map[string]struct{}, len(matches))
	for _, m := range matches {
		if _, ok := seen[m.PipelineName]; ok {
			continue
		}
		seen[m.PipelineName] = struct{}{}
		allowed = append(allowed, m.PipelineName)
	}
	sort.Strings(allowed)

	if requested == "" {
		return "", http.StatusBadRequest, fmt.Sprintf("multiple bindings match; specify pipeline (one of: %s)", strings.Join(allowed, ", "))
	}
	for _, p := range allowed {
		if p == requested {
			return requested, http.StatusOK, ""
		}
	}
	return "", http.StatusBadRequest, fmt.Sprintf("requested pipeline %q is not in the match set (allowed: %s)", requested, strings.Join(allowed, ", "))
}

// workItemRefJSON is the wire shape that mirrors the shared work_item_ref
// schema. The handler is the only writer; field tags must stay in sync with
// internal/contract/schemas/shared/work_item_ref.json.
type workItemRefJSON struct {
	Source    string   `json:"source"`
	ForgeHost string   `json:"forge_host,omitempty"`
	Owner     string   `json:"owner,omitempty"`
	Repo      string   `json:"repo,omitempty"`
	Number    int      `json:"number,omitempty"`
	URL       string   `json:"url"`
	Title     string   `json:"title"`
	Labels    []string `json:"labels,omitempty"`
	State     string   `json:"state"`
	CreatedAt string   `json:"created_at"`
}

// serializeWorkItemRef encodes the in-memory ref into the JSON wire form used
// as a pipeline run input. Forge-specific fields are populated when the ref
// looks like a forge work-item; otherwise the document falls back to the
// "manual" source. hostOverride, when non-empty, replaces the canonical host
// inferred from the forge type — used to capture self-hosted gitea/gitlab
// instances whose hostnames the canonical map can't know.
func serializeWorkItemRef(ref worksource.WorkItemRef, hostOverride string) (string, error) {
	source, host := mapForgeToSource(ref.Forge)
	if hostOverride != "" {
		host = hostOverride
	}

	owner, repo := splitRepoSlug(ref.Repo)

	var number int
	if ref.ID != "" {
		if n, err := strconv.Atoi(ref.ID); err == nil {
			number = n
		}
	}

	url := ref.URL
	title := ref.Title
	if title == "" {
		title = fmt.Sprintf("%s/%s #%s", owner, repo, ref.ID)
	}
	if url == "" {
		url = fmt.Sprintf("wave://dispatch/%s/%s/%s/%s/%s", source, host, owner, repo, ref.ID)
	}

	doc := workItemRefJSON{
		Source:    source,
		ForgeHost: host,
		Owner:     owner,
		Repo:      repo,
		Number:    number,
		URL:       url,
		Title:     title,
		Labels:    ref.Labels,
		State:     ref.State,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}
	if doc.State == "" {
		doc.State = "open"
	}

	b, err := json.Marshal(doc)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// mapForgeToSource translates a forge type as stored on a worksource binding
// (e.g. "github", "codeberg", "forgejo") into the (source, host) tuple
// expected by the shared work_item_ref schema. Codeberg/Forgejo collapse to
// "gitea" because they share Gitea's API surface.
func mapForgeToSource(forgeName string) (source, host string) {
	switch strings.ToLower(forgeName) {
	case string(forge.ForgeGitHub):
		return string(forge.ForgeGitHub), "github.com"
	case string(forge.ForgeGitLab):
		return string(forge.ForgeGitLab), "gitlab.com"
	case string(forge.ForgeBitbucket):
		return string(forge.ForgeBitbucket), "bitbucket.org"
	case string(forge.ForgeGitea):
		return string(forge.ForgeGitea), "gitea.com"
	case string(forge.ForgeCodeberg):
		return string(forge.ForgeGitea), "codeberg.org"
	case string(forge.ForgeForgejo):
		return string(forge.ForgeGitea), "codeberg.org"
	}
	return "manual", ""
}
