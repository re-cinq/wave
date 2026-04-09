package contract

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNonEmptyFileValidator_Validate(t *testing.T) {
	tempDir := t.TempDir()

	// Create a non-empty file
	nonEmptyPath := filepath.Join(tempDir, "output.json")
	if err := os.WriteFile(nonEmptyPath, []byte(`{"result": "ok"}`), 0644); err != nil {
		t.Fatal(err)
	}

	// Create an empty file
	emptyPath := filepath.Join(tempDir, "empty.txt")
	if err := os.WriteFile(emptyPath, []byte{}, 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name        string
		config      ContractConfig
		workspace   string
		expectError bool
		errContains string
	}{
		{
			name: "non_empty_file_passes",
			config: ContractConfig{
				Type:   "non_empty_file",
				Source: "output.json",
			},
			workspace:   tempDir,
			expectError: false,
		},
		{
			name: "empty_file_fails",
			config: ContractConfig{
				Type:   "non_empty_file",
				Source: "empty.txt",
			},
			workspace:   tempDir,
			expectError: true,
			errContains: "file is empty",
		},
		{
			name: "missing_file_fails",
			config: ContractConfig{
				Type:   "non_empty_file",
				Source: "does-not-exist.txt",
			},
			workspace:   tempDir,
			expectError: true,
			errContains: "file not found",
		},
		{
			name: "no_source_fails",
			config: ContractConfig{
				Type: "non_empty_file",
			},
			workspace:   tempDir,
			expectError: true,
			errContains: "no source file specified",
		},
		{
			name: "absolute_path_resolves",
			config: ContractConfig{
				Type:   "non_empty_file",
				Source: nonEmptyPath,
			},
			workspace:   "/some/other/workspace",
			expectError: false,
		},
		{
			name: "relative_path_resolves",
			config: ContractConfig{
				Type:   "non_empty_file",
				Source: "output.json",
			},
			workspace:   tempDir,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := &nonEmptyFileValidator{}
			err := validator.Validate(tt.config, tt.workspace)

			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("expected no error but got: %v", err)
			}
			if tt.expectError && err != nil && tt.errContains != "" {
				if ve, ok := err.(*ValidationError); ok {
					if !strings.Contains(ve.Message, tt.errContains) {
						t.Errorf("expected error containing %q but got: %s", tt.errContains, ve.Message)
					}
				}
			}
		})
	}
}

func TestNonEmptyFileValidator_MustPass(t *testing.T) {
	tempDir := t.TempDir()

	// Create an empty file
	emptyPath := filepath.Join(tempDir, "empty.txt")
	if err := os.WriteFile(emptyPath, []byte{}, 0644); err != nil {
		t.Fatal(err)
	}

	// Validator itself always returns error for empty files;
	// must_pass is handled by the pipeline executor, not the validator
	validator := &nonEmptyFileValidator{}
	err := validator.Validate(ContractConfig{
		Type:     "non_empty_file",
		Source:   "empty.txt",
		MustPass: true,
	}, tempDir)

	if err == nil {
		t.Error("expected error for empty file")
	}

	ve, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("expected *ValidationError, got %T", err)
	}
	if !ve.Retryable {
		t.Error("empty file error should be retryable")
	}
}

func TestNonEmptyFileValidator_ViaNewValidator(t *testing.T) {
	cfg := ContractConfig{Type: "non_empty_file"}
	validator := NewValidator(cfg)
	if validator == nil {
		t.Fatal("NewValidator should return a non-nil validator for non_empty_file")
	}
}
