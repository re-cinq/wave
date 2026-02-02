package contract

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

type jsonSchemaValidator struct{}

func (v *jsonSchemaValidator) Validate(cfg ContractConfig, workspacePath string) error {
	compiler := jsonschema.NewCompiler()
	schemaURL := "schema.json"

	if cfg.Schema != "" {
		var schemaDoc interface{}
		if err := json.Unmarshal([]byte(cfg.Schema), &schemaDoc); err != nil {
			return &ValidationError{
				ContractType: "json_schema",
				Message:      "failed to parse inline schema",
				Details:      []string{err.Error()},
				Retryable:    false,
			}
		}
		if err := compiler.AddResource(schemaURL, schemaDoc); err != nil {
			return &ValidationError{
				ContractType: "json_schema",
				Message:      "failed to add schema resource",
				Details:      []string{err.Error()},
				Retryable:    false,
			}
		}
	} else if cfg.SchemaPath != "" {
		data, err := os.ReadFile(cfg.SchemaPath)
		if err != nil {
			return &ValidationError{
				ContractType: "json_schema",
				Message:      fmt.Sprintf("failed to read schema file: %s", cfg.SchemaPath),
				Details:      []string{err.Error()},
				Retryable:    false,
			}
		}
		var schemaDoc interface{}
		if err := json.Unmarshal(data, &schemaDoc); err != nil {
			return &ValidationError{
				ContractType: "json_schema",
				Message:      fmt.Sprintf("failed to parse schema file: %s", cfg.SchemaPath),
				Details:      []string{err.Error()},
				Retryable:    false,
			}
		}
		if err := compiler.AddResource(cfg.SchemaPath, schemaDoc); err != nil {
			return &ValidationError{
				ContractType: "json_schema",
				Message:      "failed to add schema resource",
				Details:      []string{err.Error()},
				Retryable:    false,
			}
		}
		schemaURL = cfg.SchemaPath
	} else {
		return &ValidationError{
			ContractType: "json_schema",
			Message:      "no schema or schemaPath provided",
			Details:      []string{"specify either 'schema' (inline JSON) or 'schemaPath' (file path)"},
			Retryable:    false,
		}
	}

	schema, err := compiler.Compile(schemaURL)
	if err != nil {
		return &ValidationError{
			ContractType: "json_schema",
			Message:      "failed to compile schema",
			Details:      []string{err.Error()},
			Retryable:    false,
		}
	}

	artifactPath := filepath.Join(workspacePath, "artifact.json")
	data, err := os.ReadFile(artifactPath)
	if err != nil {
		return &ValidationError{
			ContractType: "json_schema",
			Message:      fmt.Sprintf("failed to read artifact file: %s", artifactPath),
			Details:      []string{err.Error()},
			Retryable:    false,
		}
	}

	var artifact interface{}
	if err := json.Unmarshal(data, &artifact); err != nil {
		return &ValidationError{
			ContractType: "json_schema",
			Message:      "failed to parse artifact JSON",
			Details:      []string{err.Error(), fmt.Sprintf("file: %s", artifactPath)},
			Retryable:    true,
		}
	}

	if err := schema.Validate(artifact); err != nil {
		details := extractSchemaValidationDetails(err)
		return &ValidationError{
			ContractType: "json_schema",
			Message:      "artifact does not match schema",
			Details:      details,
			Retryable:    true,
		}
	}

	return nil
}

// extractSchemaValidationDetails extracts detailed validation errors from the schema validator.
func extractSchemaValidationDetails(err error) []string {
	errStr := err.Error()
	// Split multi-line error messages into separate details
	lines := strings.Split(errStr, "\n")
	details := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			details = append(details, line)
		}
	}
	if len(details) == 0 {
		details = append(details, errStr)
	}
	return details
}
