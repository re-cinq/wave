package continuous

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewSourceFromConfig(t *testing.T) {
	// Create a temp file for file source test
	dir := t.TempDir()
	path := filepath.Join(dir, "queue.txt")
	if err := os.WriteFile(path, []byte("item1\nitem2\n"), 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name     string
		cfg      *SourceConfig
		wantName string
		wantErr  bool
	}{
		{
			name: "github source",
			cfg: &SourceConfig{
				Provider: "github",
				Params:   map[string]string{"label": "bug"},
			},
			wantName: "github(label=bug, state=open)",
		},
		{
			name: "file source",
			cfg: &SourceConfig{
				Provider: "file",
				Params:   map[string]string{"path": path},
			},
			wantName: "file(" + path + ")",
		},
		{
			name: "unknown provider",
			cfg: &SourceConfig{
				Provider: "gitlab",
				Params:   map[string]string{},
			},
			wantErr: true,
		},
		{
			name: "file source missing path",
			cfg: &SourceConfig{
				Provider: "file",
				Params:   map[string]string{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src, err := NewSourceFromConfig(tt.cfg)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got := src.Name(); got != tt.wantName {
				t.Errorf("Name() = %q, want %q", got, tt.wantName)
			}
		})
	}
}
