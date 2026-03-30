package continuous

import (
	"testing"
)

func TestParseSourceURI(t *testing.T) {
	tests := []struct {
		name    string
		uri     string
		want    *SourceConfig
		wantErr bool
	}{
		{
			name: "github with label",
			uri:  "github:label=bug",
			want: &SourceConfig{
				Provider: "github",
				Params:   map[string]string{"label": "bug"},
			},
		},
		{
			name: "github with multiple params",
			uri:  "github:label=bug,state=open,sort=created,direction=asc",
			want: &SourceConfig{
				Provider: "github",
				Params: map[string]string{
					"label":     "bug",
					"state":     "open",
					"sort":      "created",
					"direction": "asc",
				},
			},
		},
		{
			name: "file source",
			uri:  "file:queue.txt",
			want: &SourceConfig{
				Provider: "file",
				Params:   map[string]string{"path": "queue.txt"},
			},
		},
		{
			name: "file source with path",
			uri:  "file:/tmp/items.txt",
			want: &SourceConfig{
				Provider: "file",
				Params:   map[string]string{"path": "/tmp/items.txt"},
			},
		},
		{
			name:    "empty string",
			uri:     "",
			wantErr: true,
		},
		{
			name:    "missing colon",
			uri:     "github",
			wantErr: true,
		},
		{
			name:    "unknown provider",
			uri:     "gitlab:label=bug",
			wantErr: true,
		},
		{
			name:    "empty params",
			uri:     "github:",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseSourceURI(tt.uri)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("ParseSourceURI(%q) expected error, got nil", tt.uri)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseSourceURI(%q) unexpected error: %v", tt.uri, err)
			}
			if got.Provider != tt.want.Provider {
				t.Errorf("Provider = %q, want %q", got.Provider, tt.want.Provider)
			}
			if len(got.Params) != len(tt.want.Params) {
				t.Errorf("Params length = %d, want %d", len(got.Params), len(tt.want.Params))
			}
			for k, v := range tt.want.Params {
				if got.Params[k] != v {
					t.Errorf("Params[%q] = %q, want %q", k, got.Params[k], v)
				}
			}
		})
	}
}
