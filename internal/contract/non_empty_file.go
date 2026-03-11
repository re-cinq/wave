package contract

import (
	"fmt"
	"os"
	"path/filepath"
)

type nonEmptyFileValidator struct{}

func (v *nonEmptyFileValidator) Validate(cfg ContractConfig, workspacePath string) error {
	sourceFile := cfg.Source
	if sourceFile == "" {
		return &ValidationError{
			ContractType: "non_empty_file",
			Message:      "no source file specified",
			Details:      []string{"non_empty_file requires a source file path"},
			Retryable:    false,
		}
	}

	// Resolve relative paths against workspace
	sourcePath := sourceFile
	if !filepath.IsAbs(sourceFile) {
		sourcePath = filepath.Join(workspacePath, sourceFile)
	}

	// Check if file exists
	info, err := os.Stat(sourcePath)
	if err != nil {
		if os.IsNotExist(err) {
			return &ValidationError{
				ContractType: "non_empty_file",
				Message:      fmt.Sprintf("file not found: %s", sourcePath),
				Details:      []string{err.Error()},
				Retryable:    true,
			}
		}
		return &ValidationError{
			ContractType: "non_empty_file",
			Message:      fmt.Sprintf("cannot access file: %s", sourcePath),
			Details:      []string{err.Error()},
			Retryable:    false,
		}
	}

	// Check file is not empty
	if info.Size() == 0 {
		return &ValidationError{
			ContractType: "non_empty_file",
			Message:      fmt.Sprintf("file is empty: %s", sourcePath),
			Details:      []string{"non_empty_file requires the source file to have content"},
			Retryable:    true,
		}
	}

	return nil
}
