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

func TestExtractJSONPath_EmptyArrayReturnsemptyArrayError(t *testing.T) {
	data := []byte(`{"enhanced_issues": []}`)
	_, err := ExtractJSONPath(data, ".enhanced_issues[0].url")
	if err == nil {
		t.Fatal("expected error for index 0 on empty array")
	}

	var emptyErr *emptyArrayError
	if !errors.As(err, &emptyErr) {
		t.Fatalf("expected emptyArrayError, got %T: %v", err, err)
	}
	if emptyErr.Field != "enhanced_issues" {
		t.Errorf("expected field %q, got %q", "enhanced_issues", emptyErr.Field)
	}
}

func TestExtractJSONPath_NonEmptyArrayOOBIsNotemptyArrayError(t *testing.T) {
	data := []byte(`{"items": [{"name": "a"}, {"name": "b"}]}`)
	_, err := ExtractJSONPath(data, ".items[5].name")
	if err == nil {
		t.Fatal("expected error for index 5 on length-2 array")
	}

	var emptyErr *emptyArrayError
	if errors.As(err, &emptyErr) {
		t.Fatalf("expected non-emptyArrayError for OOB on non-empty array, got emptyArrayError{Field: %q}", emptyErr.Field)
	}
}
func TestContainsWildcard(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{".items[*].url", true},
		{".items[0].url", false},
		{".simple_field", false},
		{"[*]", true},
		{".result.items[*].name", true},
		{"", false},
		{".items[*]", true},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := ContainsWildcard(tt.path)
			if got != tt.want {
				t.Errorf("ContainsWildcard(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestExtractJSONPathAll(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		path    string
		want    []string
		wantErr bool
	}{
		{
			name: "array of strings",
			data: `{"urls": ["https://a.com", "https://b.com", "https://c.com"]}`,
			path: ".urls[*]",
			want: []string{"https://a.com", "https://b.com", "https://c.com"},
		},
		{
			name: "array of objects with sub-path",
			data: `{"enhanced_issues": [{"url": "https://github.com/issues/1"}, {"url": "https://github.com/issues/2"}]}`,
			path: ".enhanced_issues[*].url",
			want: []string{"https://github.com/issues/1", "https://github.com/issues/2"},
		},
		{
			name: "nested prefix path",
			data: `{"result": {"items": [{"name": "alpha"}, {"name": "beta"}]}}`,
			path: ".result.items[*].name",
			want: []string{"alpha", "beta"},
		},
		{
			name: "empty array returns empty slice",
			data: `{"items": []}`,
			path: ".items[*]",
			want: []string{},
		},
		{
			name:    "non-array at wildcard position",
			data:    `{"name": "not-an-array"}`,
			path:    ".name[*]",
			wantErr: true,
		},
		{
			name:    "missing field",
			data:    `{"other": "value"}`,
			path:    ".missing[*]",
			wantErr: true,
		},
		{
			name:    "invalid JSON",
			data:    `{not json`,
			path:    ".items[*]",
			wantErr: true,
		},
		{
			name: "single element array",
			data: `{"items": ["only"]}`,
			path: ".items[*]",
			want: []string{"only"},
		},
		{
			name:    "no wildcard in path returns error",
			data:    `{"items": ["a"]}`,
			path:    ".items",
			wantErr: true,
		},
		{
			name:    "empty path",
			data:    `{"items": ["a"]}`,
			path:    "",
			wantErr: true,
		},
		{
			name: "deeply nested with sub-path",
			data: `{"data": {"results": [{"issue_number": 42, "url": "https://gh.com/42"}, {"issue_number": 99, "url": "https://gh.com/99"}]}}`,
			path: ".data.results[*].url",
			want: []string{"https://gh.com/42", "https://gh.com/99"},
		},
		{
			name: "extract integer values as strings",
			data: `{"issues": [{"number": 1}, {"number": 2}]}`,
			path: ".issues[*].number",
			want: []string{"1", "2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExtractJSONPathAll([]byte(tt.data), tt.path)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got %v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("got %d results, want %d: %v", len(got), len(tt.want), got)
			}
			for i, v := range got {
				if v != tt.want[i] {
					t.Errorf("result[%d] = %q, want %q", i, v, tt.want[i])
				}
			}
		})
	}
}
