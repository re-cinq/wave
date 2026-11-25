package security

import (
	"crypto/rand"
	"fmt"
	"time"
)

// Severity represents the severity level of a security violation
type Severity string

const (
	SeverityLow      Severity = "LOW"
	SeverityMedium   Severity = "MEDIUM"
	SeverityHigh     Severity = "HIGH"
	SeverityCritical Severity = "CRITICAL"
)

// ViolationType represents the type of security violation
type ViolationType string

const (
	ViolationPathTraversal   ViolationType = "path_traversal"
	ViolationPromptInjection ViolationType = "prompt_injection"
	ViolationInvalidPersona  ViolationType = "invalid_persona"
	ViolationMalformedJSON   ViolationType = "malformed_json"
	ViolationInputValidation ViolationType = "input_validation"
)

// ViolationSource represents where the violation originated
type ViolationSource string

const (
	SourceSchemaPath         ViolationSource = "schema_path"
	SourceUserInput          ViolationSource = "user_input"
	SourceMetaPipeline       ViolationSource = "meta_pipeline"
	SourceContractValidation ViolationSource = "contract_validation"
)

// SecurityViolationEvent represents a detected security attempt
type SecurityViolationEvent struct {
	ID               string    `json:"id"`
	Timestamp        time.Time `json:"timestamp"`
	Type             string    `json:"type"`
	Source           string    `json:"source"`
	SanitizedDetails string    `json:"sanitized_details"`
	Severity         Severity  `json:"severity"`
	Blocked          bool      `json:"blocked"`
	UserID           string    `json:"user_id,omitempty"`
}

// SchemaValidationResult contains outcome of schema validation
type SchemaValidationResult struct {
	SchemaPath          string   `json:"schema_path"`
	ValidatedPath       string   `json:"validated_path"`
	Content             string   `json:"content"`
	SecurityFlags       []string `json:"security_flags"`
	IsValid             bool     `json:"is_valid"`
	ErrorMessage        string   `json:"error_message,omitempty"`
	SanitizationActions []string `json:"sanitization_actions"`
}

// InputSanitizationRecord tracks sanitization actions
type InputSanitizationRecord struct {
	InputHash         string   `json:"input_hash"`
	InputType         string   `json:"input_type"`
	SanitizationRules []string `json:"sanitization_rules"`
	ChangesDetected   bool     `json:"changes_detected"`
	SanitizedLength   int      `json:"sanitized_length"`
	OriginalLength    int      `json:"original_length"`
	RiskScore         int      `json:"risk_score"`
}

// GenerateEventID creates a unique identifier for security events
func GenerateEventID() string {
	bytes := make([]byte, 16)
	_, _ = rand.Read(bytes)
	return fmt.Sprintf("sec-%x", bytes)
}

// NewSchemaValidationResult creates a validation result for schema processing
func NewSchemaValidationResult(schemaPath, validatedPath, content string, isValid bool) *SchemaValidationResult {
	return &SchemaValidationResult{
		SchemaPath:          schemaPath,
		ValidatedPath:       validatedPath,
		Content:             content,
		SecurityFlags:       []string{},
		IsValid:             isValid,
		SanitizationActions: []string{},
	}
}

// AddSecurityFlag adds a security concern to the validation result
func (svr *SchemaValidationResult) AddSecurityFlag(flag string) {
	svr.SecurityFlags = append(svr.SecurityFlags, flag)
}
