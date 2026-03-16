package continuous

import (
	"testing"
)

func TestNewGitHubSource(t *testing.T) {
	tests := []struct {
		name      string
		params    map[string]string
		wantLabel string
		wantState string
		wantLimit int
		wantErr   bool
	}{
		{
			name:      "defaults",
			params:    map[string]string{},
			wantState: "open",
			wantLimit: 100,
		},
		{
			name:      "with label",
			params:    map[string]string{"label": "bug"},
			wantLabel: "bug",
			wantState: "open",
			wantLimit: 100,
		},
		{
			name:      "custom limit",
			params:    map[string]string{"limit": "50"},
			wantState: "open",
			wantLimit: 50,
		},
		{
			name:    "invalid limit",
			params:  map[string]string{"limit": "abc"},
			wantErr: true,
		},
		{
			name:      "all params",
			params:    map[string]string{"label": "enhancement", "state": "closed", "sort": "updated", "direction": "desc", "limit": "25"},
			wantLabel: "enhancement",
			wantState: "closed",
			wantLimit: 25,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src, err := NewGitHubSource(tt.params)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if src.Label != tt.wantLabel {
				t.Errorf("Label = %q, want %q", src.Label, tt.wantLabel)
			}
			if src.State != tt.wantState {
				t.Errorf("State = %q, want %q", src.State, tt.wantState)
			}
			if src.Limit != tt.wantLimit {
				t.Errorf("Limit = %d, want %d", src.Limit, tt.wantLimit)
			}
		})
	}
}

func TestGitHubSourceName(t *testing.T) {
	src := &GitHubSource{Label: "bug", State: "open"}
	if got := src.Name(); got != "github(label=bug, state=open)" {
		t.Errorf("Name() = %q", got)
	}

	src2 := &GitHubSource{State: "closed"}
	if got := src2.Name(); got != "github(state=closed)" {
		t.Errorf("Name() = %q", got)
	}
}
