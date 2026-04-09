package bench

import "testing"

func TestCompare(t *testing.T) {
	tests := []struct {
		name         string
		base         *BenchReport
		comp         *BenchReport
		wantImproved int
		wantRegress  int
		wantUnchange int
		wantOnlyBase int
		wantOnlyComp int
		wantDelta    float64
	}{
		{
			name: "identical runs",
			base: &BenchReport{
				Pipeline: "wave",
				PassRate: 0.5,
				Results: []BenchResult{
					{TaskID: "t1", Status: StatusPass},
					{TaskID: "t2", Status: StatusFail},
				},
			},
			comp: &BenchReport{
				Pipeline: "claude",
				PassRate: 0.5,
				Results: []BenchResult{
					{TaskID: "t1", Status: StatusPass},
					{TaskID: "t2", Status: StatusFail},
				},
			},
			wantUnchange: 2,
			wantDelta:    0.0,
		},
		{
			name: "compare improves on base",
			base: &BenchReport{
				Pipeline: "claude",
				PassRate: 0.0,
				Results: []BenchResult{
					{TaskID: "t1", Status: StatusFail},
					{TaskID: "t2", Status: StatusError},
				},
			},
			comp: &BenchReport{
				Pipeline: "wave",
				PassRate: 1.0,
				Results: []BenchResult{
					{TaskID: "t1", Status: StatusPass},
					{TaskID: "t2", Status: StatusPass},
				},
			},
			wantImproved: 2,
			wantDelta:    1.0,
		},
		{
			name: "compare regresses",
			base: &BenchReport{
				Pipeline: "wave",
				PassRate: 1.0,
				Results: []BenchResult{
					{TaskID: "t1", Status: StatusPass},
				},
			},
			comp: &BenchReport{
				Pipeline: "claude",
				PassRate: 0.0,
				Results: []BenchResult{
					{TaskID: "t1", Status: StatusFail},
				},
			},
			wantRegress: 1,
			wantDelta:   -1.0,
		},
		{
			name: "disjoint tasks",
			base: &BenchReport{
				Pipeline: "a",
				PassRate: 1.0,
				Results: []BenchResult{
					{TaskID: "t1", Status: StatusPass},
				},
			},
			comp: &BenchReport{
				Pipeline: "b",
				PassRate: 1.0,
				Results: []BenchResult{
					{TaskID: "t2", Status: StatusPass},
				},
			},
			wantOnlyBase: 1,
			wantOnlyComp: 1,
			wantDelta:    0.0,
		},
		{
			name: "empty reports",
			base: &BenchReport{Pipeline: "a"},
			comp: &BenchReport{Pipeline: "b"},
		},
		{
			name: "mixed changes",
			base: &BenchReport{
				Pipeline: "baseline",
				PassRate: 0.5,
				Results: []BenchResult{
					{TaskID: "t1", Status: StatusPass},
					{TaskID: "t2", Status: StatusFail},
					{TaskID: "t3", Status: StatusError},
					{TaskID: "t4", Status: StatusPass},
				},
			},
			comp: &BenchReport{
				Pipeline: "new",
				PassRate: 0.75,
				Results: []BenchResult{
					{TaskID: "t1", Status: StatusPass},
					{TaskID: "t2", Status: StatusPass},
					{TaskID: "t3", Status: StatusFail},
					{TaskID: "t4", Status: StatusFail},
				},
			},
			wantImproved: 1, // t2: fail→pass
			wantRegress:  1, // t4: pass→fail
			wantUnchange: 2, // t1: pass→pass, t3: error→fail (both non-pass)
			wantDelta:    0.25,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cr := Compare(tt.base, tt.comp)

			if cr.Summary.Improved != tt.wantImproved {
				t.Errorf("Improved = %d, want %d", cr.Summary.Improved, tt.wantImproved)
			}
			if cr.Summary.Regressed != tt.wantRegress {
				t.Errorf("Regressed = %d, want %d", cr.Summary.Regressed, tt.wantRegress)
			}
			if cr.Summary.Unchanged != tt.wantUnchange {
				t.Errorf("Unchanged = %d, want %d", cr.Summary.Unchanged, tt.wantUnchange)
			}
			if cr.Summary.OnlyInBase != tt.wantOnlyBase {
				t.Errorf("OnlyInBase = %d, want %d", cr.Summary.OnlyInBase, tt.wantOnlyBase)
			}
			if cr.Summary.OnlyInComp != tt.wantOnlyComp {
				t.Errorf("OnlyInComp = %d, want %d", cr.Summary.OnlyInComp, tt.wantOnlyComp)
			}

			const epsilon = 0.001
			if diff := cr.Summary.DeltaRate - tt.wantDelta; diff > epsilon || diff < -epsilon {
				t.Errorf("DeltaRate = %f, want %f", cr.Summary.DeltaRate, tt.wantDelta)
			}

			// Verify ReportRef metadata.
			if cr.Base.Pipeline != tt.base.Pipeline {
				t.Errorf("Base.Pipeline = %q, want %q", cr.Base.Pipeline, tt.base.Pipeline)
			}
			if cr.Compare.Pipeline != tt.comp.Pipeline {
				t.Errorf("Compare.Pipeline = %q, want %q", cr.Compare.Pipeline, tt.comp.Pipeline)
			}
		})
	}
}
