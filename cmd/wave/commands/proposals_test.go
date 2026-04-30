package commands

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/recinq/wave/internal/proposals"
	"github.com/recinq/wave/internal/state"
)

// proposalsTestHelper holds the chdir-based scaffolding used by all CLI
// proposals tests. The CLI commands always read .agents/state.db relative
// to the current working directory, so each test must chdir to its tempdir.
type proposalsTestHelper struct {
	t       *testing.T
	tmpDir  string
	origDir string
	dbPath  string
}

func newProposalsTestHelper(t *testing.T) *proposalsTestHelper {
	t.Helper()
	tmpDir := t.TempDir()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(tmpDir, ".agents"), 0o755); err != nil {
		t.Fatalf("mkdir .agents: %v", err)
	}
	dbPath := filepath.Join(tmpDir, ".agents", "state.db")
	// Pre-create the DB so subsequent CLI invocations open the same migrated handle.
	store, err := state.NewStateStore(dbPath)
	if err != nil {
		t.Fatalf("NewStateStore: %v", err)
	}
	store.Close()

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(orig) })

	return &proposalsTestHelper{t: t, tmpDir: tmpDir, origDir: orig, dbPath: dbPath}
}

func (h *proposalsTestHelper) openStore() state.StateStore {
	h.t.Helper()
	store, err := state.NewStateStore(h.dbPath)
	if err != nil {
		h.t.Fatalf("open store: %v", err)
	}
	return store
}

// seedProposalCLI inserts a proposal and writes the matching post-diff yaml.
func (h *proposalsTestHelper) seedProposalCLI(pipelineName, yamlBody string) (int64, string) {
	h.t.Helper()
	diffPath := filepath.Join(h.tmpDir, "p"+pipelineName+".diff")
	if err := os.WriteFile(diffPath, []byte("--- a\n+++ b\n"), 0o600); err != nil {
		h.t.Fatalf("write diff: %v", err)
	}
	yamlPath := diffPath + ".after.yaml"
	if err := os.WriteFile(yamlPath, []byte(yamlBody), 0o600); err != nil {
		h.t.Fatalf("write yaml: %v", err)
	}
	store := h.openStore()
	defer store.Close()
	id, err := store.CreateProposal(state.EvolutionProposalRecord{
		PipelineName:  pipelineName,
		VersionBefore: 1,
		VersionAfter:  2,
		DiffPath:      diffPath,
		Reason:        "tighten contract",
		SignalSummary: `{"judge_score":0.78}`,
	})
	if err != nil {
		h.t.Fatalf("CreateProposal: %v", err)
	}
	return id, yamlPath
}

func (h *proposalsTestHelper) seedV1Active(pipelineName string) {
	h.t.Helper()
	store := h.openStore()
	defer store.Close()
	if err := store.CreatePipelineVersion(state.PipelineVersionRecord{
		PipelineName: pipelineName, Version: 1, SHA256: "sha-old",
		YAMLPath: "old.yaml", Active: true,
	}); err != nil {
		h.t.Fatalf("CreatePipelineVersion(v1): %v", err)
	}
}

func TestRunProposalsList_TextEmpty(t *testing.T) {
	_ = newProposalsTestHelper(t)
	if err := runProposalsList("proposed", "text"); err != nil {
		t.Fatalf("list: %v", err)
	}
}

func TestRunProposalsApprove_FlipsActive(t *testing.T) {
	h := newProposalsTestHelper(t)
	h.seedV1Active("impl-issue")
	id, yamlPath := h.seedProposalCLI("impl-issue", "version: 2\nsteps: []\n")

	if err := runProposalsApprove(strconv.FormatInt(id, 10), "ok"); err != nil {
		t.Fatalf("approve: %v", err)
	}

	store := h.openStore()
	defer store.Close()
	active, err := store.GetActiveVersion("impl-issue")
	if err != nil {
		t.Fatalf("GetActiveVersion: %v", err)
	}
	if active == nil || active.Version != 2 {
		t.Fatalf("expected active v2, got %+v", active)
	}
	if active.YAMLPath != yamlPath {
		t.Errorf("expected yaml_path=%s, got %s", yamlPath, active.YAMLPath)
	}
	versions, _ := store.ListPipelineVersions("impl-issue")
	if len(versions) != 2 {
		t.Errorf("expected 2 versions, got %d", len(versions))
	}
	for _, v := range versions {
		if v.Version == 1 && v.Active {
			t.Errorf("v1 should be deactivated after approve")
		}
	}
}

func TestRunProposalsReject_LeavesVersions(t *testing.T) {
	h := newProposalsTestHelper(t)
	h.seedV1Active("scope")
	id, _ := h.seedProposalCLI("scope", "v2 yaml\n")

	if err := runProposalsReject(strconv.FormatInt(id, 10), "scope creep"); err != nil {
		t.Fatalf("reject: %v", err)
	}
	store := h.openStore()
	defer store.Close()
	rec, err := store.GetProposal(id)
	if err != nil {
		t.Fatalf("GetProposal: %v", err)
	}
	if rec == nil || rec.Status != state.ProposalRejected {
		t.Fatalf("expected rejected, got %+v", rec)
	}
	versions, _ := store.ListPipelineVersions("scope")
	if len(versions) != 1 || versions[0].Version != 1 {
		t.Errorf("reject should leave versions unchanged, got %+v", versions)
	}
	active, _ := store.GetActiveVersion("scope")
	if active == nil || active.Version != 1 {
		t.Errorf("expected v1 still active, got %+v", active)
	}
}

func TestRunProposalsReject_RequiresReason(t *testing.T) {
	h := newProposalsTestHelper(t)
	id, _ := h.seedProposalCLI("scope", "v2\n")
	err := runProposalsReject(strconv.FormatInt(id, 10), "")
	if err == nil {
		t.Fatal("expected error for missing reason")
	}
}

func TestRunProposalsApprove_AlreadyDecided(t *testing.T) {
	h := newProposalsTestHelper(t)
	h.seedV1Active("impl-issue")
	id, _ := h.seedProposalCLI("impl-issue", "v2\n")

	if err := runProposalsApprove(strconv.FormatInt(id, 10), ""); err != nil {
		t.Fatalf("first approve: %v", err)
	}
	err := runProposalsApprove(strconv.FormatInt(id, 10), "")
	if err == nil {
		t.Fatal("expected error on second approve of same proposal")
	}
}

func TestRunProposalsShow_NotFound(t *testing.T) {
	_ = newProposalsTestHelper(t)
	err := runProposalsShow("9999", "text")
	if err == nil {
		t.Fatal("expected error for missing proposal")
	}
}

func TestApproveErrorMapping(t *testing.T) {
	cases := []struct {
		name string
		in   error
	}{
		{"already-decided", proposals.ErrAlreadyDecided},
		{"version-conflict", proposals.ErrVersionConflict},
		{"after-yaml-missing", proposals.ErrAfterYAMLMissing},
		{"unknown", errors.New("boom")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := mapApproveError(tc.in)
			if err == nil {
				t.Fatal("expected non-nil")
			}
			var cliErr *CLIError
			if !errors.As(err, &cliErr) {
				t.Fatalf("expected *CLIError, got %T", err)
			}
		})
	}
}

// TestAcceptanceGate is the end-to-end gate from the spec: synthetic proposal
// → CLI approve → GetActiveVersion returns the new row → loader resolves the
// new yaml_path on a subsequent lookup.
func TestAcceptanceGate(t *testing.T) {
	h := newProposalsTestHelper(t)
	h.seedV1Active("impl-issue")
	id, yamlPath := h.seedProposalCLI("impl-issue", "version: 2\nsteps:\n  - id: noop\n")

	// 1. List confirms the proposal is visible.
	if err := runProposalsList("proposed", "text"); err != nil {
		t.Fatalf("list: %v", err)
	}

	// 2. Approve flips active.
	if err := runProposalsApprove(strconv.FormatInt(id, 10), "acceptance"); err != nil {
		t.Fatalf("approve: %v", err)
	}

	// 3. GetActiveVersion now returns v2 with the new yaml_path — what the
	//    pipeline loader queries before each `wave run`.
	store := h.openStore()
	defer store.Close()
	active, err := store.GetActiveVersion("impl-issue")
	if err != nil {
		t.Fatalf("GetActiveVersion: %v", err)
	}
	if active == nil {
		t.Fatal("expected active version, got nil")
	}
	if active.Version != 2 {
		t.Errorf("expected active v2, got v%d", active.Version)
	}
	if active.YAMLPath != yamlPath {
		t.Errorf("expected yaml_path=%s, got %s", yamlPath, active.YAMLPath)
	}
	if active.SHA256 == "" {
		t.Error("expected non-empty sha256")
	}

	// 4. The yaml file pointed at by the active row exists and contains the
	//    payload pipeline-evolve emitted — the loader will read it next.
	body, err := os.ReadFile(active.YAMLPath)
	if err != nil {
		t.Fatalf("read active yaml: %v", err)
	}
	if string(body) != "version: 2\nsteps:\n  - id: noop\n" {
		t.Errorf("yaml body mismatch: %q", string(body))
	}
}
