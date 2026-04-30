package webui

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/recinq/wave/internal/worksource"
)

// TestHandleWorkBoard_Empty verifies that /work renders a 200 with the
// empty-state copy when no bindings are configured.
func TestHandleWorkBoard_Empty(t *testing.T) {
	srv, _ := testServer(t)

	req := httptest.NewRequest("GET", "/work", nil)
	rec := httptest.NewRecorder()
	srv.handleWorkBoard(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "No bindings yet") {
		t.Errorf("expected empty-state copy, got: %s", body)
	}
}

// TestHandleWorkBoard_WithBindings verifies that two created bindings appear
// in the rendered output with their pipeline names and trigger labels.
func TestHandleWorkBoard_WithBindings(t *testing.T) {
	srv, _ := testServer(t)

	ctx := context.Background()
	if _, err := srv.runtime.worksource.CreateBinding(ctx, worksource.BindingSpec{
		Forge:        "github",
		RepoPattern:  "re-cinq/wave",
		PipelineName: "impl-issue",
		Trigger:      worksource.TriggerOnLabel,
		LabelFilter:  []string{"auto-impl"},
	}); err != nil {
		t.Fatalf("CreateBinding 1: %v", err)
	}
	if _, err := srv.runtime.worksource.CreateBinding(ctx, worksource.BindingSpec{
		Forge:        "github",
		RepoPattern:  "re-cinq/*",
		PipelineName: "research",
		Trigger:      worksource.TriggerOnDemand,
	}); err != nil {
		t.Fatalf("CreateBinding 2: %v", err)
	}

	req := httptest.NewRequest("GET", "/work", nil)
	rec := httptest.NewRecorder()
	srv.handleWorkBoard(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "impl-issue") {
		t.Errorf("expected body to contain pipeline name 'impl-issue', got: %s", body)
	}
	if !strings.Contains(body, "research") {
		t.Errorf("expected body to contain pipeline name 'research', got: %s", body)
	}
	if !strings.Contains(body, "On label") {
		t.Errorf("expected body to contain trigger label 'On label', got: %s", body)
	}
	if !strings.Contains(body, "On demand") {
		t.Errorf("expected body to contain trigger label 'On demand', got: %s", body)
	}
	if strings.Contains(body, "No bindings yet") {
		t.Errorf("expected populated state, found empty-state copy: %s", body)
	}
}

// TestHandleWorkItemDetail_NoMatch verifies that a path with no matching
// binding still renders successfully and surfaces the "no bindings match"
// copy.
func TestHandleWorkItemDetail_NoMatch(t *testing.T) {
	srv, _ := testServer(t)

	req := httptest.NewRequest("GET", "/work/github/foo/bar/1", nil)
	req.SetPathValue("forge", "github")
	req.SetPathValue("owner", "foo")
	req.SetPathValue("repo", "bar")
	req.SetPathValue("number", "1")
	rec := httptest.NewRecorder()
	srv.handleWorkItemDetail(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "No bindings match this work item") {
		t.Errorf("expected 'No bindings match' copy, got: %s", body)
	}
	if !strings.Contains(body, "Work item #1") {
		t.Errorf("expected fallback title 'Work item #1', got: %s", body)
	}
	if !strings.Contains(body, "github / foo/bar #1") {
		t.Errorf("expected coordinates header, got: %s", body)
	}
}

// TestHandleWorkItemDetail_OneMatch verifies that a binding whose RepoPattern
// matches the path renders the binding pipeline name and the disabled
// "Run on this issue" button.
func TestHandleWorkItemDetail_OneMatch(t *testing.T) {
	srv, _ := testServer(t)

	ctx := context.Background()
	if _, err := srv.runtime.worksource.CreateBinding(ctx, worksource.BindingSpec{
		Forge:        "github",
		RepoPattern:  "foo/bar",
		PipelineName: "impl-issue",
		Trigger:      worksource.TriggerOnDemand,
	}); err != nil {
		t.Fatalf("CreateBinding: %v", err)
	}

	req := httptest.NewRequest("GET", "/work/github/foo/bar/42", nil)
	req.SetPathValue("forge", "github")
	req.SetPathValue("owner", "foo")
	req.SetPathValue("repo", "bar")
	req.SetPathValue("number", "42")
	rec := httptest.NewRecorder()
	srv.handleWorkItemDetail(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "impl-issue") {
		t.Errorf("expected matched binding pipeline 'impl-issue', got: %s", body)
	}
	if !strings.Contains(body, "Run on this issue") {
		t.Errorf("expected disabled CTA copy 'Run on this issue', got: %s", body)
	}
	if !strings.Contains(body, "disabled") {
		t.Errorf("expected disabled attribute on CTA, got: %s", body)
	}
	if strings.Contains(body, "No bindings match this work item") {
		t.Errorf("did not expect no-match copy when a binding matches: %s", body)
	}
}

// TestHandleWorkItemDetail_BadNumber verifies the handler rejects a
// non-integer number with HTTP 400.
func TestHandleWorkItemDetail_BadNumber(t *testing.T) {
	srv, _ := testServer(t)

	req := httptest.NewRequest("GET", "/work/github/foo/bar/abc", nil)
	req.SetPathValue("forge", "github")
	req.SetPathValue("owner", "foo")
	req.SetPathValue("repo", "bar")
	req.SetPathValue("number", "abc")
	rec := httptest.NewRecorder()
	srv.handleWorkItemDetail(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for non-integer number, got %d", rec.Code)
	}
}
