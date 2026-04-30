package contract

import (
	"sync"
	"testing"
	"time"
)

func TestSignalKind_StringValues(t *testing.T) {
	cases := []struct {
		kind SignalKind
		want string
	}{
		{SignalSuccess, "success"},
		{SignalFailure, "failure"},
		{SignalContractFailure, "contract_failure"},
		{SignalJudgeScore, "judge_score"},
		{SignalDuration, "duration"},
		{SignalCost, "cost"},
	}
	for _, c := range cases {
		if string(c.kind) != c.want {
			t.Errorf("SignalKind(%q) round-trip = %q, want %q", c.kind, string(c.kind), c.want)
		}
	}
}

func TestSignalSet_FailureClassPriority(t *testing.T) {
	cases := []struct {
		name    string
		signals []Signal
		want    string
	}{
		{
			name:    "empty set",
			signals: nil,
			want:    "",
		},
		{
			name:    "all-success",
			signals: []Signal{{Kind: SignalSuccess, StepID: "s1"}, {Kind: SignalSuccess, StepID: "s2"}},
			want:    "",
		},
		{
			name:    "single failure",
			signals: []Signal{{Kind: SignalFailure, StepID: "s1"}},
			want:    "failure",
		},
		{
			name: "contract_failure beats failure",
			signals: []Signal{
				{Kind: SignalFailure, StepID: "s1"},
				{Kind: SignalContractFailure, StepID: "s2"},
			},
			want: "contract_failure",
		},
		{
			name: "contract_failure first",
			signals: []Signal{
				{Kind: SignalContractFailure, StepID: "s1"},
				{Kind: SignalFailure, StepID: "s2"},
			},
			want: "contract_failure",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			set := NewSignalSet()
			for _, s := range tc.signals {
				set.Add(s)
			}
			got := set.FailureClass()
			if got != tc.want {
				t.Errorf("FailureClass() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestSignalSet_Aggregate_EmptySet(t *testing.T) {
	set := NewSignalSet()
	rec := set.Aggregate("run-1", "test-pipeline", time.Time{})

	if rec.PipelineName != "test-pipeline" || rec.RunID != "run-1" {
		t.Errorf("identity fields wrong: name=%q run=%q", rec.PipelineName, rec.RunID)
	}
	if rec.JudgeScore != nil {
		t.Errorf("empty set should not produce JudgeScore, got %v", *rec.JudgeScore)
	}
	if rec.ContractPass != nil {
		t.Errorf("empty set should leave ContractPass nil, got %v", *rec.ContractPass)
	}
	if rec.RetryCount != nil {
		t.Errorf("empty set should leave RetryCount nil, got %v", *rec.RetryCount)
	}
	if rec.FailureClass != "" {
		t.Errorf("empty set FailureClass = %q, want empty", rec.FailureClass)
	}
	if rec.RecordedAt.IsZero() {
		t.Error("RecordedAt should default to time.Now()")
	}
}

func TestSignalSet_Aggregate_AllSuccess(t *testing.T) {
	set := NewSignalSet()
	set.Add(Signal{Kind: SignalSuccess, StepID: "s1"})
	set.Add(Signal{Kind: SignalSuccess, StepID: "s2"})

	startedAt := time.Now().Add(-2 * time.Second)
	rec := set.Aggregate("run-1", "p", startedAt)

	if rec.ContractPass == nil || !*rec.ContractPass {
		t.Errorf("all-success should yield ContractPass=true, got %v", rec.ContractPass)
	}
	if rec.FailureClass != "" {
		t.Errorf("all-success FailureClass = %q, want empty", rec.FailureClass)
	}
	if rec.DurationMs == nil || *rec.DurationMs <= 0 {
		t.Errorf("DurationMs should be positive, got %v", rec.DurationMs)
	}
}

func TestSignalSet_Aggregate_ContractFailure(t *testing.T) {
	set := NewSignalSet()
	set.Add(Signal{Kind: SignalSuccess, StepID: "s1"})
	set.Add(Signal{Kind: SignalContractFailure, StepID: "s2"})

	rec := set.Aggregate("run-1", "p", time.Now())

	if rec.ContractPass == nil || *rec.ContractPass {
		t.Errorf("contract_failure should yield ContractPass=false, got %v", rec.ContractPass)
	}
	if rec.FailureClass != "contract_failure" {
		t.Errorf("FailureClass = %q, want contract_failure", rec.FailureClass)
	}
}

func TestSignalSet_Aggregate_JudgeScoreAverage(t *testing.T) {
	set := NewSignalSet()
	set.Add(Signal{Kind: SignalJudgeScore, StepID: "s1", Value: 0.6})
	set.Add(Signal{Kind: SignalJudgeScore, StepID: "s2", Value: 0.8})
	set.Add(Signal{Kind: SignalJudgeScore, StepID: "s3", Value: 1.0})

	rec := set.Aggregate("run-1", "p", time.Now())

	if rec.JudgeScore == nil {
		t.Fatal("JudgeScore should not be nil")
	}
	got := *rec.JudgeScore
	want := (0.6 + 0.8 + 1.0) / 3.0
	if got < want-1e-9 || got > want+1e-9 {
		t.Errorf("JudgeScore avg = %v, want %v", got, want)
	}
}

func TestSignalSet_Aggregate_RetryCount(t *testing.T) {
	set := NewSignalSet()
	set.Add(Signal{Kind: SignalSuccess, StepID: "s1"})
	set.RecordRetry()
	set.RecordRetry()
	set.RecordRetry()

	rec := set.Aggregate("run-1", "p", time.Now())

	if rec.RetryCount == nil {
		t.Fatal("RetryCount should not be nil after RecordRetry calls")
	}
	if *rec.RetryCount != 3 {
		t.Errorf("RetryCount = %d, want 3", *rec.RetryCount)
	}
}

func TestSignalSet_Add_Concurrent(t *testing.T) {
	set := NewSignalSet()
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			set.Add(Signal{Kind: SignalSuccess, StepID: "s"})
		}()
	}
	wg.Wait()
	if set.Len() != 100 {
		t.Errorf("Len after concurrent Add = %d, want 100", set.Len())
	}
}

func TestSignalSet_Aggregate_Timestamps(t *testing.T) {
	set := NewSignalSet()
	set.Add(Signal{Kind: SignalSuccess, StepID: "s1"})

	rec := set.Aggregate("run-1", "p", time.Time{})

	if rec.DurationMs != nil {
		t.Errorf("zero startedAt should not produce DurationMs, got %v", *rec.DurationMs)
	}
}
