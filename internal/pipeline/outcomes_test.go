package pipeline

import (
	"errors"
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
		{
			name: "array index first element",
			data: `{"enhanced_issues": [{"url": "https://github.com/re-cinq/wave/issues/42"}, {"url": "https://github.com/re-cinq/wave/issues/43"}]}`,
			path: ".enhanced_issues[0].url",
			want: "https://github.com/re-cinq/wave/issues/42",
		},
		{
			name: "array index second element",
			data: `{"items": [{"name": "a"}, {"name": "b"}]}`,
			path: ".items[1].name",
			want: "b",
		},
		{
			name:    "array index out of bounds",
			data:    `{"items": [{"name": "a"}]}`,
			path:    ".items[5].name",
			wantErr: true,
		},
		{
			name:    "array index on non-array",
			data:    `{"items": "not an array"}`,
			path:    ".items[0]",
			wantErr: true,
		},
		{
			name:    "invalid array index",
			data:    `{"items": [1, 2]}`,
			path:    ".items[abc]",
			wantErr: true,
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

func TestExtractJSONPath_EmptyArrayReturnsEmptyArrayError(t *testing.T) {
	data := []byte(`{"enhanced_issues": []}`)
	_, err := ExtractJSONPath(data, ".enhanced_issues[0].url")
	if err == nil {
		t.Fatal("expected error for index 0 on empty array")
	}

	var emptyErr *EmptyArrayError
	if !errors.As(err, &emptyErr) {
		t.Fatalf("expected EmptyArrayError, got %T: %v", err, err)
	}
	if emptyErr.Field != "enhanced_issues" {
		t.Errorf("expected field %q, got %q", "enhanced_issues", emptyErr.Field)
	}
}

func TestExtractJSONPath_NonEmptyArrayOOBIsNotEmptyArrayError(t *testing.T) {
	data := []byte(`{"items": [{"name": "a"}, {"name": "b"}]}`)
	_, err := ExtractJSONPath(data, ".items[5].name")
	if err == nil {
		t.Fatal("expected error for index 5 on length-2 array")
	}

	var emptyErr *EmptyArrayError
	if errors.As(err, &emptyErr) {
		t.Fatalf("expected non-EmptyArrayError for OOB on non-empty array, got EmptyArrayError{Field: %q}", emptyErr.Field)
	}
}
