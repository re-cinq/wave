package webui

import (
	"html/template"
	"testing"
	"time"
)

func TestStatusClass(t *testing.T) {
	tests := []struct {
		status string
		want   string
	}{
		{"completed", "status-completed"},
		{"running", "status-running"},
		{"failed", "status-failed"},
		{"cancelled", "status-cancelled"},
		{"pending", "status-pending"},
		{"unknown", "status-unknown"},
		{"", "status-unknown"},
		{"COMPLETED", "status-unknown"},
		{"Running", "status-unknown"},
		{"queued", "status-unknown"},
		{"in_progress", "status-unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			got := statusClass(tt.status)
			if got != tt.want {
				t.Errorf("statusClass(%q) = %q, want %q", tt.status, got, tt.want)
			}
		})
	}
}

func TestFormatDurationShort(t *testing.T) {
	tests := []struct {
		name    string
		seconds float64
		want    string
	}{
		{"zero", 0, "<1s"},
		{"negative", -1, "<1s"},
		{"fraction below one", 0.5, "<1s"},
		{"just below one", 0.999, "<1s"},
		{"exactly one", 1, "1s"},
		{"just above one", 1.1, "1s"},
		{"five seconds", 5, "5s"},
		{"ten seconds", 10, "10s"},
		{"fifty nine seconds", 59, "59s"},
		{"rounds down at 1.9", 1.9, "2s"},
		{"rounds at 1.5", 1.5, "2s"},
		{"30.4 rounds to 30", 30.4, "30s"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatDurationShort(tt.seconds)
			if got != tt.want {
				t.Errorf("formatDurationShort(%v) = %q, want %q", tt.seconds, got, tt.want)
			}
		})
	}
}

func TestFormatMinSec(t *testing.T) {
	tests := []struct {
		name string
		m    int
		s    int
		want string
	}{
		{"1 minute 0 seconds", 1, 0, "1m 0s"},
		{"1 minute 30 seconds", 1, 30, "1m 30s"},
		{"2 minutes 5 seconds", 2, 5, "2m 5s"},
		{"0 minutes 45 seconds", 0, 45, "0m 45s"},
		{"60 minutes 0 seconds", 60, 0, "60m 0s"},
		{"10 minutes 59 seconds", 10, 59, "10m 59s"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatMinSec(tt.m, tt.s)
			if got != tt.want {
				t.Errorf("formatMinSec(%d, %d) = %q, want %q", tt.m, tt.s, got, tt.want)
			}
		})
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name    string
		seconds float64
		want    string
	}{
		// Below 60: delegates to formatDurationShort (HTML-escaped)
		{"zero", 0, "&lt;1s"},
		{"negative", -5, "&lt;1s"},
		{"half second", 0.5, "&lt;1s"},
		{"one second", 1, "1s"},
		{"thirty seconds", 30, "30s"},
		{"fifty nine seconds", 59, "59s"},
		{"just below sixty", 59.9, "60s"},
		// At and above 60: delegates to formatMinSec
		{"exactly sixty", 60, "1m 0s"},
		{"ninety seconds", 90, "1m 30s"},
		{"two minutes", 120, "2m 0s"},
		{"two minutes five seconds", 125, "2m 5s"},
		{"one hour", 3600, "60m 0s"},
		{"one hour thirty minutes", 5400, "90m 0s"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatDuration(tt.seconds)
			if got != tt.want {
				t.Errorf("formatDuration(%v) = %q, want %q", tt.seconds, got, tt.want)
			}
		})
	}
}

func TestFormatDuration_HTMLEscaping(t *testing.T) {
	// The "<1s" output from formatDurationShort must be HTML-escaped in formatDuration.
	got := formatDuration(0)
	if got != "&lt;1s" {
		t.Errorf("formatDuration(0) = %q, want %q (HTML-escaped)", got, "&lt;1s")
	}

	// Values >= 60 produce no HTML special chars, so escaping is a no-op.
	got = formatDuration(65)
	if got != "1m 5s" {
		t.Errorf("formatDuration(65) = %q, want %q", got, "1m 5s")
	}
}

func TestFormatTime(t *testing.T) {
	fixedTime := time.Date(2024, 6, 15, 10, 30, 45, 0, time.UTC)
	wantFormatted := "2024-06-15 10:30:45"

	tests := []struct {
		name string
		arg  interface{}
		want string
	}{
		// time.Time cases
		{
			name: "valid time.Time",
			arg:  fixedTime,
			want: wantFormatted,
		},
		{
			name: "zero time.Time",
			arg:  time.Time{},
			want: "-",
		},
		// *time.Time cases
		{
			name: "valid *time.Time",
			arg:  &fixedTime,
			want: wantFormatted,
		},
		{
			name: "nil *time.Time",
			arg:  (*time.Time)(nil),
			want: "-",
		},
		{
			name: "zero *time.Time",
			arg: func() *time.Time {
				z := time.Time{}
				return &z
			}(),
			want: "-",
		},
		// Unknown types
		{
			name: "string type",
			arg:  "2024-01-01",
			want: "-",
		},
		{
			name: "int type",
			arg:  12345,
			want: "-",
		},
		{
			name: "nil interface",
			arg:  nil,
			want: "-",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatTime(tt.arg)
			if got != tt.want {
				t.Errorf("formatTime(%v) = %q, want %q", tt.arg, got, tt.want)
			}
		})
	}
}

func TestFormatTime_Format(t *testing.T) {
	// Verify the exact layout "2006-01-02 15:04:05" is used.
	ts := time.Date(2000, 1, 2, 3, 4, 5, 0, time.UTC)
	got := formatTime(ts)
	want := "2000-01-02 03:04:05"
	if got != want {
		t.Errorf("formatTime layout: got %q, want %q", got, want)
	}
}

func TestStatusIcon(t *testing.T) {
	tests := []struct {
		status string
		want   string
	}{
		{"completed", "✓"},
		{"running", "●"},
		{"failed", "✕"},
		{"cancelled", "○"},
		{"pending", "◌"},
		{"unknown", "·"},
		{"", "·"},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			got := statusIcon(tt.status)
			if got != tt.want {
				t.Errorf("statusIcon(%q) = %q, want %q", tt.status, got, tt.want)
			}
		})
	}
}

func TestFormatTimeISO(t *testing.T) {
	fixedTime := time.Date(2024, 6, 15, 10, 30, 45, 0, time.UTC)

	tests := []struct {
		name string
		arg  interface{}
		want string
	}{
		{"valid time.Time", fixedTime, "2024-06-15T10:30:45Z"},
		{"zero time.Time", time.Time{}, ""},
		{"valid *time.Time", &fixedTime, "2024-06-15T10:30:45Z"},
		{"nil *time.Time", (*time.Time)(nil), ""},
		{"zero *time.Time", func() *time.Time { z := time.Time{}; return &z }(), ""},
		{"string type", "2024-01-01", ""},
		{"nil", nil, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatTimeISO(tt.arg)
			if got != tt.want {
				t.Errorf("formatTimeISO(%v) = %q, want %q", tt.arg, got, tt.want)
			}
		})
	}
}

func TestParseTemplates(t *testing.T) {
	tmpl, err := parseTemplates(template.FuncMap{
		"csrfToken":      func() string { return "test-token" },
		"featureEnabled": func(name string) bool { return true },
	})
	if err != nil {
		t.Fatalf("parseTemplates() error: %v", err)
	}
	if tmpl == nil {
		t.Fatal("parseTemplates() returned nil")
	}
	// Verify all expected page templates are present
	expected := []string{
		"templates/runs.html",
		"templates/run_detail.html",
		"templates/personas.html",
		"templates/persona_detail.html",
		"templates/pipelines.html",
		"templates/contracts.html",
		"templates/contract_detail.html",
		"templates/skills.html",
		"templates/compose.html",
		"templates/issues.html",
		"templates/prs.html",
		"templates/health.html",
		"templates/ontology.html",
		"templates/notfound.html",
		"templates/webhook_detail.html",
	}
	for _, page := range expected {
		if tmpl[page] == nil {
			t.Errorf("missing template for %q", page)
		}
	}
}

func TestFormatTokensFunc(t *testing.T) {
	tests := []struct {
		name string
		arg  interface{}
		want string
	}{
		// int cases
		{"int zero", 0, "0"},
		{"int small", 500, "500"},
		{"int exactly 1000", 1000, "1.0k"},
		{"int 1500", 1500, "1.5k"},
		{"int 999", 999, "999"},
		{"int 1_000_000", 1_000_000, "1.0M"},
		{"int 2_500_000", 2_500_000, "2.5M"},
		{"int 1_000_000_000", 1_000_000_000, "1.0B"},
		// int64 cases
		{"int64 zero", int64(0), "0"},
		{"int64 small", int64(42), "42"},
		{"int64 1000", int64(1000), "1.0k"},
		{"int64 large", int64(3_000_000), "3.0M"},
		// unknown types fall back to "0"
		{"float64 type", float64(1000), "0"},
		{"string type", "1000", "0"},
		{"nil", nil, "0"},
		{"bool type", true, "0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatTokensFunc(tt.arg)
			if got != tt.want {
				t.Errorf("formatTokensFunc(%v) = %q, want %q", tt.arg, got, tt.want)
			}
		})
	}
}

func TestModelTierClass(t *testing.T) {
	tests := []struct {
		model string
		want  string
	}{
		{"claude-opus-4-20250514", "tier-strongest"},
		{"opus", "tier-strongest"},
		{"strongest", "tier-strongest"},
		{"claude-sonnet-4-20250514", "tier-balanced"},
		{"sonnet", "tier-balanced"},
		{"balanced", "tier-balanced"},
		{"claude-haiku-4-5-20251001", "tier-cheapest"},
		{"haiku", "tier-cheapest"},
		{"cheapest", "tier-cheapest"},
		{"fastest", "tier-cheapest"},
		{"unknown-model", ""},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			got := modelTierClass(tt.model)
			if got != tt.want {
				t.Errorf("modelTierClass(%q) = %q, want %q", tt.model, got, tt.want)
			}
		})
	}
}
