package bench

import "time"

// BenchStatus represents the outcome of a single benchmark task.
type BenchStatus string

const (
	StatusPass  BenchStatus = "pass"
	StatusFail  BenchStatus = "fail"
	StatusError BenchStatus = "error"
)

// BenchTask describes a single SWE-bench task to be solved by a pipeline.
type BenchTask struct {
	// ID is a unique identifier for this task (e.g. "django__django-16379").
	ID string `json:"instance_id"`
	// Repo is the repository slug (e.g. "django/django").
	Repo string `json:"repo"`
	// BaseCommit is the git commit to check out before applying the fix.
	BaseCommit string `json:"base_commit"`
	// Problem is the natural-language problem statement given to the pipeline.
	Problem string `json:"problem_statement"`
	// TestPatch is the test diff to apply for verification.
	TestPatch string `json:"test_patch"`
	// TestCommand is the shell command used to verify correctness.
	TestCommand string `json:"test_cmd"`
}

// BenchResult records the outcome of running a pipeline against one task.
type BenchResult struct {
	TaskID     string      `json:"task_id"`
	RunID      string      `json:"run_id"`
	Pipeline   string      `json:"pipeline"`
	Status     BenchStatus `json:"status"`
	DurationMs int64  `json:"duration_ms"`
	PatchDiff  string `json:"patch_diff,omitempty"`
	Error      string      `json:"error,omitempty"`
	StartedAt  time.Time   `json:"started_at"`
}

// BenchReport aggregates the results of a full benchmark run.
type BenchReport struct {
	Dataset  string        `json:"dataset"`
	Pipeline string        `json:"pipeline"`
	Mode     string        `json:"mode,omitempty"`
	RunLabel string        `json:"run_label,omitempty"`
	Total    int           `json:"total"`
	Passed   int           `json:"passed"`
	Failed   int           `json:"failed"`
	Errors   int           `json:"errors"`
	PassRate float64       `json:"pass_rate"`
	Results  []BenchResult `json:"results"`
	// Metadata
	StartedAt   time.Time `json:"started_at"`
	CompletedAt time.Time `json:"completed_at"`
	DurationMs  int64     `json:"duration_ms"`
}

// Tally recalculates aggregate counts from Results.
func (r *BenchReport) Tally() {
	r.Total = len(r.Results)
	r.Passed = 0
	r.Failed = 0
	r.Errors = 0
	for _, res := range r.Results {
		switch res.Status {
		case StatusPass:
			r.Passed++
		case StatusFail:
			r.Failed++
		case StatusError:
			r.Errors++
		}
	}
	if r.Total > 0 {
		r.PassRate = float64(r.Passed) / float64(r.Total)
	}
}

// CompareReport holds the comparison between two benchmark runs.
type CompareReport struct {
	Base    ReportRef      `json:"base"`
	Compare ReportRef      `json:"compare"`
	Summary CompareSummary `json:"summary"`
	Diffs   []TaskDiff     `json:"diffs"`
}

// ReportRef identifies one side of a comparison.
type ReportRef struct {
	Pipeline string  `json:"pipeline"`
	Mode     string  `json:"mode,omitempty"`
	RunLabel string  `json:"run_label,omitempty"`
	Total    int     `json:"total"`
	Passed   int     `json:"passed"`
	PassRate float64 `json:"pass_rate"`
}

// CompareSummary provides aggregate comparison metrics.
type CompareSummary struct {
	Improved    int     `json:"improved"`
	Regressed   int     `json:"regressed"`
	Unchanged   int     `json:"unchanged"`
	OnlyInBase  int     `json:"only_in_base"`
	OnlyInComp  int     `json:"only_in_compare"`
	DeltaRate   float64 `json:"delta_rate"`
}

// TaskDiff describes how a single task's result changed between runs.
type TaskDiff struct {
	TaskID     string      `json:"task_id"`
	Change     string      `json:"change"` // "improved", "regressed", "unchanged", "only_base", "only_compare"
	BaseStatus BenchStatus `json:"base_status,omitempty"`
	CompStatus BenchStatus `json:"compare_status,omitempty"`
}
