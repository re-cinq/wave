package bench

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Prediction is one line of a SWE-bench predictions file, the input format of
// the official evaluation harness (swebench.harness.run_evaluation).
type Prediction struct {
	InstanceID      string `json:"instance_id"`
	ModelNameOrPath string `json:"model_name_or_path"`
	ModelPatch      string `json:"model_patch"`
}

// Predictions converts a report into SWE-bench predictions. Every task is
// included; tasks without a patch get an empty model_patch so the harness
// counts them as unresolved instead of silently dropping them.
func Predictions(report *BenchReport, modelName string) []Prediction {
	preds := make([]Prediction, 0, len(report.Results))
	for _, r := range report.Results {
		preds = append(preds, Prediction{
			InstanceID:      r.TaskID,
			ModelNameOrPath: modelName,
			ModelPatch:      r.PatchDiff,
		})
	}
	return preds
}

// WritePredictions writes predictions as JSONL.
func WritePredictions(path string, preds []Prediction) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(f)
	for _, p := range preds {
		if err := enc.Encode(p); err != nil {
			_ = f.Close()
			return err
		}
	}
	return f.Close()
}

// HarnessReport is the subset of the SWE-bench harness summary report
// (<model>.<run_id>.json) that carries per-instance verdicts.
type HarnessReport struct {
	ResolvedIDs   []string `json:"resolved_ids"`
	UnresolvedIDs []string `json:"unresolved_ids"`
	EmptyPatchIDs []string `json:"empty_patch_ids"`
	ErrorIDs      []string `json:"error_ids"`
	IncompleteIDs []string `json:"incomplete_ids"`
}

// LoadHarnessReport reads a SWE-bench harness summary report.
func LoadHarnessReport(path string) (*HarnessReport, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read harness report %s: %w", path, err)
	}
	var hr HarnessReport
	if err := json.Unmarshal(data, &hr); err != nil {
		return nil, fmt.Errorf("parse harness report %s: %w", path, err)
	}
	if hr.ResolvedIDs == nil && hr.UnresolvedIDs == nil && hr.EmptyPatchIDs == nil && hr.ErrorIDs == nil {
		return nil, fmt.Errorf("%s has no resolved_ids/unresolved_ids/empty_patch_ids/error_ids: not a SWE-bench harness report", path)
	}
	return &hr, nil
}

// ApplyHarnessReport sets each result's status from the harness verdicts:
// resolved -> pass, unresolved or empty patch -> fail. Results that already
// errored before evaluation keep StatusError. A harness error or an instance
// the harness did not evaluate becomes StatusError with an explanation, so
// nothing is counted as passed without a verdict. It re-tallies the report.
func ApplyHarnessReport(report *BenchReport, hr *HarnessReport, source string) {
	verdict := make(map[string]BenchStatus)
	for _, id := range hr.UnresolvedIDs {
		verdict[id] = StatusFail
	}
	for _, id := range hr.EmptyPatchIDs {
		verdict[id] = StatusFail
	}
	for _, id := range hr.ErrorIDs {
		verdict[id] = StatusError
	}
	for _, id := range hr.ResolvedIDs {
		verdict[id] = StatusPass
	}

	for i := range report.Results {
		r := &report.Results[i]
		v, ok := verdict[r.TaskID]
		switch {
		case !ok:
			if r.Status != StatusError {
				r.Status = StatusError
				r.Error = "not evaluated by the SWE-bench harness"
			}
		case v == StatusError:
			if r.Status != StatusError {
				r.Error = "SWE-bench harness error while evaluating the patch"
			}
			r.Status = StatusError
		case r.Status == StatusError && v == StatusFail:
			// Keep the run error: it explains why there was no patch.
		default:
			r.Status = v
		}
	}
	report.Evaluation = "swe-bench harness: " + source
	report.Tally()
}
