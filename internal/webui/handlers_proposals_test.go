package webui

import (
	"encoding/json"
	"html/template"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/recinq/wave/internal/state"
)

// proposalsTestServer extends testServer with the proposals page templates so
// the proposals handlers can render. The shared testTemplates map only stubs
// the older pages.
func proposalsTestServer(t *testing.T) (*Server, state.StateStore) {
	t.Helper()
	srv, rw := testServer(t)

	listTmpl := template.Must(template.New("templates/layout.html").Funcs(template.FuncMap{
		"proposalStatusBadgeClass": proposalStatusBadgeClass,
		"formatTime":               formatTime,
	}).Parse(`<html><body>` +
		`<h1>Proposals</h1>` +
		`<div class="filter">{{.Filter}}</div>` +
		`<ul>{{range .Proposals}}<li data-id="{{.ID}}">{{.PipelineName}} v{{.VersionBefore}}-&gt;v{{.VersionAfter}} <span class="status">{{.Status}}</span></li>{{end}}</ul>` +
		`<div class="counts">proposed={{index .Counts "proposed"}} approved={{index .Counts "approved"}} rejected={{index .Counts "rejected"}}</div>` +
		`</body></html>`))
	detailTmpl := template.Must(template.New("templates/layout.html").Funcs(template.FuncMap{
		"proposalStatusBadgeClass": proposalStatusBadgeClass,
		"formatTime":               formatTime,
	}).Parse(`<html><body>` +
		`<h1>#{{.Proposal.ID}}: {{.Proposal.Reason}}</h1>` +
		`<div class="status">{{.Proposal.Status}}</div>` +
		`<div class="pipeline">{{.Proposal.PipelineName}}</div>` +
		`{{if .DiffLines}}<pre>{{range .DiffLines}}<span class="{{.Class}}">{{.Text}}</span>` + "\n" + `{{end}}</pre>{{end}}` +
		`{{if .DiffMissing}}<p class="diff-missing">no diff</p>{{end}}` +
		`{{if .DiffError}}<p class="diff-error">{{.DiffError}}</p>{{end}}` +
		`{{if .SignalSummary}}<pre class="signal">{{.SignalSummary}}</pre>{{end}}` +
		`</body></html>`))
	srv.assets.templates["templates/proposals/list.html"] = listTmpl
	srv.assets.templates["templates/proposals/detail.html"] = detailTmpl
	return srv, rw
}

func seedProposal(t *testing.T, store state.StateStore, name string, status state.EvolutionProposalStatus, diffPath string) int64 {
	t.Helper()
	id, err := store.CreateProposal(state.EvolutionProposalRecord{
		PipelineName:  name,
		VersionBefore: 1,
		VersionAfter:  2,
		DiffPath:      diffPath,
		Reason:        "tighten contract for " + name,
		SignalSummary: `{"judge_score":0.78}`,
	})
	if err != nil {
		t.Fatalf("CreateProposal: %v", err)
	}
	if status != state.ProposalProposed {
		if err := store.DecideProposal(id, status, "seed"); err != nil {
			t.Fatalf("DecideProposal: %v", err)
		}
	}
	return id
}

// writeAfterYAML writes a synthetic post-diff yaml file alongside diffPath so
// the approve flow has something to hash + record.
func writeAfterYAML(t *testing.T, diffPath, body string) string {
	t.Helper()
	yamlPath := diffPath + ".after.yaml"
	if err := os.WriteFile(yamlPath, []byte(body), 0o600); err != nil {
		t.Fatalf("write after-yaml: %v", err)
	}
	return yamlPath
}

func TestHandleProposalsPage_FiltersAndCounts(t *testing.T) {
	srv, rw := proposalsTestServer(t)
	dir := t.TempDir()
	diff := filepath.Join(dir, "p1.diff")
	_ = os.WriteFile(diff, []byte("--- a\n+++ b\n"), 0o600)

	seedProposal(t, rw, "impl-issue", state.ProposalProposed, diff)
	seedProposal(t, rw, "scope", state.ProposalApproved, diff)

	req := httptest.NewRequest("GET", "/proposals?status=proposed", nil)
	req.SetPathValue("id", "")
	rec := httptest.NewRecorder()
	srv.handleProposalsPage(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "impl-issue") {
		t.Errorf("expected impl-issue in body, got: %s", body)
	}
	if strings.Contains(body, "scope") {
		t.Errorf("approved 'scope' proposal should not appear under default proposed filter, body=%s", body)
	}
	if !strings.Contains(body, "proposed=1") || !strings.Contains(body, "approved=1") {
		t.Errorf("counts wrong, body=%s", body)
	}
}

func TestHandleProposalDetailPage_RendersDiff(t *testing.T) {
	srv, rw := proposalsTestServer(t)
	dir := t.TempDir()
	diff := filepath.Join(dir, "p.diff")
	_ = os.WriteFile(diff, []byte("--- a/x\n+++ b/x\n@@ -1,2 +1,2 @@\n-old\n+new\n ctx\n"), 0o600)
	id := seedProposal(t, rw, "impl-issue", state.ProposalProposed, diff)

	req := httptest.NewRequest("GET", "/proposals/"+strconv.FormatInt(id, 10), nil)
	req.SetPathValue("id", strconv.FormatInt(id, 10))
	rec := httptest.NewRecorder()
	srv.handleProposalDetailPage(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, want := range []string{"impl-issue", `class="diff-line-add"`, `class="diff-line-del"`, `class="diff-line-ctx"`, `judge_score`} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing %q. body=%s", want, body)
		}
	}
}

func TestHandleProposalDetailPage_NotFound(t *testing.T) {
	srv, _ := proposalsTestServer(t)
	req := httptest.NewRequest("GET", "/proposals/9999", nil)
	req.SetPathValue("id", "9999")
	rec := httptest.NewRecorder()
	srv.handleProposalDetailPage(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestHandleProposalDetailPage_BadID(t *testing.T) {
	srv, _ := proposalsTestServer(t)
	req := httptest.NewRequest("GET", "/proposals/abc", nil)
	req.SetPathValue("id", "abc")
	rec := httptest.NewRecorder()
	srv.handleProposalDetailPage(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandleProposalApprove_FlipsActive(t *testing.T) {
	srv, rw := proposalsTestServer(t)
	dir := t.TempDir()
	diff := filepath.Join(dir, "p.diff")
	_ = os.WriteFile(diff, []byte("diff"), 0o600)
	yaml := writeAfterYAML(t, diff, "version: 2\nsteps: []\n")

	// Pre-seed a v1 active row so approve must compute v2.
	if err := rw.CreatePipelineVersion(state.PipelineVersionRecord{
		PipelineName: "impl-issue", Version: 1, SHA256: "sha-old", YAMLPath: "old.yaml", Active: true,
	}); err != nil {
		t.Fatalf("CreatePipelineVersion(v1): %v", err)
	}

	id := seedProposal(t, rw, "impl-issue", state.ProposalProposed, diff)
	idStr := strconv.FormatInt(id, 10)
	req := httptest.NewRequest("POST", "/proposals/"+idStr+"/approve", nil)
	req.SetPathValue("id", idStr)
	rec := httptest.NewRecorder()
	srv.handleProposalApprove(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var resp proposalDecisionResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.NewVersion != 2 {
		t.Errorf("expected NewVersion=2, got %d", resp.NewVersion)
	}
	if resp.NewYAMLPath != yaml {
		t.Errorf("yaml mismatch: got %s want %s", resp.NewYAMLPath, yaml)
	}
	active, err := rw.GetActiveVersion("impl-issue")
	if err != nil {
		t.Fatalf("GetActiveVersion: %v", err)
	}
	if active == nil || active.Version != 2 || !active.Active {
		t.Errorf("expected active v2, got %+v", active)
	}
}

func TestHandleProposalApprove_AfterYAMLMissing(t *testing.T) {
	srv, rw := proposalsTestServer(t)
	dir := t.TempDir()
	diff := filepath.Join(dir, "p.diff")
	_ = os.WriteFile(diff, []byte("diff"), 0o600)
	id := seedProposal(t, rw, "impl-issue", state.ProposalProposed, diff)

	idStr := strconv.FormatInt(id, 10)
	req := httptest.NewRequest("POST", "/proposals/"+idStr+"/approve", nil)
	req.SetPathValue("id", idStr)
	rec := httptest.NewRecorder()
	srv.handleProposalApprove(rec, req)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 missing-yaml, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandleProposalReject_LeavesVersionsUntouched(t *testing.T) {
	srv, rw := proposalsTestServer(t)
	dir := t.TempDir()
	diff := filepath.Join(dir, "p.diff")
	_ = os.WriteFile(diff, []byte("diff"), 0o600)

	if err := rw.CreatePipelineVersion(state.PipelineVersionRecord{
		PipelineName: "impl-issue", Version: 1, SHA256: "sha-old", YAMLPath: "old.yaml", Active: true,
	}); err != nil {
		t.Fatalf("CreatePipelineVersion(v1): %v", err)
	}
	id := seedProposal(t, rw, "impl-issue", state.ProposalProposed, diff)

	idStr := strconv.FormatInt(id, 10)
	req := httptest.NewRequest("POST", "/proposals/"+idStr+"/reject", nil)
	req.SetPathValue("id", idStr)
	rec := httptest.NewRecorder()
	srv.handleProposalReject(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	versions, err := rw.ListPipelineVersions("impl-issue")
	if err != nil {
		t.Fatalf("ListPipelineVersions: %v", err)
	}
	if len(versions) != 1 || versions[0].Version != 1 {
		t.Errorf("rejection should leave versions unchanged, got %+v", versions)
	}
	rec2, err := rw.GetProposal(id)
	if err != nil {
		t.Fatalf("GetProposal: %v", err)
	}
	if rec2 == nil || rec2.Status != state.ProposalRejected {
		t.Errorf("expected status=rejected, got %+v", rec2)
	}
}

func TestParseDiffLines_Classes(t *testing.T) {
	in := "--- a/x\n+++ b/x\n@@ -1,2 +1,2 @@\n-foo\n+bar\n ctx\n"
	out := parseDiffLines(in)
	wantClasses := []string{"diff-line-meta", "diff-line-meta", "diff-line-hunk", "diff-line-del", "diff-line-add", "diff-line-ctx"}
	if len(out) != len(wantClasses) {
		t.Fatalf("got %d lines want %d. lines=%+v", len(out), len(wantClasses), out)
	}
	for i, want := range wantClasses {
		if out[i].Class != want {
			t.Errorf("line %d class=%q want %q", i, out[i].Class, want)
		}
	}
}
