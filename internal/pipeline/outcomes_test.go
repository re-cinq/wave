package pipeline

import (
	"testing"
)

func TestExtractJSONPath(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		path    string
		want    string
		wantErr bool
	}{
		{
			name: "simple string field",
			data: `{"comment_url": "https://github.com/re-cinq/wave/pull/42#issuecomment-123"}`,
			path: ".comment_url",
			want: "https://github.com/re-cinq/wave/pull/42#issuecomment-123",
		},
		{
			name: "nested field",
			data: `{"result": {"pr_url": "https://github.com/re-cinq/wave/pull/99"}}`,
			path: ".result.pr_url",
			want: "https://github.com/re-cinq/wave/pull/99",
		},
		{
			name: "deeply nested field",
			data: `{"a": {"b": {"c": "deep"}}}`,
			path: ".a.b.c",
			want: "deep",
		},
		{
			name: "integer value",
			data: `{"pr_number": 42}`,
			path: ".pr_number",
			want: "42",
		},
		{
			name: "float value",
			data: `{"score": 3.14}`,
			path: ".score",
			want: "3.14",
		},
		{
			name: "boolean value",
			data: `{"success": true}`,
			path: ".success",
			want: "true",
		},
		{
			name: "path without leading dot",
			data: `{"url": "https://example.com"}`,
			path: "url",
			want: "https://example.com",
		},
		{
			name:    "missing key",
			data:    `{"other": "value"}`,
			path:    ".comment_url",
			wantErr: true,
		},
		{
			name:    "navigate into non-object",
			data:    `{"name": "test"}`,
			path:    ".name.sub",
			wantErr: true,
		},
		{
			name:    "null value",
			data:    `{"url": null}`,
			path:    ".url",
			wantErr: true,
		},
		{
			name:    "empty path",
			data:    `{"url": "test"}`,
			path:    "",
			wantErr: true,
		},
		{
			name:    "dot-only path",
			data:    `{"url": "test"}`,
			path:    ".",
			wantErr: true,
		},
		{
			name:    "invalid JSON",
			data:    `{not json`,
			path:    ".url",
			wantErr: true,
		},
		{
			name: "object value returns JSON",
			data: `{"meta": {"key": "val"}}`,
			path: ".meta",
			want: `{"key":"val"}`,
		},
		{
			name: "array value returns JSON",
			data: `{"items": [1, 2, 3]}`,
			path: ".items",
			want: "[1,2,3]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExtractJSONPath([]byte(tt.data), tt.path)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got value %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
