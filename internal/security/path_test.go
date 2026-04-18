package security

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newTestPathValidator(opts ...func(*SecurityConfig)) *PathValidator {
	config := DefaultSecurityConfig()
	for _, opt := range opts {
		opt(config)
	}
	logger := NewSecurityLogger(false)
	return NewPathValidator(*config, logger)
}

func TestValidatePath(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name    string
		path    string
		opts    []func(*SecurityConfig)
		wantErr bool
	}{
		{
			name:    "valid relative path within approved dir",
			path:    ".agents/contracts/schema.json",
			wantErr: false,
		},
		{
			name:    "valid relative path in schemas dir",
			path:    ".agents/schemas/output.json",
			wantErr: false,
		},
		{
			name:    "traversal with dot-dot-slash",
			path:    "../../etc/passwd",
			wantErr: true,
		},
		{
			name:    "traversal with backslash",
			path:    "..\\..\\etc\\passwd",
			wantErr: true,
		},
		{
			name:    "encoded traversal %2e%2e",
			path:    "%2e%2e/%2e%2e/etc/passwd",
			wantErr: true,
		},
		{
			name:    "path exceeds max length",
			path:    strings.Repeat("a", 256),
			wantErr: true,
		},
		{
			name:    "path at max length within approved dir",
			path:    ".agents/contracts/" + strings.Repeat("a", 230),
			wantErr: false,
		},
		{
			name:    "absolute path with no approved dirs rejects",
			path:    "/etc/passwd",
			opts:    []func(*SecurityConfig){func(c *SecurityConfig) { c.PathValidation.ApprovedDirectories = nil }},
			wantErr: true,
		},
		{
			name:    "relative path with no approved dirs accepts",
			path:    "simple.json",
			opts:    []func(*SecurityConfig){func(c *SecurityConfig) { c.PathValidation.ApprovedDirectories = nil }},
			wantErr: false,
		},
		{
			name:    "absolute path within approved dir",
			path:    filepath.Join(tmpDir, "file.json"),
			opts:    []func(*SecurityConfig){func(c *SecurityConfig) { c.PathValidation.ApprovedDirectories = []string{tmpDir} }},
			wantErr: false,
		},
		{
			name:    "absolute path outside approved dir",
			path:    "/tmp/other/file.json",
			opts:    []func(*SecurityConfig){func(c *SecurityConfig) { c.PathValidation.ApprovedDirectories = []string{tmpDir} }},
			wantErr: true,
		},
		{
			name:    "dot-slash traversal pattern",
			path:    "./../../secret",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pv := newTestPathValidator(tt.opts...)
			_, err := pv.ValidatePath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePath(%q) error = %v, wantErr %v", tt.path, err, tt.wantErr)
			}
		})
	}
}

func TestContainsTraversal(t *testing.T) {
	pv := newTestPathValidator()

	tests := []struct {
		name string
		path string
		want bool
	}{
		{"double dot in filename triggers detection", "foo/..bar", true},
		{"dot-dot standalone", "foo/../bar", true},
		{"dot-dot at start", "../bar", true},
		{"dot-slash", "./bar", true},
		{"dot-dot-slash", "foo/../bar", true},
		{"dot-dot-backslash", "foo/..\\bar", true},
		{"dot-backslash", "foo/.\\bar", true},
		{"double-dot-double-backslash", "foo/..\\\\bar", true},
		{"encoded %2e%2e", "foo/%2e%2e/bar", true},
		{"double encoded %252e%252e", "foo/%252e%252e/bar", true},
		{"encoded dot-dot-slash ..%2f", "foo/..%2f/bar", true},
		{"encoded dot-dot-backslash ..%5c", "foo/..%5c/bar", true},
		{"clean path", "foo/bar/baz", false},
		{"just a filename", "schema.json", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pv.containsTraversal(tt.path)
			if got != tt.want {
				t.Errorf("containsTraversal(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestValidateApprovedDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	approvedDir := filepath.Join(tmpDir, "approved")
	if err := os.MkdirAll(approvedDir, 0755); err != nil {
		t.Fatalf("Failed to create approved dir: %v", err)
	}

	tests := []struct {
		name         string
		path         string
		approvedDirs []string
		wantErr      bool
	}{
		{
			name:         "path within approved directory",
			path:         filepath.Join(approvedDir, "schema.json"),
			approvedDirs: []string{approvedDir},
			wantErr:      false,
		},
		{
			name:         "path outside approved directory",
			path:         filepath.Join(tmpDir, "outside", "file.json"),
			approvedDirs: []string{approvedDir},
			wantErr:      true,
		},
		{
			name:         "empty approved dirs allows relative paths",
			path:         "relative/file.json",
			approvedDirs: nil,
			wantErr:      false,
		},
		{
			name:         "empty approved dirs rejects absolute paths",
			path:         "/etc/passwd",
			approvedDirs: nil,
			wantErr:      true,
		},
		{
			name:         "multiple approved dirs - first matches",
			path:         filepath.Join(approvedDir, "a.json"),
			approvedDirs: []string{approvedDir, filepath.Join(tmpDir, "other")},
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultSecurityConfig()
			config.PathValidation.ApprovedDirectories = tt.approvedDirs
			logger := NewSecurityLogger(false)
			pv := NewPathValidator(*config, logger)

			_, err := pv.validateApprovedDirectory(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateApprovedDirectory(%q) error = %v, wantErr %v", tt.path, err, tt.wantErr)
			}
		})
	}
}

func TestContainsSymlinks(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a real directory and file
	realDir := filepath.Join(tmpDir, "real")
	if err := os.MkdirAll(realDir, 0755); err != nil {
		t.Fatalf("Failed to create real dir: %v", err)
	}
	realFile := filepath.Join(realDir, "file.txt")
	if err := os.WriteFile(realFile, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to create real file: %v", err)
	}

	// Create a symlink
	symlinkPath := filepath.Join(tmpDir, "link")
	if err := os.Symlink(realDir, symlinkPath); err != nil {
		t.Skip("Symlinks not supported on this platform")
	}

	pv := newTestPathValidator()

	// containsSymlinks splits on filepath.Separator and walks components,
	// which requires relative paths from CWD to work correctly with
	// os.Lstat. Test with CWD set to tmpDir and relative paths.
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get CWD: %v", err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(origDir) })

	tests := []struct {
		name string
		path string
		want bool
	}{
		{
			name: "path without symlinks",
			path: filepath.Join("real", "file.txt"),
			want: false,
		},
		{
			name: "path through symlink",
			path: filepath.Join("link", "file.txt"),
			want: true,
		},
		{
			name: "symlink itself",
			path: "link",
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pv.containsSymlinks(tt.path)
			if got != tt.want {
				t.Errorf("containsSymlinks(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestIsWithinDirectory(t *testing.T) {
	pv := newTestPathValidator()

	tests := []struct {
		name string
		path string
		dir  string
		want bool
	}{
		{
			name: "child within parent",
			path: "/home/user/project/file.txt",
			dir:  "/home/user/project",
			want: true,
		},
		{
			name: "deeply nested child",
			path: "/home/user/project/a/b/c/file.txt",
			dir:  "/home/user/project",
			want: true,
		},
		{
			name: "sibling directory",
			path: "/home/user/other/file.txt",
			dir:  "/home/user/project",
			want: false,
		},
		{
			name: "identical paths",
			path: "/home/user/project",
			dir:  "/home/user/project",
			want: true,
		},
		{
			name: "parent of directory",
			path: "/home/user",
			dir:  "/home/user/project",
			want: false,
		},
		{
			name: "path with dot-dot components",
			path: "/home/user/project/../other/file.txt",
			dir:  "/home/user/project",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pv.isWithinDirectory(tt.path, tt.dir)
			if got != tt.want {
				t.Errorf("isWithinDirectory(%q, %q) = %v, want %v", tt.path, tt.dir, got, tt.want)
			}
		})
	}
}

func TestSanitizePathForDisplay(t *testing.T) {
	pv := newTestPathValidator()

	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "short path unchanged",
			path: "simple/file.txt",
			want: "simple/file.txt",
		},
		{
			name: "short path with traversal replaced",
			path: "../secret/file.txt",
			want: "[..]/secret/file.txt",
		},
		{
			name: "multiple traversals replaced",
			path: "../../etc/passwd",
			want: "[..]/[..]/etc/passwd",
		},
		{
			name: "long path replaced with placeholder",
			path: strings.Repeat("a/", 30) + "file.txt",
			want: "<path:68 chars>",
		},
		{
			name: "exactly 50 chars not replaced",
			path: strings.Repeat("a", 50),
			want: strings.Repeat("a", 50),
		},
		{
			name: "51 chars replaced with placeholder",
			path: strings.Repeat("a", 51),
			want: "<path:51 chars>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pv.SanitizePathForDisplay(tt.path)
			if got != tt.want {
				t.Errorf("SanitizePathForDisplay(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}
