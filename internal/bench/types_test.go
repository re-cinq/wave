package bench

import "testing"

func TestBenchReport_Tally(t *testing.T) {
	tests := []struct {
		name     string
		results  []BenchResult
		wantPass int
		wantFail int
		wantErr  int
		wantRate float64
	}{
		{
			name:     "empty report",
			results:  nil,
			wantPass: 0,
			wantFail: 0,
			wantErr:  0,
			wantRate: 0,
		},
		{
			name: "all pass",
			results: []BenchResult{
				{TaskID: "a", Status: StatusPass},
				{TaskID: "b", Status: StatusPass},
			},
			wantPass: 2,
			wantFail: 0,
			wantErr:  0,
			wantRate: 1.0,
		},
		{
			name: "mixed results",
			results: []BenchResult{
				{TaskID: "a", Status: StatusPass},
				{TaskID: "b", Status: StatusFail},
				{TaskID: "c", Status: StatusError},
				{TaskID: "d", Status: StatusPass},
			},
			wantPass: 2,
			wantFail: 1,
			wantErr:  1,
			wantRate: 0.5,
		},
		{
			name: "all fail",
			results: []BenchResult{
				{TaskID: "a", Status: StatusFail},
				{TaskID: "b", Status: StatusFail},
			},
			wantPass: 0,
			wantFail: 2,
			wantErr:  0,
			wantRate: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &BenchReport{Results: tt.results}
			r.Tally()

			if r.Passed != tt.wantPass {
				t.Errorf("Passed = %d, want %d", r.Passed, tt.wantPass)
			}
			if r.Failed != tt.wantFail {
				t.Errorf("Failed = %d, want %d", r.Failed, tt.wantFail)
			}
			if r.Errors != tt.wantErr {
				t.Errorf("Errors = %d, want %d", r.Errors, tt.wantErr)
			}
			if r.Total != len(tt.results) {
				t.Errorf("Total = %d, want %d", r.Total, len(tt.results))
			}
			if r.PassRate != tt.wantRate {
				t.Errorf("PassRate = %f, want %f", r.PassRate, tt.wantRate)
			}
		})
	}
}

func TestBenchStatus_Values(t *testing.T) {
	if StatusPass != "pass" {
		t.Errorf("StatusPass = %q, want %q", StatusPass, "pass")
	}
	if StatusFail != "fail" {
		t.Errorf("StatusFail = %q, want %q", StatusFail, "fail")
	}
	if StatusError != "error" {
		t.Errorf("StatusError = %q, want %q", StatusError, "error")
	}
}

func TestBenchReport_TallyPatchedIsNotPass(t *testing.T) {
	r := &BenchReport{Results: []BenchResult{
		{TaskID: "a", Status: StatusPatched},
		{TaskID: "b", Status: StatusPass},
	}}
	r.Tally()
	if r.Patched != 1 || r.Passed != 1 || r.PassRate != 0.5 {
		t.Errorf("Patched=%d Passed=%d PassRate=%f, want 1, 1, 0.5", r.Patched, r.Passed, r.PassRate)
	}
}
