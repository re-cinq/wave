package webui

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/recinq/wave/internal/contract/schemas/shared"
	"github.com/recinq/wave/internal/state"
	"github.com/recinq/wave/internal/worksource"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

// dispatchTestServer extends testServer with a worksource service backed by the
// real state store. The handler under test reads s.runtime.worksource directly,
// so tests must wire it up explicitly.
func dispatchTestServer(t *testing.T) (*Server, state.StateStore) {
	t.Helper()
	srv, store := testServer(t)
	srv.runtime.worksource = worksource.NewService(store)
	// Pin cwd so the detached subprocess SpawnDetached doesn't litter the repo
	// with a top-level .agents/logs directory across the test suite.
	tmp := t.TempDir()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	return srv, store
}

// seedBinding inserts a worksource binding directly into the state store. It
// returns the new binding id.
func seedBinding(t *testing.T, store state.WorksourceStore, pipelineName string) int64 {
	t.Helper()
	const selector = "{}"
	id, err := store.CreateBinding(state.WorksourceBindingRecord{
		Forge:        "github",
		Repo:         "re-cinq/wave",
		Selector:     selector,
		PipelineName: pipelineName,
		Trigger:      state.TriggerOnDemand,
		Active:       true,
	})
	if err != nil {
		t.Fatalf("seedBinding: %v", err)
	}
	return id
}

func newDispatchRequest(t *testing.T, number string, body url.Values) *http.Request {
	t.Helper()
	const (
		forge = "github"
		owner = "re-cinq"
		repo  = "wave"
	)
	target := "/work/" + forge + "/" + owner + "/" + repo + "/" + number + "/dispatch"
	r := httptest.NewRequest(http.MethodPost, target, strings.NewReader(body.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.SetPathValue("forge", forge)
	r.SetPathValue("owner", owner)
	r.SetPathValue("repo", repo)
	r.SetPathValue("number", number)
	return r
}

func compileSharedWorkItemRef(t *testing.T) *jsonschema.Schema {
	t.Helper()
	raw, ok := shared.Lookup("work_item_ref")
	if !ok {
		t.Fatal("work_item_ref schema not registered")
	}
	var doc any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("schema parse: %v", err)
	}
	const schemaURL = "wave://shared/work_item_ref"
	c := jsonschema.NewCompiler()
	if err := c.AddResource(schemaURL, doc); err != nil {
		t.Fatalf("add resource: %v", err)
	}
	compiled, err := c.Compile(schemaURL)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	return compiled
}

func TestHandleWorkDispatch_NoMatch_409(t *testing.T) {
	srv, _ := dispatchTestServer(t)

	req := newDispatchRequest(t, "1595", url.Values{})
	rec := httptest.NewRecorder()
	srv.handleWorkDispatch(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleWorkDispatch_SingleMatch_RedirectAndRunRow(t *testing.T) {
	srv, store := dispatchTestServer(t)
	seedBinding(t, store, "impl-issue")

	req := newDispatchRequest(t, "1595", url.Values{})
	rec := httptest.NewRecorder()
	srv.handleWorkDispatch(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d: %s", rec.Code, rec.Body.String())
	}
	loc := rec.Header().Get("Location")
	if !strings.HasPrefix(loc, "/runs/") {
		t.Fatalf("expected /runs/<id> redirect, got %q", loc)
	}
	runID := strings.TrimPrefix(loc, "/runs/")

	run, err := store.GetRun(runID)
	if err != nil {
		t.Fatalf("GetRun(%s): %v", runID, err)
	}
	if run.PipelineName != "impl-issue" {
		t.Errorf("expected pipeline impl-issue, got %q", run.PipelineName)
	}

	// Round-trip: input must validate against the shared work_item_ref schema
	// and decode to the expected work-item coordinates.
	schema := compileSharedWorkItemRef(t)
	var doc any
	if err := json.Unmarshal([]byte(run.Input), &doc); err != nil {
		t.Fatalf("input is not JSON: %v\n%s", err, run.Input)
	}
	if err := schema.Validate(doc); err != nil {
		t.Errorf("input fails work_item_ref schema: %v\n%s", err, run.Input)
	}

	var parsed workItemRefJSON
	if err := json.Unmarshal([]byte(run.Input), &parsed); err != nil {
		t.Fatalf("re-decode input: %v", err)
	}
	if parsed.Source != "github" {
		t.Errorf("source = %q, want github", parsed.Source)
	}
	if parsed.Owner != "re-cinq" || parsed.Repo != "wave" || parsed.Number != 1595 {
		t.Errorf("coords = %s/%s#%d, want re-cinq/wave#1595", parsed.Owner, parsed.Repo, parsed.Number)
	}
}

func TestHandleWorkDispatch_MultiMatch_NoPipeline_400(t *testing.T) {
	srv, store := dispatchTestServer(t)
	seedBinding(t, store, "impl-issue")
	seedBinding(t, store, "research")

	req := newDispatchRequest(t, "1595", url.Values{})
	rec := httptest.NewRecorder()
	srv.handleWorkDispatch(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "impl-issue") || !strings.Contains(rec.Body.String(), "research") {
		t.Errorf("error body should list candidates, got %q", rec.Body.String())
	}
}

func TestHandleWorkDispatch_MultiMatch_ValidPipeline_Redirect(t *testing.T) {
	srv, store := dispatchTestServer(t)
	seedBinding(t, store, "impl-issue")
	seedBinding(t, store, "research")

	body := url.Values{"pipeline": []string{"research"}}
	req := newDispatchRequest(t, "1595", body)
	rec := httptest.NewRecorder()
	srv.handleWorkDispatch(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d: %s", rec.Code, rec.Body.String())
	}
	runID := strings.TrimPrefix(rec.Header().Get("Location"), "/runs/")
	run, err := store.GetRun(runID)
	if err != nil {
		t.Fatalf("GetRun: %v", err)
	}
	if run.PipelineName != "research" {
		t.Errorf("expected research, got %q", run.PipelineName)
	}
}

func TestHandleWorkDispatch_MultiMatch_BogusPipeline_400(t *testing.T) {
	srv, store := dispatchTestServer(t)
	seedBinding(t, store, "impl-issue")
	seedBinding(t, store, "research")

	body := url.Values{"pipeline": []string{"definitely-not-real"}}
	req := newDispatchRequest(t, "1595", body)
	rec := httptest.NewRecorder()
	srv.handleWorkDispatch(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleWorkDispatch_MalformedNumber_400(t *testing.T) {
	srv, _ := dispatchTestServer(t)

	req := newDispatchRequest(t, "abc", url.Values{})
	rec := httptest.NewRecorder()
	srv.handleWorkDispatch(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

// TestSelectBindingPipeline_RequestedConflictsSingleMatch covers the corner
// where a single binding matches but the caller's `pipeline` form value
// disagrees — we 400 instead of silently overriding the binding decision.
func TestSelectBindingPipeline_RequestedConflictsSingleMatch(t *testing.T) {
	matches := []worksource.BindingRecord{
		{PipelineName: "impl-issue", Active: true},
	}
	pipe, status, msg := selectBindingPipeline(matches, "research")
	if status != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d (%s)", status, msg)
	}
	if pipe != "" {
		t.Errorf("expected empty pipeline on error, got %q", pipe)
	}
}
