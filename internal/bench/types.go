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
	// Instance is the git ref or version to check out.
	Instance string `json:"version"`
	// Problem is the natural-language problem statement given to the pipeline.
	Problem string `json:"problem_statement"`
	// ExpectedPatch is the gold-standard patch (unified diff) for validation.
	ExpectedPatch string `json:"patch"`
	// TestCommand is the shell command used to verify correctness.
	TestCommand string `json:"test_cmd"`
}

// BenchResult records the outcome of running a pipeline against one task.
type BenchResult struct {
	TaskID     string      `json:"task_id"`
	RunID      string      `json:"run_id"`
	Pipeline   string      `json:"pipeline"`
	Status     BenchStatus `json:"status"`
	DurationMs int64       `json:"duration_ms"`
	TokensUsed int64       `json:"tokens_used,omitempty"`
	PatchDiff  string      `json:"patch_diff,omitempty"`
	Error      string      `json:"error,omitempty"`
	StartedAt  time.Time   `json:"started_at"`
}

// BenchReport aggregates the results of a full benchmark run.
type BenchReport struct {
	Dataset  string        `json:"dataset"`
	Pipeline string        `json:"pipeline"`
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
