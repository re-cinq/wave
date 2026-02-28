package pathfmt

import "testing"

func TestFileURI(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "absolute path",
			path: "/home/user/file.json",
			want: "file:///home/user/file.json",
		},
		{
			name: "relative path unchanged",
			path: ".wave/workspaces/run-123/step/artifact.json",
			want: ".wave/workspaces/run-123/step/artifact.json",
		},
		{
			name: "already file:// prefixed",
			path: "file:///home/user/file.json",
			want: "file:///home/user/file.json",
		},
		{
			name: "https URL unchanged",
			path: "https://github.com/org/repo",
			want: "https://github.com/org/repo",
		},
		{
			name: "empty string",
			path: "",
			want: "",
		},
		{
			name: "path with spaces",
			path: "/path/with spaces/file.json",
			want: "file:///path/with spaces/file.json",
		},
		{
			name: "root path",
			path: "/",
			want: "file:///",
		},
		{
			name: "deeply nested absolute path",
			path: "/home/mwc/Coding/recinq/wave/.wave/workspaces/gh-implement-20260228-050356-9682/__wt_gh-implement-20260228-050356-9682/.wave/output/issue-assessment.json",
			want: "file:///home/mwc/Coding/recinq/wave/.wave/workspaces/gh-implement-20260228-050356-9682/__wt_gh-implement-20260228-050356-9682/.wave/output/issue-assessment.json",
		},
		{
			name: "path with special characters",
			path: "/tmp/file (1).json",
			want: "file:///tmp/file (1).json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FileURI(tt.path)
			if got != tt.want {
				t.Errorf("FileURI(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}
