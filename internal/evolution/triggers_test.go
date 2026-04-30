package evolution

import (
	"errors"
	"testing"
	"time"

	"github.com/recinq/wave/internal/state"
)

// fakeStore is a minimal Store implementation that returns the configured
// rows verbatim. Tests assemble synthetic eval slices without touching
// SQLite.
type fakeStore struct {
	evals        []state.PipelineEvalRecord
	lastProposal time.Time
	hasProposal  bool
	getErr       error
	proposalErr  error
}

func (f *fakeStore) GetEvalsForPipeline(_ string, _ int) ([]state.PipelineEvalRecord, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	out := make([]state.PipelineEvalRecord, len(f.evals))
	copy(out, f.evals)
	return out, nil
}

func (f *fakeStore) LastProposalAt(_ string) (time.Time, bool, error) {
	if f.proposalErr != nil {
		return time.Time{}, false, f.proposalErr
	}
	return f.lastProposal, f.hasProposal, nil
}

// row builds a PipelineEvalRecord with the supplied judge_score, contract_pass,
// and retry_count. Use nil pointers to mark missing values. recordedAt is set
// to the supplied offset; tests use this to put rows in newest-first order.
func row(t time.Time, score *float64, pass *bool, retries *int) state.PipelineEvalRecord {
	return state.PipelineEvalRecord{
		PipelineName: "p",
		RunID:        "r",
		JudgeScore:   score,
		ContractPass: pass,
		RetryCount:   retries,
		RecordedAt:   t,
	}
}

func ptrFloat(v float64) *float64 { return &v }
func ptrBool(v bool) *bool         { return &v }
func ptrInt(v int) *int            { return &v }

// scoredRows returns n eval rows newest-first with the supplied judge score.
// Each row is one second older than the previous so RecordedAt ordering is
// stable.
func scoredRows(n int, score float64, base time.Time) []state.PipelineEvalRecord {
	out := make([]state.PipelineEvalRecord, n)
	for i := 0; i < n; i++ {
		out[i] = row(base.Add(-time.Duration(i)*time.Second), ptrFloat(score), nil, nil)
	}
	return out
}

func TestShouldEvolve_DisabledShortCircuits(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = false
	svc := NewService(&fakeStore{evals: scoredRows(50, 0.1, time.Now())}, cfg)
	fire, reason, err := svc.ShouldEvolve("p")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if fire || reason != "" {
		t.Fatalf("disabled should not fire, got fire=%v reason=%q", fire, reason)
	}
}

func TestShouldEvolve_NilStoreReturnsFalse(t *testing.T) {
	svc := NewService(nil, DefaultConfig())
	fire, _, err := svc.ShouldEvolve("p")
	if err != nil || fire {
		t.Fatalf("nil store should be a no-op, got fire=%v err=%v", fire, err)
	}
}

func TestShouldEvolve_EmptyPipelineNameReturnsFalse(t *testing.T) {
	svc := NewService(&fakeStore{}, DefaultConfig())
	fire, _, err := svc.ShouldEvolve("")
	if err != nil || fire {
		t.Fatalf("empty name no-op, got fire=%v err=%v", fire, err)
	}
}

func TestShouldEvolve_StoreErrorPropagates(t *testing.T) {
	svc := NewService(&fakeStore{getErr: errors.New("db down")}, DefaultConfig())
	_, _, err := svc.ShouldEvolve("p")
	if err == nil {
		t.Fatal("expected error from store")
	}
}

func TestShouldEvolve_ProposalErrorPropagates(t *testing.T) {
	svc := NewService(&fakeStore{
		evals:       scoredRows(1, 0.5, time.Now()),
		proposalErr: errors.New("boom"),
	}, DefaultConfig())
	_, _, err := svc.ShouldEvolve("p")
	if err == nil {
		t.Fatal("expected error from LastProposalAt")
	}
}

// TestEveryNJudgeDrop_FiresOnMedianDrop seeds 10 recent low-score rows and
// 10 prior high-score rows; median drop > 0.1 should fire.
func TestEveryNJudgeDrop_FiresOnMedianDrop(t *testing.T) {
	now := time.Now()
	recent := scoredRows(10, 0.5, now)
	prior := scoredRows(10, 0.9, now.Add(-20*time.Second))
	all := append([]state.PipelineEvalRecord{}, recent...)
	all = append(all, prior...)

	svc := NewService(&fakeStore{evals: all}, DefaultConfig())
	fire, reason, err := svc.ShouldEvolve("p")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !fire {
		t.Fatalf("expected every-N to fire, reason=%q", reason)
	}
	if reason == "" {
		t.Fatal("expected non-empty reason")
	}
}

// TestEveryNJudgeDrop_NoFireWhenStable seeds two stable windows; no drop →
// no fire.
func TestEveryNJudgeDrop_NoFireWhenStable(t *testing.T) {
	now := time.Now()
	all := append(scoredRows(10, 0.85, now), scoredRows(10, 0.85, now.Add(-20*time.Second))...)
	svc := NewService(&fakeStore{evals: all}, DefaultConfig())
	fire, _, err := svc.ShouldEvolve("p")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if fire {
		t.Fatal("stable scores should not fire every-N")
	}
}

// TestEveryNJudgeDrop_RespectsLastProposal seeds rows older than the last
// proposal; they must be filtered out so the heuristic has insufficient
// data and does not fire.
func TestEveryNJudgeDrop_RespectsLastProposal(t *testing.T) {
	base := time.Now()
	recent := scoredRows(10, 0.5, base)
	prior := scoredRows(10, 0.9, base.Add(-20*time.Second))
	all := append([]state.PipelineEvalRecord{}, recent...)
	all = append(all, prior...)
	// Last proposal recorded "now" — so all 20 rows are at or before the
	// proposal timestamp and must be filtered out.
	svc := NewService(&fakeStore{
		evals:        all,
		lastProposal: base.Add(time.Second),
		hasProposal:  true,
	}, DefaultConfig())
	fire, _, err := svc.ShouldEvolve("p")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if fire {
		t.Fatal("every-N must filter rows recorded before last proposal")
	}
}

// TestEveryNJudgeDrop_InsufficientData provides fewer than 2*window rows;
// must not fire.
func TestEveryNJudgeDrop_InsufficientData(t *testing.T) {
	all := scoredRows(15, 0.4, time.Now()) // need 20
	svc := NewService(&fakeStore{evals: all}, DefaultConfig())
	fire, _, _ := svc.ShouldEvolve("p")
	if fire {
		t.Fatal("insufficient data must not fire every-N")
	}
}

// TestContractPassDrift_FiresOnDrop seeds 20 recent failing rows and 20
// prior passing rows; drift > 15% should fire.
func TestContractPassDrift_FiresOnDrop(t *testing.T) {
	cfg := DefaultConfig()
	// Disable every-N so it can't pre-empt drift in the OR composition.
	cfg.EveryNWindow = 0
	cfg.RetryWindow = 0

	now := time.Now()
	recent := make([]state.PipelineEvalRecord, 20)
	for i := range recent {
		recent[i] = row(now.Add(-time.Duration(i)*time.Second), nil, ptrBool(false), nil)
	}
	prior := make([]state.PipelineEvalRecord, 20)
	for i := range prior {
		prior[i] = row(now.Add(-time.Duration(20+i)*time.Second), nil, ptrBool(true), nil)
	}
	all := make([]state.PipelineEvalRecord, 0, len(recent)+len(prior))
	all = append(all, recent...)
	all = append(all, prior...)
	svc := NewService(&fakeStore{evals: all}, cfg)
	fire, reason, err := svc.ShouldEvolve("p")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !fire {
		t.Fatalf("drift should fire, reason=%q", reason)
	}
}

// TestContractPassDrift_NoFireWhenStable: equal pass rates → no drift.
func TestContractPassDrift_NoFireWhenStable(t *testing.T) {
	cfg := DefaultConfig()
	cfg.EveryNWindow = 0
	cfg.RetryWindow = 0
	now := time.Now()
	all := make([]state.PipelineEvalRecord, 40)
	for i := range all {
		all[i] = row(now.Add(-time.Duration(i)*time.Second), nil, ptrBool(true), nil)
	}
	svc := NewService(&fakeStore{evals: all}, cfg)
	fire, _, _ := svc.ShouldEvolve("p")
	if fire {
		t.Fatal("stable pass rate must not fire drift")
	}
}

// TestContractPassDrift_InsufficientData: fewer than 2*DriftWindow rows with
// non-nil ContractPass → no fire.
func TestContractPassDrift_InsufficientData(t *testing.T) {
	cfg := DefaultConfig()
	cfg.EveryNWindow = 0
	cfg.RetryWindow = 0
	now := time.Now()
	all := make([]state.PipelineEvalRecord, 30) // need 40
	for i := range all {
		all[i] = row(now.Add(-time.Duration(i)*time.Second), nil, ptrBool(false), nil)
	}
	svc := NewService(&fakeStore{evals: all}, cfg)
	fire, _, _ := svc.ShouldEvolve("p")
	if fire {
		t.Fatal("insufficient data must not fire drift")
	}
}

// TestRetryRateSpike_Fires: avg retries = 3 over window of 10; threshold 2.0.
func TestRetryRateSpike_Fires(t *testing.T) {
	cfg := DefaultConfig()
	cfg.EveryNWindow = 0
	cfg.DriftWindow = 0
	now := time.Now()
	all := make([]state.PipelineEvalRecord, 10)
	for i := range all {
		all[i] = row(now.Add(-time.Duration(i)*time.Second), nil, nil, ptrInt(3))
	}
	svc := NewService(&fakeStore{evals: all}, cfg)
	fire, reason, err := svc.ShouldEvolve("p")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !fire {
		t.Fatalf("retry-rate should fire, reason=%q", reason)
	}
}

// TestRetryRateSpike_NoFireBelowThreshold: avg = 1.5 over window of 10.
func TestRetryRateSpike_NoFireBelowThreshold(t *testing.T) {
	cfg := DefaultConfig()
	cfg.EveryNWindow = 0
	cfg.DriftWindow = 0
	now := time.Now()
	all := make([]state.PipelineEvalRecord, 10)
	for i := range all {
		retries := i % 4 // 0,1,2,3 cycles → avg 1.5
		all[i] = row(now.Add(-time.Duration(i)*time.Second), nil, nil, ptrInt(retries))
	}
	svc := NewService(&fakeStore{evals: all}, cfg)
	fire, _, _ := svc.ShouldEvolve("p")
	if fire {
		t.Fatal("avg below threshold must not fire retry-rate")
	}
}

// TestRetryRateSpike_InsufficientData: fewer than RetryWindow rows.
func TestRetryRateSpike_InsufficientData(t *testing.T) {
	cfg := DefaultConfig()
	cfg.EveryNWindow = 0
	cfg.DriftWindow = 0
	all := make([]state.PipelineEvalRecord, 5)
	now := time.Now()
	for i := range all {
		all[i] = row(now.Add(-time.Duration(i)*time.Second), nil, nil, ptrInt(10))
	}
	svc := NewService(&fakeStore{evals: all}, cfg)
	fire, _, _ := svc.ShouldEvolve("p")
	if fire {
		t.Fatal("insufficient data must not fire retry-rate")
	}
}

// TestComposition_EveryNTakesPrecedence: data triggers all three; first match
// must be every-N because heuristics evaluate in defined order.
//
// Layout (newest → oldest):
//
//	rows 0..9   — recent window: low judge, contract fail, high retries.
//	rows 10..19 — prior window:  high judge, contract pass, no retries.
//	rows 20..39 — older drift/retry padding so contract-pass drift would
//	              also fire on its own.
func TestComposition_EveryNTakesPrecedence(t *testing.T) {
	now := time.Now()
	all := make([]state.PipelineEvalRecord, 0, 40)
	// Recent every-N half: low judge.
	for i := 0; i < 10; i++ {
		all = append(all, row(
			now.Add(-time.Duration(i)*time.Second),
			ptrFloat(0.4),
			ptrBool(false),
			ptrInt(3),
		))
	}
	// Prior every-N half: high judge.
	for i := 0; i < 10; i++ {
		all = append(all, row(
			now.Add(-time.Duration(10+i)*time.Second),
			ptrFloat(0.95),
			ptrBool(false),
			ptrInt(3),
		))
	}
	// Older drift padding: passing contracts so drift sees a 100% prior rate.
	for i := 0; i < 20; i++ {
		all = append(all, row(
			now.Add(-time.Duration(20+i)*time.Second),
			ptrFloat(0.95),
			ptrBool(true),
			ptrInt(0),
		))
	}
	svc := NewService(&fakeStore{evals: all}, DefaultConfig())
	fire, reason, _ := svc.ShouldEvolve("p")
	if !fire {
		t.Fatal("composition should fire")
	}
	if len(reason) < 7 || reason[:7] != "every-N" {
		t.Fatalf("expected every-N precedence, got %q", reason)
	}
}

// TestCustomConfig_Override: tighten thresholds so a previously-non-firing
// dataset now fires.
func TestCustomConfig_Override(t *testing.T) {
	now := time.Now()
	all := append(scoredRows(10, 0.85, now), scoredRows(10, 0.90, now.Add(-20*time.Second))...)

	// Default judge_drop = 0.1 → median moves only 0.05; no fire.
	svc := NewService(&fakeStore{evals: all}, DefaultConfig())
	if fire, _, _ := svc.ShouldEvolve("p"); fire {
		t.Fatal("default config should not fire on 0.05 drop")
	}

	// Tighten threshold to 0.04 → now fires.
	cfg := DefaultConfig()
	cfg.EveryNJudgeDrop = 0.04
	svc = NewService(&fakeStore{evals: all}, cfg)
	if fire, _, _ := svc.ShouldEvolve("p"); !fire {
		t.Fatal("tightened threshold should fire")
	}
}

// TestYAMLOverridesApply confirms partial overrides retain defaults.
func TestYAMLOverridesApply(t *testing.T) {
	enabled := false
	o := YAMLOverrides{
		Enabled:           &enabled,
		EveryNWindow:      5,
		RetryAvgThreshold: 7,
	}
	cfg := o.Apply()
	if cfg.Enabled {
		t.Fatal("Enabled override ignored")
	}
	if cfg.EveryNWindow != 5 {
		t.Fatalf("EveryNWindow = %d, want 5", cfg.EveryNWindow)
	}
	if cfg.RetryAvgThreshold != 7 {
		t.Fatalf("RetryAvgThreshold = %v, want 7", cfg.RetryAvgThreshold)
	}
	if cfg.DriftWindow != DefaultConfig().DriftWindow {
		t.Fatalf("unset DriftWindow not defaulted; got %d", cfg.DriftWindow)
	}
}

// TestYAMLOverrides_NilApply returns DefaultConfig untouched.
func TestYAMLOverrides_NilApply(t *testing.T) {
	var o *YAMLOverrides
	got := o.Apply()
	want := DefaultConfig()
	if got != want {
		t.Fatalf("nil overrides changed defaults: %+v", got)
	}
}

// TestMedianFloats covers even/odd lengths and stable input ordering.
func TestMedianFloats(t *testing.T) {
	cases := []struct {
		name string
		in   []float64
		want float64
	}{
		{"empty", nil, 0},
		{"one", []float64{0.5}, 0.5},
		{"odd", []float64{0.1, 0.3, 0.2}, 0.2},
		{"even", []float64{0.4, 0.2, 0.6, 0.8}, 0.5},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := medianFloats(tc.in)
			if got != tc.want {
				t.Fatalf("medianFloats(%v) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}
