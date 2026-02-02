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
		if err := compiler.AddResource(schemaURL, strings.NewReader(cfg.Schema)); err != nil {
			return fmt.Errorf("failed to add schema resource: %w", err)
		}
	} else if cfg.SchemaPath != "" {
		data, err := os.ReadFile(cfg.SchemaPath)
		if err != nil {
			return fmt.Errorf("failed to read schema file: %w", err)
		}
		if err := compiler.AddResource(cfg.SchemaPath, strings.NewReader(string(data))); err != nil {
			return fmt.Errorf("failed to add schema resource: %w", err)
		}
		schemaURL = cfg.SchemaPath
	} else {
		return fmt.Errorf("no schema or schemaPath provided")
	}

	schema, err := compiler.Compile(schemaURL)
	if err != nil {
		return fmt.Errorf("failed to compile schema: %w", err)
	}

	artifactPath := filepath.Join(workspacePath, "artifact.json")
	data, err := os.ReadFile(artifactPath)
	if err != nil {
		return fmt.Errorf("failed to read artifact file: %w", err)
	}

	var artifact interface{}
	if err := json.Unmarshal(data, &artifact); err != nil {
		return fmt.Errorf("failed to parse artifact JSON: %w", err)
	}

	if err := schema.Validate(artifact); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	return nil
}
