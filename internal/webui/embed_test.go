package webui

import (
	"html/template"
	"strings"
	"testing"
	"time"

	"github.com/recinq/wave/internal/humanize"
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

// TestTemplateFormatDurationRendersEscaped verifies the formatDuration
// template helper produces the canonical humanize output and that html/template
// escapes the "<" in the "<1s" sub-second variant.
func TestTemplateFormatDurationRendersEscaped(t *testing.T) {
	tmpl, err := template.New("t").Funcs(template.FuncMap{
		"formatDuration": humanize.DurationSeconds,
	}).Parse(`{{ formatDuration .S }}`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	cases := []struct {
		name    string
		seconds float64
		want    string
	}{
		{"zero renders dash", 0, "-"},
		{"sub-second escapes lt", 0.5, "&lt;1s"},
		{"seconds", 5, "5s"},
		{"minutes", 90, "1m30s"},
		{"hours", 3661, "1h1m"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var sb strings.Builder
			if err := tmpl.Execute(&sb, struct{ S float64 }{tc.seconds}); err != nil {
				t.Fatalf("execute: %v", err)
			}
			if got := sb.String(); got != tc.want {
				t.Errorf("formatDuration(%v) = %q, want %q", tc.seconds, got, tc.want)
			}
		})
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

func TestStatusClass_HookAndSkippedStatuses(t *testing.T) {
	tests := []struct {
		status string
		want   string
	}{
		{"skipped", "status-skipped"},
		{"hook_started", "status-hook-started"},
		{"hook_passed", "status-hook-passed"},
		{"hook_failed", "status-hook-failed"},
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

func TestCheckClass(t *testing.T) {
	tests := []struct {
		name       string
		status     string
		conclusion string
		want       string
	}{
		// Non-completed statuses always return running
		{"in_progress", "in_progress", "", "status-running"},
		{"queued", "queued", "", "status-running"},
		{"pending", "pending", "", "status-running"},
		// Completed statuses depend on conclusion
		{"completed_success", "completed", "success", "status-completed"},
		{"completed_failure", "completed", "failure", "status-failed"},
		{"completed_timed_out", "completed", "timed_out", "status-failed"},
		{"completed_action_required", "completed", "action_required", "status-failed"},
		{"completed_cancelled", "completed", "cancelled", "status-cancelled"},
		{"completed_skipped", "completed", "skipped", "status-pending"},
		{"completed_neutral", "completed", "neutral", "status-pending"},
		{"completed_unknown_conclusion", "completed", "stale", "status-unknown"},
		{"completed_empty_conclusion", "completed", "", "status-unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := checkClass(tt.status, tt.conclusion)
			if got != tt.want {
				t.Errorf("checkClass(%q, %q) = %q, want %q", tt.status, tt.conclusion, got, tt.want)
			}
		})
	}
}

func TestCheckIcon(t *testing.T) {
	tests := []struct {
		name       string
		status     string
		conclusion string
		want       string
	}{
		// Non-completed statuses
		{"in_progress", "in_progress", "", "●"},
		{"queued", "queued", "", "●"},
		// Completed statuses
		{"completed_success", "completed", "success", "✓"},
		{"completed_failure", "completed", "failure", "✕"},
		{"completed_timed_out", "completed", "timed_out", "✕"},
		{"completed_action_required", "completed", "action_required", "✕"},
		{"completed_cancelled", "completed", "cancelled", "✕"},
		{"completed_skipped", "completed", "skipped", "—"},
		{"completed_neutral", "completed", "neutral", "—"},
		{"completed_unknown", "completed", "something_else", "?"},
		{"completed_empty", "completed", "", "?"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := checkIcon(tt.status, tt.conclusion)
			if got != tt.want {
				t.Errorf("checkIcon(%q, %q) = %q, want %q", tt.status, tt.conclusion, got, tt.want)
			}
		})
	}
}

func TestCheckLabel(t *testing.T) {
	tests := []struct {
		name       string
		status     string
		conclusion string
		want       string
	}{
		// Non-completed statuses
		{"in_progress", "in_progress", "", "In Progress"},
		{"queued", "queued", "", "Queued"},
		{"pending", "pending", "", "Queued"},
		{"empty_status", "", "", "Queued"},
		// Completed conclusions
		{"completed_success", "completed", "success", "Passed"},
		{"completed_failure", "completed", "failure", "Failed"},
		{"completed_cancelled", "completed", "cancelled", "Cancelled"},
		{"completed_skipped", "completed", "skipped", "Skipped"},
		{"completed_neutral", "completed", "neutral", "Neutral"},
		{"completed_timed_out", "completed", "timed_out", "Timed Out"},
		{"completed_action_required", "completed", "action_required", "Action Required"},
		// Default: returns the conclusion string itself
		{"completed_unknown", "completed", "stale", "stale"},
		{"completed_empty", "completed", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := checkLabel(tt.status, tt.conclusion)
			if got != tt.want {
				t.Errorf("checkLabel(%q, %q) = %q, want %q", tt.status, tt.conclusion, got, tt.want)
			}
		})
	}
}

func TestRichInputFunc(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		linkedURL string
		want      string
	}{
		// GitHub URLs
		{"github_pr", "https://github.com/owner/repo/pull/123", "", "PR #123"},
		{"github_issue", "https://github.com/owner/repo/issues/456", "", "Issue #456"},
		{"github_pr_via_linkedURL", "some input", "https://github.com/owner/repo/pull/99", "PR #99"},
		{"github_issue_via_linkedURL", "some input", "https://github.com/owner/repo/issues/42", "Issue #42"},
		// GitLab URLs
		{"gitlab_mr", "https://gitlab.com/owner/repo/-/merge_requests/77", "", "MR !77"},
		{"gitlab_issue", "https://gitlab.com/owner/repo/-/issues/88", "", "Issue #88"},
		// Non-URL inputs
		{"short_input", "fix the login bug", "", "fix the login bug"},
		{"empty_input", "", "", ""},
		// Long input truncation (>80 chars): first 77 chars + "..."
		{"long_input_truncated", "This is a very long input string that exceeds eighty characters and should be truncated by the function to seventy-seven plus ellipsis", "", "This is a very long input string that exceeds eighty characters and should be..."},
		// Exactly 80 chars should not be truncated
		{"exactly_80_chars", "12345678901234567890123456789012345678901234567890123456789012345678901234567890", "", "12345678901234567890123456789012345678901234567890123456789012345678901234567890"},
		// 81 chars should be truncated
		{"81_chars_truncated", "123456789012345678901234567890123456789012345678901234567890123456789012345678901", "", "12345678901234567890123456789012345678901234567890123456789012345678901234567..."},
		// linkedURL takes precedence over input for URL matching
		{"linkedURL_precedence", "not a url", "https://github.com/org/repo/pull/5", "PR #5"},
		// Non-matching URL falls through to input display
		{"non_matching_url", "just some text", "https://example.com/page", "just some text"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := richInputFunc(tt.input, tt.linkedURL)
			if got != tt.want {
				t.Errorf("richInputFunc(%q, %q) = %q, want %q", tt.input, tt.linkedURL, got, tt.want)
			}
		})
	}
}

func TestFriendlyModelFunc(t *testing.T) {
	tests := []struct {
		model string
		want  string
	}{
		// Opus variants
		{"claude-opus-4-20250514", "strongest"},
		{"opus", "strongest"},
		{"strongest", "strongest"},
		// Sonnet variants
		{"claude-sonnet-4-20250514", "balanced"},
		{"sonnet", "balanced"},
		{"balanced", "balanced"},
		// Haiku variants
		{"claude-haiku-4-5-20251001", "cheapest"},
		{"haiku", "cheapest"},
		{"cheapest", "cheapest"},
		{"fastest", "cheapest"},
		// Case insensitivity
		{"OPUS", "strongest"},
		{"Sonnet", "balanced"},
		{"HAIKU", "cheapest"},
		// Default branch: short model name returned as-is
		{"gpt-4o", "gpt-4o"},
		{"custom-model", "custom-model"},
		{"", ""},
		// Default branch: exactly 20 chars (boundary)
		{"12345678901234567890", "12345678901234567890"},
		// Default branch: >20 chars triggers truncation with ellipsis
		{"123456789012345678901", "12345678901234567890…"},
		{"this-is-a-very-long-model-name-that-exceeds-twenty", "this-is-a-very-long-…"},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			got := friendlyModelFunc(tt.model)
			if got != tt.want {
				t.Errorf("friendlyModelFunc(%q) = %q, want %q", tt.model, got, tt.want)
			}
		})
	}
}

func TestFormatBytesFunc(t *testing.T) {
	tests := []struct {
		name string
		arg  interface{}
		want string
	}{
		// int64 cases
		{"int64_zero", int64(0), "0B"},
		{"int64_1_byte", int64(1), "1B"},
		{"int64_512_bytes", int64(512), "512B"},
		{"int64_1023_bytes", int64(1023), "1023B"},
		{"int64_1024_bytes", int64(1024), "1.0KB"},
		{"int64_1536_bytes", int64(1536), "1.5KB"},
		{"int64_1MB", int64(1024 * 1024), "1.0MB"},
		{"int64_1_5MB", int64(1024 * 1024 * 3 / 2), "1.5MB"},
		{"int64_10MB", int64(10 * 1024 * 1024), "10.0MB"},
		// int cases
		{"int_zero", 0, "0B"},
		{"int_100", 100, "100B"},
		{"int_2048", 2048, "2.0KB"},
		{"int_1MB", 1024 * 1024, "1.0MB"},
		// Unsupported types
		{"float64", float64(1024), "0B"},
		{"string", "1024", "0B"},
		{"nil", nil, "0B"},
		{"bool", true, "0B"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatBytesFunc(tt.arg)
			if got != tt.want {
				t.Errorf("formatBytesFunc(%v) = %q, want %q", tt.arg, got, tt.want)
			}
		})
	}
}

func TestTitleCaseFunc(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"empty", "", ""},
		{"single_word", "hello", "Hello"},
		{"already_capitalized", "Hello", "Hello"},
		{"two_words", "hello world", "Hello World"},
		{"hyphenated", "foo-bar-baz", "Foo Bar Baz"},
		{"underscored", "foo_bar_baz", "Foo Bar Baz"},
		{"mixed_separators", "foo-bar_baz qux", "Foo Bar Baz Qux"},
		{"all_caps", "ALL CAPS", "ALL CAPS"},
		{"single_char_words", "a b c", "A B C"},
		{"extra_spaces", "  hello  world  ", "Hello World"},
		{"single_char", "x", "X"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := titleCaseFunc(tt.input)
			if got != tt.want {
				t.Errorf("titleCaseFunc(%q) = %q, want %q", tt.input, got, tt.want)
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
