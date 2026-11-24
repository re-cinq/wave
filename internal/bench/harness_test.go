package bench

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPredictionsIncludeEveryTask(t *testing.T) {
	r := &BenchReport{Results: []BenchResult{
		{TaskID: "a", PatchDiff: "diff a"},
		{TaskID: "b", Status: StatusError},
	}}
	path := filepath.Join(t.TempDir(), "sub", "preds.jsonl")
	if err := WritePredictions(path, Predictions(r, "wave-x")); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 2 {
		t.Fatalf("got %d lines, want 2", len(lines))
	}
	var p Prediction
	if err := json.Unmarshal([]byte(lines[1]), &p); err != nil {
		t.Fatal(err)
	}
	if p.InstanceID != "b" || p.ModelNameOrPath != "wave-x" || p.ModelPatch != "" {
		t.Errorf("prediction = %+v", p)
	}
}

func TestApplyHarnessReport(t *testing.T) {
	r := &BenchReport{Results: []BenchResult{
		{TaskID: "resolved", Status: StatusPatched},
		{TaskID: "unresolved", Status: StatusPatched},
		{TaskID: "no-patch", Status: StatusError, Error: "solver produced no patch"},
		{TaskID: "harness-error", Status: StatusPatched},
		{TaskID: "missing", Status: StatusPatched},
	}}
	hr := &HarnessReport{
		ResolvedIDs:   []string{"resolved"},
		UnresolvedIDs: []string{"unresolved"},
		EmptyPatchIDs: []string{"no-patch"},
		ErrorIDs:      []string{"harness-error"},
	}
	ApplyHarnessReport(r, hr, "m.r1.json")

	want := map[string]BenchStatus{
		"resolved":      StatusPass,
		"unresolved":    StatusFail,
		"no-patch":      StatusError,
		"harness-error": StatusError,
		"missing":       StatusError,
	}
	for _, res := range r.Results {
		if res.Status != want[res.TaskID] {
			t.Errorf("%s: status %q, want %q", res.TaskID, res.Status, want[res.TaskID])
		}
	}
	if r.Results[2].Error != "solver produced no patch" {
		t.Errorf("run error should be kept, got %q", r.Results[2].Error)
	}
	if r.Passed != 1 || r.Failed != 1 || r.Errors != 3 || r.Patched != 0 {
		t.Errorf("tally = %d/%d/%d/%d", r.Passed, r.Failed, r.Errors, r.Patched)
	}
	if r.Evaluation != "swe-bench harness: m.r1.json" {
		t.Errorf("Evaluation = %q", r.Evaluation)
	}
}

func TestLoadHarnessReport(t *testing.T) {
	dir := t.TempDir()
	good := filepath.Join(dir, "good.json")
	_ = os.WriteFile(good, []byte(`{"total_instances":2,"resolved_ids":["a"],"unresolved_ids":["b"]}`), 0o644)
	hr, err := LoadHarnessReport(good)
	if err != nil || len(hr.ResolvedIDs) != 1 {
		t.Fatalf("LoadHarnessReport = %+v, %v", hr, err)
	}
	bad := filepath.Join(dir, "bad.json")
	_ = os.WriteFile(bad, []byte(`{"pipeline":"x"}`), 0o644)
	if _, err := LoadHarnessReport(bad); err == nil {
		t.Error("expected an error for a file that is not a harness report")
	}
}
