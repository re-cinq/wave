package webui

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/recinq/wave/internal/state"
)

func TestHandleAnalyticsPage(t *testing.T) {
	srv, _ := testServer(t)

	req := httptest.NewRequest("GET", "/analytics", nil)
	rec := httptest.NewRecorder()
	srv.handleAnalyticsPage(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	contentType := rec.Header().Get("Content-Type")
	if !strings.Contains(contentType, "text/html") {
		t.Errorf("expected text/html content type, got %q", contentType)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Token Usage Analytics") {
		t.Error("expected page to contain 'Token Usage Analytics' heading")
	}
}

func TestHandleAPIAnalytics(t *testing.T) {
	srv, rwStore := testServer(t)

	req := httptest.NewRequest("GET", "/api/analytics", nil)
	rec := httptest.NewRecorder()
	srv.handleAPIAnalytics(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var analytics TokenAnalytics
	if err := json.NewDecoder(rec.Body).Decode(&analytics); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Empty store should return zero values
	if analytics.TotalRuns != 0 {
		t.Errorf("expected 0 total runs, got %d", analytics.TotalRuns)
	}
	if analytics.TotalTokens != 0 {
		t.Errorf("expected 0 total tokens, got %d", analytics.TotalTokens)
	}

	// Create some runs with token data
	runID1, err := rwStore.CreateRun("impl-issue", "test input 1")
	if err != nil {
		t.Fatalf("failed to create run: %v", err)
	}
	_ = rwStore.UpdateRunStatus(runID1, "completed", "step-1", 5000)

	runID2, err := rwStore.CreateRun("impl-issue", "test input 2")
	if err != nil {
		t.Fatalf("failed to create run: %v", err)
	}
	_ = rwStore.UpdateRunStatus(runID2, "completed", "step-1", 3000)

	runID3, err := rwStore.CreateRun("audit-security", "test input 3")
	if err != nil {
		t.Fatalf("failed to create run: %v", err)
	}
	_ = rwStore.UpdateRunStatus(runID3, "completed", "step-1", 10000)

	// Wait briefly for SQLite to flush
	time.Sleep(10 * time.Millisecond)

	// Query again
	req = httptest.NewRequest("GET", "/api/analytics", nil)
	rec = httptest.NewRecorder()
	srv.handleAPIAnalytics(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	if err := json.NewDecoder(rec.Body).Decode(&analytics); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if analytics.TotalRuns != 3 {
		t.Errorf("expected 3 total runs, got %d", analytics.TotalRuns)
	}
	if analytics.TotalTokens != 18000 {
		t.Errorf("expected 18000 total tokens, got %d", analytics.TotalTokens)
	}
	if analytics.TokensThisWeek != 18000 {
		t.Errorf("expected 18000 tokens this week, got %d", analytics.TokensThisWeek)
	}
	if analytics.RunsThisWeek != 3 {
		t.Errorf("expected 3 runs this week, got %d", analytics.RunsThisWeek)
	}

	// Check pipeline aggregation
	if len(analytics.TopPipelines) < 2 {
		t.Fatalf("expected at least 2 pipelines, got %d", len(analytics.TopPipelines))
	}
	// audit-security should be first (10k avg vs 4k avg for impl-issue)
	if analytics.TopPipelines[0].Name != "audit-security" {
		t.Errorf("expected audit-security as top pipeline, got %q", analytics.TopPipelines[0].Name)
	}

	// Check recent runs (should be in chronological order, oldest first)
	if len(analytics.RecentRuns) != 3 {
		t.Errorf("expected 3 recent runs, got %d", len(analytics.RecentRuns))
	}
}

func TestHandleAPIAnalyticsWithPersonaMetrics(t *testing.T) {
	srv, rwStore := testServer(t)

	// Create a run and record performance metrics with persona data
	runID, err := rwStore.CreateRun("impl-issue", "test")
	if err != nil {
		t.Fatalf("failed to create run: %v", err)
	}
	_ = rwStore.UpdateRunStatus(runID, "completed", "step-1", 5000)

	// Record performance metrics for different personas
	now := time.Now()
	_ = rwStore.RecordPerformanceMetric(&state.PerformanceMetricRecord{
		RunID:        runID,
		StepID:       "fetch",
		PipelineName: "impl-issue",
		Persona:      "navigator",
		StartedAt:    now.Add(-time.Minute),
		TokensUsed:   2000,
		Success:      true,
	})
	_ = rwStore.RecordPerformanceMetric(&state.PerformanceMetricRecord{
		RunID:        runID,
		StepID:       "implement",
		PipelineName: "impl-issue",
		Persona:      "craftsman",
		StartedAt:    now.Add(-30 * time.Second),
		TokensUsed:   3000,
		Success:      true,
	})

	time.Sleep(10 * time.Millisecond)

	req := httptest.NewRequest("GET", "/api/analytics", nil)
	rec := httptest.NewRecorder()
	srv.handleAPIAnalytics(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var analytics TokenAnalytics
	if err := json.NewDecoder(rec.Body).Decode(&analytics); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Should have persona data
	if len(analytics.TopPersonas) < 2 {
		t.Fatalf("expected at least 2 personas, got %d", len(analytics.TopPersonas))
	}

	// craftsman (3000 tokens) should be first
	if analytics.TopPersonas[0].Name != "craftsman" {
		t.Errorf("expected craftsman as top persona, got %q", analytics.TopPersonas[0].Name)
	}
	if analytics.TopPersonas[0].TotalTokens != 3000 {
		t.Errorf("expected 3000 tokens for craftsman, got %d", analytics.TopPersonas[0].TotalTokens)
	}
}

func TestEstimateCost(t *testing.T) {
	tests := []struct {
		tokens   int
		expected string
	}{
		{0, "$0.00"},
		{1000000, "$5.40"}, // 800k * $3/M + 200k * $15/M = $2.40 + $3.00
		{100000, "$0.54"},  // 80k * $3/M + 20k * $15/M = $0.24 + $0.30
		{1000, "$0.0054"},  // very small
	}

	for _, tt := range tests {
		got := estimateCost(tt.tokens)
		if got != tt.expected {
			t.Errorf("estimateCost(%d) = %q, want %q", tt.tokens, got, tt.expected)
		}
	}
}
