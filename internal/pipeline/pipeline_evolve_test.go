package pipeline

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/recinq/wave/internal/state"
)

// TestPipelineEvolveYAMLStructure asserts the embedded pipeline-evolve.yaml
// loads cleanly and exposes the four canonical steps in the order required by
// the spec (gather-eval → analyze → propose → record). Catches YAML drift
// without spinning up the executor.
func TestPipelineEvolveYAMLStructure(t *testing.T) {
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	path := filepath.Join(repoRoot, "internal", "defaults", "embedfs", "pipelines", "pipeline-evolve.yaml")

	loader := &YAMLPipelineLoader{}
	p, err := loader.Load(path)
	if err != nil {
		t.Fatalf("load pipeline-evolve.yaml: %v", err)
	}

	if p.Metadata.Name != "pipeline-evolve" {
		t.Fatalf("metadata.name = %q; want pipeline-evolve", p.Metadata.Name)
	}

	wantSteps := []struct {
		id      string
		kind    string // "command" or "" (prompt)
		persona string
	}{
		{"gather-eval", "command", ""},
		{"analyze", "", "navigator"},
		{"propose", "", "craftsman"},
		{"record", "command", ""},
	}
	if len(p.Steps) != len(wantSteps) {
		t.Fatalf("step count = %d; want %d", len(p.Steps), len(wantSteps))
	}
	for i, want := range wantSteps {
		got := p.Steps[i]
		if got.ID != want.id {
			t.Errorf("steps[%d].id = %q; want %q", i, got.ID, want.id)
		}
		if got.Type != want.kind {
			t.Errorf("steps[%d].type = %q; want %q", i, got.Type, want.kind)
		}
		if got.Persona != want.persona {
			t.Errorf("steps[%d].persona = %q; want %q", i, got.Persona, want.persona)
		}
	}

	// analyze + propose must validate against the contract schemas committed
	// alongside the pipeline. Without these the LLM steps would proceed on
	// unstructured output and the record step would crash on missing fields.
	if p.Steps[1].Handover.Contract.Type != "json_schema" ||
		!strings.HasSuffix(p.Steps[1].Handover.Contract.SchemaPath, "evolution-findings.schema.json") {
		t.Errorf("analyze handover schema = %+v; want json_schema → evolution-findings.schema.json",
			p.Steps[1].Handover.Contract)
	}
	if p.Steps[2].Handover.Contract.Type != "json_schema" ||
		!strings.HasSuffix(p.Steps[2].Handover.Contract.SchemaPath, "evolution-proposal.schema.json") {
		t.Errorf("propose handover schema = %+v; want json_schema → evolution-proposal.schema.json",
			p.Steps[2].Handover.Contract)
	}
}

// TestPipelineEvolveRecordWritesProposal exercises the record step's bash
// script against a real on-disk SQLite state DB. Seeds five synthetic
// pipeline_eval rows, fakes the upstream propose-summary artifact, runs the
// extracted bash script with WAVE_DEP_PROPOSE_PROPOSAL_SUMMARY pointing at the
// canned summary, then verifies a row landed in evolution_proposal with
// status='proposed' and the expected pipeline name / version_before /
// version_after / non-empty diff_path / parseable signal_summary JSON.
func TestPipelineEvolveRecordWritesProposal(t *testing.T) {
	if _, err := exec.LookPath("sqlite3"); err != nil {
		t.Skipf("sqlite3 CLI not available: %v", err)
	}
	if _, err := exec.LookPath("jq"); err != nil {
		t.Skipf("jq CLI not available: %v", err)
	}

	tmpRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpRoot, "wave.yaml"), []byte("kind: WaveProject\n"), 0644); err != nil {
		t.Fatalf("write wave.yaml: %v", err)
	}

	dbDir := filepath.Join(tmpRoot, ".agents")
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		t.Fatalf("mkdir .agents: %v", err)
	}
	dbPath := filepath.Join(dbDir, "state.db")

	// Initialise the DB by running migrations and seeding 5 synthetic eval
	// rows for the target pipeline. Closing immediately so sqlite3 CLI can
	// open it from the bash script without lock contention.
	store, err := state.NewStateStore(dbPath)
	if err != nil {
		t.Fatalf("open state store: %v", err)
	}
	pipelineName := "pipeline-evolve-test"
	now := time.Now()
	scoreLow, scoreMid := 0.55, 0.92
	pass, fail := true, false
	one := 1
	dur := int64(12345)
	cost := 0.012
	for i := 0; i < 5; i++ {
		score := &scoreLow
		cp := &fail
		fc := "contract_validation_failed"
		if i%2 == 0 {
			score = &scoreMid
			cp = &pass
			fc = ""
		}
		rec := state.PipelineEvalRecord{
			PipelineName: pipelineName,
			RunID:        "run-" + string(rune('a'+i)),
			JudgeScore:   score,
			ContractPass: cp,
			RetryCount:   &one,
			FailureClass: fc,
			DurationMs:   &dur,
			CostDollars:  &cost,
			RecordedAt:   now.Add(time.Duration(-i) * time.Hour),
		}
		if err := store.RecordEval(rec); err != nil {
			t.Fatalf("RecordEval[%d]: %v", i, err)
		}
	}
	if err := store.Close(); err != nil {
		t.Fatalf("close store: %v", err)
	}

	// Stand up a workspace under <tmpRoot>/.agents/workspaces/<pipe>/record/
	// matching what the executor would create, plus the auto-injected
	// upstream artifact for the propose step.
	wsDir := filepath.Join(tmpRoot, ".agents", "workspaces", "pipeline-evolve", "record")
	depDir := filepath.Join(wsDir, ".agents", "artifacts", "propose")
	if err := os.MkdirAll(depDir, 0755); err != nil {
		t.Fatalf("mkdir dep: %v", err)
	}

	// Canned propose-summary mimicking what a craftsman LLM would emit. The
	// signal_summary JSON is what gets stored verbatim; reason references
	// the dominant failure class so the proposal is auditable.
	summary := map[string]any{
		"pipeline_name":       pipelineName,
		"candidate_yaml_path": ".agents/output/evolution/" + pipelineName + ".next.yaml",
		"diff_path":           ".agents/output/evolution/prompt.diff",
		"reason":              "contract_validation_failed dominates failures; loosen schema and add retry",
		"signal_summary": map[string]any{
			"sample_size":        5,
			"failure_classes":    []map[string]any{{"class": "contract_validation_failed", "count": 2}},
			"judge_score_median": 0.75,
		},
		"version_before": 0,
		"version_after":  1,
	}
	summaryBytes, err := json.Marshal(summary)
	if err != nil {
		t.Fatalf("marshal summary: %v", err)
	}
	summaryPath := filepath.Join(depDir, "proposal-summary")
	if err := os.WriteFile(summaryPath, summaryBytes, 0644); err != nil {
		t.Fatalf("write summary: %v", err)
	}
	// Also stage the diff file at the path the summary points at, so the
	// `cp -f "$DIFF_PATH" "$STABLE_DIFF"` line in the script has something
	// to copy. The script tolerates a missing file but we want the happy
	// path here.
	stagedDiff := filepath.Join(wsDir, ".agents", "output", "evolution", "prompt.diff")
	if err := os.MkdirAll(filepath.Dir(stagedDiff), 0755); err != nil {
		t.Fatalf("mkdir diff: %v", err)
	}
	if err := os.WriteFile(stagedDiff, []byte("--- a/foo\n+++ b/foo\n@@ -1 +1 @@\n-old\n+new\n"), 0644); err != nil {
		t.Fatalf("write diff: %v", err)
	}

	// Pull the record step's script straight out of the embedded YAML so
	// this test catches drift if the script body changes.
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	loader := &YAMLPipelineLoader{}
	p, err := loader.Load(filepath.Join(repoRoot, "internal", "defaults", "embedfs", "pipelines", "pipeline-evolve.yaml"))
	if err != nil {
		t.Fatalf("load pipeline-evolve.yaml: %v", err)
	}
	var script string
	for _, s := range p.Steps {
		if s.ID == "record" {
			script = s.Script
			break
		}
	}
	if script == "" {
		t.Fatal("record step has empty script")
	}

	// Resolve {{ input }} the way the executor would.
	ctx := NewPipelineContext("test-run", "pipeline-evolve", "record")
	ctx.Input = pipelineName
	script = ctx.ResolvePlaceholders(script)
	// Also resolve the relative DIFF_PATH stored in the summary into an
	// absolute path so cp -f finds it from the workspace CWD.
	script = strings.ReplaceAll(script,
		`SUMMARY="${WAVE_DEP_PROPOSE_PROPOSAL_SUMMARY:-.agents/artifacts/propose/proposal-summary}"`,
		`SUMMARY="${WAVE_DEP_PROPOSE_PROPOSAL_SUMMARY}"`,
	)

	// Rewrite the relative diff_path in the canned summary to the absolute
	// staged path so the cp succeeds. Easier than rewriting the script.
	summary["diff_path"] = stagedDiff
	summaryBytes, _ = json.Marshal(summary)
	_ = os.WriteFile(summaryPath, summaryBytes, 0644)

	cmd := exec.Command("sh", "-c", script)
	cmd.Dir = wsDir
	cmd.Env = append(os.Environ(),
		"WAVE_DEP_PROPOSE_PROPOSAL_SUMMARY="+summaryPath,
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("record script failed: %v\nstdout/stderr:\n%s", err, out)
	}

	// Verify record-status.json was written and signals success.
	statusPath := filepath.Join(wsDir, ".agents", "output", "record-status.json")
	statusData, err := os.ReadFile(statusPath)
	if err != nil {
		t.Fatalf("read record-status: %v\nscript output:\n%s", err, out)
	}
	var status struct {
		Status     string `json:"status"`
		ProposalID *int64 `json:"proposal_id"`
		DiffPath   string `json:"diff_path"`
	}
	if err := json.Unmarshal(statusData, &status); err != nil {
		t.Fatalf("parse record-status: %v\ncontent: %s", err, statusData)
	}
	if status.Status != "proposed" {
		t.Errorf("status.status = %q; want proposed", status.Status)
	}
	if status.ProposalID == nil || *status.ProposalID == 0 {
		t.Errorf("status.proposal_id missing/zero: %+v", status)
	}
	if status.DiffPath == "" {
		t.Errorf("status.diff_path empty")
	}

	// Reopen the DB and confirm the row landed with the expected shape.
	store2, err := state.NewStateStore(dbPath)
	if err != nil {
		t.Fatalf("reopen state store: %v", err)
	}
	defer store2.Close()
	rows, err := store2.ListProposalsByStatus(state.ProposalProposed, 0)
	if err != nil {
		t.Fatalf("ListProposalsByStatus: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("proposed rows = %d; want 1; rows: %+v", len(rows), rows)
	}
	got := rows[0]
	if got.PipelineName != pipelineName {
		t.Errorf("pipeline_name = %q; want %q", got.PipelineName, pipelineName)
	}
	if got.VersionBefore != 0 || got.VersionAfter != 1 {
		t.Errorf("version_before=%d version_after=%d; want 0 → 1", got.VersionBefore, got.VersionAfter)
	}
	if got.DiffPath == "" {
		t.Errorf("diff_path empty")
	}
	if got.Reason == "" {
		t.Errorf("reason empty")
	}
	var sig map[string]any
	if err := json.Unmarshal([]byte(got.SignalSummary), &sig); err != nil {
		t.Errorf("signal_summary not valid JSON: %v\ncontent: %s", err, got.SignalSummary)
	}
	if sig["sample_size"] == nil {
		t.Errorf("signal_summary missing sample_size: %+v", sig)
	}
}

// TestPipelineEvolveRecordSkipsInsufficientData verifies the record step's
// short-circuit branch: when propose hands over an empty diff_path (the
// insufficient-data signal), no row is inserted and the status artifact
// reports status='skipped'.
func TestPipelineEvolveRecordSkipsInsufficientData(t *testing.T) {
	if _, err := exec.LookPath("sqlite3"); err != nil {
		t.Skipf("sqlite3 CLI not available: %v", err)
	}
	if _, err := exec.LookPath("jq"); err != nil {
		t.Skipf("jq CLI not available: %v", err)
	}

	tmpRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpRoot, "wave.yaml"), []byte("kind: WaveProject\n"), 0644); err != nil {
		t.Fatalf("write wave.yaml: %v", err)
	}
	dbDir := filepath.Join(tmpRoot, ".agents")
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	dbPath := filepath.Join(dbDir, "state.db")
	store, err := state.NewStateStore(dbPath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	_ = store.Close()

	wsDir := filepath.Join(tmpRoot, ".agents", "workspaces", "pipeline-evolve", "record")
	depDir := filepath.Join(wsDir, ".agents", "artifacts", "propose")
	if err := os.MkdirAll(depDir, 0755); err != nil {
		t.Fatalf("mkdir dep: %v", err)
	}
	summary := map[string]any{
		"pipeline_name":       "absent-pipeline",
		"candidate_yaml_path": "",
		"diff_path":           "",
		"reason":              "insufficient evaluation data; no proposal generated",
		"signal_summary":      map[string]any{"sample_size": 0, "failure_classes": []any{}},
		"version_before":      0,
		"version_after":       1,
	}
	summaryBytes, _ := json.Marshal(summary)
	summaryPath := filepath.Join(depDir, "proposal-summary")
	if err := os.WriteFile(summaryPath, summaryBytes, 0644); err != nil {
		t.Fatalf("write summary: %v", err)
	}

	repoRoot, _ := filepath.Abs(filepath.Join("..", ".."))
	loader := &YAMLPipelineLoader{}
	p, err := loader.Load(filepath.Join(repoRoot, "internal", "defaults", "embedfs", "pipelines", "pipeline-evolve.yaml"))
	if err != nil {
		t.Fatalf("load pipeline: %v", err)
	}
	var script string
	for _, s := range p.Steps {
		if s.ID == "record" {
			script = s.Script
			break
		}
	}
	ctx := NewPipelineContext("test-run", "pipeline-evolve", "record")
	ctx.Input = "absent-pipeline"
	script = ctx.ResolvePlaceholders(script)

	cmd := exec.Command("sh", "-c", script)
	cmd.Dir = wsDir
	cmd.Env = append(os.Environ(), "WAVE_DEP_PROPOSE_PROPOSAL_SUMMARY="+summaryPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("record script failed: %v\noutput:\n%s", err, out)
	}

	statusData, err := os.ReadFile(filepath.Join(wsDir, ".agents", "output", "record-status.json"))
	if err != nil {
		t.Fatalf("read status: %v", err)
	}
	var status struct {
		Status     string `json:"status"`
		ProposalID *int64 `json:"proposal_id"`
		SkipReason string `json:"skip_reason"`
	}
	if err := json.Unmarshal(statusData, &status); err != nil {
		t.Fatalf("parse status: %v", err)
	}
	if status.Status != "skipped" {
		t.Errorf("status = %q; want skipped\nscript output:\n%s", status.Status, out)
	}
	if status.ProposalID != nil {
		t.Errorf("proposal_id = %v; want nil for skipped", status.ProposalID)
	}

	// The DB must remain empty — the short-circuit branch must not insert
	// a noise row.
	store2, err := state.NewStateStore(dbPath)
	if err != nil {
		t.Fatalf("reopen store: %v", err)
	}
	defer store2.Close()
	rows, err := store2.ListProposalsByStatus(state.ProposalProposed, 0)
	if err != nil {
		t.Fatalf("list proposals: %v", err)
	}
	if len(rows) != 0 {
		t.Errorf("proposed rows = %d; want 0 on insufficient-data short-circuit", len(rows))
	}
}

// TestPipelineEvolveGatherEvalReadsSeededRows seeds pipeline_eval rows and
// runs the gather-eval bash script directly to verify the eval-rollup.json
// shape (sample_size, status, rows) the analyze step depends on.
func TestPipelineEvolveGatherEvalReadsSeededRows(t *testing.T) {
	if _, err := exec.LookPath("sqlite3"); err != nil {
		t.Skipf("sqlite3 CLI not available: %v", err)
	}
	if _, err := exec.LookPath("jq"); err != nil {
		t.Skipf("jq CLI not available: %v", err)
	}

	tmpRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpRoot, "wave.yaml"), []byte("kind: WaveProject\n"), 0644); err != nil {
		t.Fatalf("write wave.yaml: %v", err)
	}
	dbDir := filepath.Join(tmpRoot, ".agents")
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	dbPath := filepath.Join(dbDir, "state.db")
	store, err := state.NewStateStore(dbPath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	pipelineName := "demo-pipe"
	score := 0.8
	pass := true
	for i := 0; i < 3; i++ {
		if err := store.RecordEval(state.PipelineEvalRecord{
			PipelineName: pipelineName,
			RunID:        "r-" + string(rune('a'+i)),
			JudgeScore:   &score,
			ContractPass: &pass,
			RecordedAt:   time.Now().Add(time.Duration(-i) * time.Minute),
		}); err != nil {
			t.Fatalf("seed eval %d: %v", i, err)
		}
	}
	_ = store.Close()

	wsDir := filepath.Join(tmpRoot, ".agents", "workspaces", "pipeline-evolve", "gather-eval")
	if err := os.MkdirAll(wsDir, 0755); err != nil {
		t.Fatalf("mkdir ws: %v", err)
	}

	repoRoot, _ := filepath.Abs(filepath.Join("..", ".."))
	loader := &YAMLPipelineLoader{}
	p, err := loader.Load(filepath.Join(repoRoot, "internal", "defaults", "embedfs", "pipelines", "pipeline-evolve.yaml"))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	var script string
	for _, s := range p.Steps {
		if s.ID == "gather-eval" {
			script = s.Script
			break
		}
	}
	ctx := NewPipelineContext("test-run", "pipeline-evolve", "gather-eval")
	ctx.Input = pipelineName
	script = ctx.ResolvePlaceholders(script)

	cmd := exec.Command("sh", "-c", script)
	cmd.Dir = wsDir
	cmd.Env = os.Environ()
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("gather-eval failed: %v\noutput:\n%s", err, out)
	}

	rollupBytes, err := os.ReadFile(filepath.Join(wsDir, ".agents", "output", "eval-rollup.json"))
	if err != nil {
		t.Fatalf("read rollup: %v", err)
	}
	var rollup struct {
		PipelineName string `json:"pipeline_name"`
		SampleSize   int    `json:"sample_size"`
		Status       string `json:"status"`
		Rows         []any  `json:"rows"`
	}
	if err := json.Unmarshal(rollupBytes, &rollup); err != nil {
		t.Fatalf("parse rollup: %v\ncontent: %s", err, rollupBytes)
	}
	if rollup.PipelineName != pipelineName {
		t.Errorf("pipeline_name = %q; want %q", rollup.PipelineName, pipelineName)
	}
	if rollup.SampleSize != 3 || len(rollup.Rows) != 3 {
		t.Errorf("sample_size=%d rows=%d; want 3 / 3", rollup.SampleSize, len(rollup.Rows))
	}
	if rollup.Status != "ready" {
		t.Errorf("status = %q; want ready", rollup.Status)
	}
}
