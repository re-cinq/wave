package contract

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/recinq/wave/internal/pathfmt"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

// preloadSharedDefs scans .wave/contracts/_defs/*.schema.json and registers
// each file as a compiler resource so that $ref across schema files resolves.
// schemaURI is the URI used to register the main schema (for computing the
// relative _defs URI prefix). fsSchemaDir is the filesystem directory containing
// the main schema (for reading _defs files from disk).
// If the _defs directory does not exist, this is a no-op (backwards compatible).
func preloadSharedDefs(compiler *jsonschema.Compiler, schemaURI string, fsSchemaDir string) error {
	uriDir := filepath.Dir(schemaURI)
	defsFSDir := filepath.Join(fsSchemaDir, "_defs")

	entries, err := os.ReadDir(defsFSDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // _defs/ doesn't exist — skip silently
		}
		return fmt.Errorf("reading _defs directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".schema.json") {
			continue
		}

		filePath := filepath.Join(defsFSDir, entry.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("reading shared def %s: %w", entry.Name(), err)
		}

		var doc interface{}
		if err := json.Unmarshal(data, &doc); err != nil {
			return fmt.Errorf("parsing shared def %s: %w", entry.Name(), err)
		}

		// URI matches what $ref resolves to from the parent schema's URI
		uri := filepath.Join(uriDir, "_defs", entry.Name())
		if err := compiler.AddResource(uri, doc); err != nil {
			return fmt.Errorf("registering shared def %s: %w", entry.Name(), err)
		}
	}

	return nil
}

type jsonSchemaValidator struct{}

func (v *jsonSchemaValidator) Validate(cfg ContractConfig, workspacePath string) error {
	compiler := jsonschema.NewCompiler()
	schemaURL := "schema.json"

	switch {
	case cfg.Schema != "":
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
	case cfg.SchemaPath != "":
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
	default:
		return &ValidationError{
			ContractType: "json_schema",
			Message:      "no schema or schemaPath provided",
			Details:      []string{"specify either 'schema' (inline JSON) or 'schemaPath' (file path)"},
			Retryable:    false,
		}
	}

	// Pre-load shared definition schemas from _defs/ so $ref across files resolves.
	if cfg.SchemaPath != "" {
		if err := preloadSharedDefs(compiler, cfg.SchemaPath, filepath.Dir(cfg.SchemaPath)); err != nil {
			return &ValidationError{
				ContractType: "json_schema",
				Message:      "failed to preload shared definitions",
				Details:      []string{err.Error()},
				Retryable:    false,
			}
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

	// Use source path if provided, otherwise default to artifact.json
	artifactFile := ".wave/artifact.json"
	if cfg.Source != "" {
		artifactFile = cfg.Source
	}
	artifactPath := filepath.Join(workspacePath, artifactFile)
	data, err := os.ReadFile(artifactPath)
	if err != nil {
		return &ValidationError{
			ContractType: "json_schema",
			Message:      fmt.Sprintf("failed to read artifact file: %s", pathfmt.FileURI(artifactPath)),
			Details:      []string{err.Error()},
			Retryable:    false,
		}
	}

	// NEW: Error wrapper detection and extraction
	// Detection is enabled by default, disabled only if explicitly configured
	if !cfg.DisableWrapperDetection {

		wrapperResult, wrapperErr := DetectErrorWrapper(data)
		switch {
		case wrapperErr != nil:
			// Wrapper detection failed, continue with original data
			if cfg.DebugMode {
				fmt.Printf("[DEBUG] Wrapper detection failed: %v\n", wrapperErr)
			}
		case wrapperResult.IsWrapper:
			// Extract raw content from wrapper for validation
			originalDataLength := len(data)
			data = wrapperResult.RawContent

			// Log wrapper extraction for debugging if needed
			// Note: In production this should use proper logging infrastructure
			if cfg.DebugMode {
				debug := wrapperResult.GetDebugInfo(originalDataLength)
				fmt.Printf("[DEBUG] Error wrapper detected and extracted: %+v\n", debug)
			}
		default:
			if cfg.DebugMode {
				debug := wrapperResult.GetDebugInfo(len(data))
				fmt.Printf("[DEBUG] No error wrapper detected: %+v\n", debug)
			}
		}
	}

	// Enhanced JSON recovery with progressive validation
	// Default to allowing recovery for better reliability
	allowRecovery := cfg.AllowRecovery
	if !cfg.AllowRecovery && cfg.RecoveryLevel == "" {
		// If recovery settings are not explicitly configured, enable progressive recovery by default
		allowRecovery = true
	}

	var artifact interface{}
	var recoveryResult *RecoveryResult

	if allowRecovery {
		recoveryLevel := determineRecoveryLevel(cfg)
		// Default to progressive recovery for better AI compatibility,
		// but keep conservative for must_pass contracts to avoid
		// aggressive inference corrupting data.
		if recoveryLevel == ConservativeRecovery && cfg.RecoveryLevel == "" && !cfg.MustPass {
			recoveryLevel = ProgressiveRecovery
		}

		recoveryParser := NewJSONRecoveryParser(recoveryLevel)

		var recoveryErr error
		recoveryResult, recoveryErr = recoveryParser.ParseWithRecovery(string(data))
		if recoveryErr != nil || !recoveryResult.IsValid {
			details := []string{fmt.Sprintf("file: %s", pathfmt.FileURI(artifactPath))}
			if recoveryErr != nil {
				details = append(details, recoveryErr.Error())
			}
			if recoveryResult != nil {
				if len(recoveryResult.AppliedFixes) > 0 {
					details = append(details, fmt.Sprintf("JSON Recovery Applied: %v", recoveryResult.AppliedFixes))
				}
				if len(recoveryResult.Warnings) > 0 {
					details = append(details, fmt.Sprintf("Warnings: %v", recoveryResult.Warnings))
				}
			}

			return &ValidationError{
				ContractType: "json_schema",
				Message:      "failed to parse artifact JSON after recovery attempts",
				Details:      details,
				Retryable:    true,
			}
		}

		// Verify recovered JSON is structurally complete before using it.
		// This catches cases where markdown extraction or truncation
		// produced parseable but incomplete JSON.
		if !isStructurallyComplete(recoveryResult.RecoveredJSON) {
			details := []string{
				fmt.Sprintf("file: %s", pathfmt.FileURI(artifactPath)),
				"recovered JSON has unbalanced braces/brackets (likely truncated)",
			}
			if len(recoveryResult.AppliedFixes) > 0 {
				details = append(details, fmt.Sprintf("JSON Recovery Applied: %v", recoveryResult.AppliedFixes))
			}
			return &ValidationError{
				ContractType: "json_schema",
				Message:      "recovered JSON is structurally incomplete",
				Details:      details,
				Retryable:    true,
			}
		}

		// Use the recovered and parsed data
		artifact = recoveryResult.ParsedData
	} else {
		// No recovery - try to parse as-is
		if err := json.Unmarshal(data, &artifact); err != nil {
			return &ValidationError{
				ContractType: "json_schema",
				Message:      "failed to parse artifact JSON",
				Details:      []string{fmt.Sprintf("file: %s", pathfmt.FileURI(artifactPath)), err.Error()},
				Retryable:    true,
			}
		}

		// Create a simple recovery result for consistent error formatting
		recoveryResult = &RecoveryResult{
			OriginalInput: string(data),
			RecoveredJSON: string(data),
			IsValid:       true,
			AppliedFixes:  []string{},
			Warnings:      []string{},
			RecoveryLevel: ConservativeRecovery,
			ParsedData:    artifact,
		}
	}

	if err := schema.Validate(artifact); err != nil {
		// Use enhanced error formatting for better user experience
		formatter := &ValidationErrorFormatter{}
		validationErr := formatter.FormatJSONSchemaError(err, recoveryResult, artifactPath)

		// Apply progressive validation logic
		mustPass := cfg.MustPass

		// Check if progressive validation is enabled
		if cfg.ProgressiveValidation && !mustPass {
			// In progressive mode, convert to warnings instead of blocking errors
			validationErr.Message += " (progressive validation: warning only)"
			validationErr.Retryable = false // Don't retry warnings

			// Progressive warnings are formatted but not yet routed to the audit system.
			_ = formatter.FormatProgressiveValidationWarning(err, recoveryResult)
		} else {
			// Normal validation mode - schema mismatches are always retryable
			// (the AI can be given repair guidance to fix the artifact)
			validationErr.Retryable = true
		}

		// Add context about validation mode
		if !mustPass && !cfg.ProgressiveValidation {
			validationErr.Message += " (must_pass: false)"
		}

		return validationErr
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

// determineRecoveryLevel determines the appropriate JSON recovery level based on contract configuration
func determineRecoveryLevel(cfg ContractConfig) RecoveryLevel {
	// If recovery is explicitly disabled, return conservative
	if !cfg.AllowRecovery {
		return ConservativeRecovery
	}

	// Check for explicit recovery level configuration
	switch cfg.RecoveryLevel {
	case "conservative":
		return ConservativeRecovery
	case "progressive":
		return ProgressiveRecovery
	case "aggressive":
		return AggressiveRecovery
	}

	// Progressive validation logic:
	// - If MustPass is true, use conservative recovery (maintain strict validation)
	// - If progressive validation is enabled, use progressive recovery
	// - Default to conservative recovery for safety

	if cfg.MustPass {
		return ConservativeRecovery
	}

	// If progressive validation is enabled, use progressive recovery
	if cfg.ProgressiveValidation {
		return ProgressiveRecovery
	}

	// Default to conservative recovery for safety
	return ConservativeRecovery
}
