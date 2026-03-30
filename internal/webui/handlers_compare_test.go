package webui

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHandleAPICompare_MissingParams(t *testing.T) {
	srv, _ := testServer(t)

	tests := []struct {
		name string
		url  string
	}{
		{"both missing", "/api/compare"},
		{"left missing", "/api/compare?right=abc"},
		{"right missing", "/api/compare?left=abc"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.url, nil)
			rec := httptest.NewRecorder()
			srv.handleAPICompare(rec, req)
			if rec.Code != http.StatusBadRequest {
				t.Errorf("expected 400, got %d", rec.Code)
			}
		})
	}
}

func TestHandleAPICompare_RunNotFound(t *testing.T) {
	srv, _ := testServer(t)

	req := httptest.NewRequest("GET", "/api/compare?left=nonexistent&right=also-nonexistent", nil)
	rec := httptest.NewRecorder()
	srv.handleAPICompare(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestHandleAPICompare_Success(t *testing.T) {
	srv, rwStore := testServer(t)

	// Create two runs
	leftID, err := rwStore.CreateRun("test-pipeline", "input1")
	if err != nil {
		t.Fatalf("failed to create left run: %v", err)
	}
	rightID, err := rwStore.CreateRun("test-pipeline", "input2")
	if err != nil {
		t.Fatalf("failed to create right run: %v", err)
	}

	// Complete both runs with different token counts
	if err := rwStore.UpdateRunStatus(leftID, "completed", "", 1000); err != nil {
		t.Fatalf("failed to update left run: %v", err)
	}
	if err := rwStore.UpdateRunStatus(rightID, "completed", "", 1500); err != nil {
		t.Fatalf("failed to update right run: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/compare?left="+leftID+"&right="+rightID, nil)
	rec := httptest.NewRecorder()
	srv.handleAPICompare(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp CompareResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Left.RunID != leftID {
		t.Errorf("Left.RunID: expected %q, got %q", leftID, resp.Left.RunID)
	}
	if resp.Right.RunID != rightID {
		t.Errorf("Right.RunID: expected %q, got %q", rightID, resp.Right.RunID)
	}
}

func TestHandleComparePage_MissingParams(t *testing.T) {
	srv, _ := testServer(t)

	req := httptest.NewRequest("GET", "/compare", nil)
	rec := httptest.NewRecorder()
	srv.handleComparePage(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	if !contains(body, "<select") {
		t.Error("expected body to contain <select> elements for run selector")
	}
	if !contains(body, "compare-left-select") {
		t.Error("expected body to contain left select element")
	}
	if !contains(body, "compare-right-select") {
		t.Error("expected body to contain right select element")
	}
	if !contains(body, "<nav") {
		t.Error("expected body to contain navbar (layout template)")
	}
}

func TestHandleComparePage_RunNotFound(t *testing.T) {
	srv, _ := testServer(t)

	req := httptest.NewRequest("GET", "/compare?left=nonexistent&right=also-nonexistent", nil)
	rec := httptest.NewRecorder()
	srv.handleComparePage(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	if !contains(body, "not found") {
		t.Error("expected body to contain error message about run not found")
	}
	if !contains(body, "<nav") {
		t.Error("expected body to contain navbar (layout template)")
	}
	if !contains(body, "<select") {
		t.Error("expected body to contain selector form for retry")
	}
}

func TestHandleComparePage_Success(t *testing.T) {
	srv, rwStore := testServer(t)

	leftID, _ := rwStore.CreateRun("test-pipeline", "input1")
	rightID, _ := rwStore.CreateRun("test-pipeline", "input2")
	_ = rwStore.UpdateRunStatus(leftID, "completed", "", 500)
	_ = rwStore.UpdateRunStatus(rightID, "completed", "", 800)

	req := httptest.NewRequest("GET", "/compare?left="+leftID+"&right="+rightID, nil)
	rec := httptest.NewRecorder()
	srv.handleComparePage(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	body := rec.Body.String()
	if !contains(body, leftID) {
		t.Errorf("expected body to contain left run ID %q", leftID)
	}
	if !contains(body, rightID) {
		t.Errorf("expected body to contain right run ID %q", rightID)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestBuildCompareRows_MatchingSteps(t *testing.T) {
	now := time.Now()
	later := now.Add(30 * time.Second)
	muchLater := now.Add(60 * time.Second)

	leftSteps := []StepDetail{
		{StepID: "step-1", State: "completed", TokensUsed: 100, StartedAt: &now, CompletedAt: &later},
		{StepID: "step-2", State: "failed", TokensUsed: 50},
	}
	rightSteps := []StepDetail{
		{StepID: "step-1", State: "completed", TokensUsed: 150, StartedAt: &now, CompletedAt: &muchLater},
		{StepID: "step-2", State: "completed", TokensUsed: 80},
	}

	rows := buildCompareRows(leftSteps, rightSteps)

	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}

	// Step 1: same status
	if rows[0].StepID != "step-1" {
		t.Errorf("row 0: expected step-1, got %q", rows[0].StepID)
	}
	if rows[0].StateDiff {
		t.Error("row 0: expected StateDiff=false for matching states")
	}
	if rows[0].DurationClass != "compare-regression" {
		t.Errorf("row 0: expected duration regression, got %q", rows[0].DurationClass)
	}
	if rows[0].TokensClass != "compare-regression" {
		t.Errorf("row 0: expected tokens regression, got %q", rows[0].TokensClass)
	}

	// Step 2: different status
	if rows[1].StepID != "step-2" {
		t.Errorf("row 1: expected step-2, got %q", rows[1].StepID)
	}
	if !rows[1].StateDiff {
		t.Error("row 1: expected StateDiff=true for different states")
	}
}

func TestBuildCompareRows_UnmatchedSteps(t *testing.T) {
	leftSteps := []StepDetail{
		{StepID: "left-only", State: "completed"},
	}
	rightSteps := []StepDetail{
		{StepID: "right-only", State: "completed"},
	}

	rows := buildCompareRows(leftSteps, rightSteps)

	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}

	// Left-only step
	if rows[0].StepID != "left-only" {
		t.Errorf("row 0: expected left-only, got %q", rows[0].StepID)
	}
	if rows[0].RightState != "-" {
		t.Errorf("row 0: expected RightState='-', got %q", rows[0].RightState)
	}
	if !rows[0].StateDiff {
		t.Error("row 0: expected StateDiff=true")
	}

	// Right-only step
	if rows[1].StepID != "right-only" {
		t.Errorf("row 1: expected right-only, got %q", rows[1].StepID)
	}
	if rows[1].LeftState != "-" {
		t.Errorf("row 1: expected LeftState='-', got %q", rows[1].LeftState)
	}
}

func TestComputeTokensDelta(t *testing.T) {
	tests := []struct {
		name      string
		left      int
		right     int
		wantDelta string
		wantClass string
	}{
		{"both zero", 0, 0, "", ""},
		{"equal", 100, 100, "same", ""},
		{"improvement", 1000, 500, "-500", "compare-improvement"},
		{"regression", 500, 1000, "+500", "compare-regression"},
		{"large improvement", 50000, 30000, "-20k", "compare-improvement"},
		{"small values", 10, 20, "+10", "compare-regression"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotDelta, gotClass := computeTokensDelta(tt.left, tt.right)
			if gotDelta != tt.wantDelta {
				t.Errorf("delta: expected %q, got %q", tt.wantDelta, gotDelta)
			}
			if gotClass != tt.wantClass {
				t.Errorf("class: expected %q, got %q", tt.wantClass, gotClass)
			}
		})
	}
}

func TestFormatTokensDelta(t *testing.T) {
	tests := []struct {
		tokens int
		want   string
	}{
		{0, "0"},
		{500, "500"},
		{999, "999"},
		{1000, "1k"},
		{1500, "1.5k"},
		{2000, "2k"},
		{10000, "10k"},
		{45300, "45k"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := formatTokensDelta(tt.tokens)
			if got != tt.want {
				t.Errorf("formatTokensDelta(%d): expected %q, got %q", tt.tokens, tt.want, got)
			}
		})
	}
}

func TestStepDuration(t *testing.T) {
	now := time.Now()
	later := now.Add(45 * time.Second)

	tests := []struct {
		name string
		sd   StepDetail
		want time.Duration
	}{
		{"no start", StepDetail{}, 0},
		{"started no end", StepDetail{StartedAt: &now}, 0},
		{"completed", StepDetail{StartedAt: &now, CompletedAt: &later}, 45 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stepDuration(tt.sd)
			if got != tt.want {
				t.Errorf("expected %v, got %v", tt.want, got)
			}
		})
	}
}
