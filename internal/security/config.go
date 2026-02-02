package security

import "time"

// PathValidationConfig configures path traversal prevention
type PathValidationConfig struct {
	// ApprovedDirectories lists allowlisted base directories for schema files
	ApprovedDirectories []string `yaml:"approved_directories"`
	// MaxPathLength sets maximum allowed path length
	MaxPathLength int `yaml:"max_path_length"`
	// AllowSymlinks determines whether to follow symbolic links
	AllowSymlinks bool `yaml:"allow_symlinks"`
	// RequireRelativePaths determines whether to require relative paths only
	RequireRelativePaths bool `yaml:"require_relative_paths"`
}

// SanitizationConfig configures input sanitization
type SanitizationConfig struct {
	// MaxInputLength sets maximum allowed input length
	MaxInputLength int `yaml:"max_input_length"`
	// EnablePromptInjectionDetection enables/disables prompt injection scanning
	EnablePromptInjectionDetection bool `yaml:"enable_prompt_injection_detection"`
	// PromptInjectionPatterns contains regex patterns for prompt injection detection
	PromptInjectionPatterns []string `yaml:"prompt_injection_patterns"`
	// ContentSizeLimit sets maximum content size for processing
	ContentSizeLimit int `yaml:"content_size_limit"`
	// StrictMode determines whether to use strict validation
	StrictMode bool `yaml:"strict_mode"`
}

// PersonaValidationConfig configures persona validation for meta-pipelines
type PersonaValidationConfig struct {
	// ValidatePersonaReferences enables/disables persona reference validation
	ValidatePersonaReferences bool `yaml:"validate_persona_references"`
	// AllowUnknownPersonas allows unknown personas (for testing)
	AllowUnknownPersonas bool `yaml:"allow_unknown_personas"`
	// PersonaRefreshInterval sets how often to refresh persona list from manifest
	PersonaRefreshInterval time.Duration `yaml:"persona_refresh_interval"`
	// RequirePersonaDescriptions determines if personas must have descriptions
	RequirePersonaDescriptions bool `yaml:"require_persona_descriptions"`
}

// SecurityConfig aggregates all security configuration
type SecurityConfig struct {
	// Enabled determines if security validation is active
	Enabled bool `yaml:"enabled"`
	// LoggingEnabled determines if security events are logged
	LoggingEnabled bool `yaml:"logging_enabled"`
	// PathValidation configures path traversal prevention
	PathValidation PathValidationConfig `yaml:"path_validation"`
	// Sanitization configures input sanitization
	Sanitization SanitizationConfig `yaml:"sanitization"`
	// PersonaValidation configures meta-pipeline persona validation
	PersonaValidation PersonaValidationConfig `yaml:"persona_validation"`
}

// DefaultSecurityConfig returns a secure default configuration
func DefaultSecurityConfig() *SecurityConfig {
	return &SecurityConfig{
		Enabled:        true,
		LoggingEnabled: true,
		PathValidation: PathValidationConfig{
			ApprovedDirectories: []string{
				".wave/contracts/",
				".wave/schemas/",
				"contracts/",
				"schemas/",
			},
			MaxPathLength:        255,
			AllowSymlinks:        false,
			RequireRelativePaths: true,
		},
		Sanitization: SanitizationConfig{
			MaxInputLength:                 10000,
			EnablePromptInjectionDetection: true,
			PromptInjectionPatterns: []string{
				`(?i)ignore.*previous.*instructions?`,
				`(?i)system.*prompt`,
				`(?i)you.*are.*now`,
				`(?i)disregard.*above`,
				`(?i)forget.*instructions?`,
				`(?i)new.*instructions?`,
				`(?i)override.*system`,
			},
			ContentSizeLimit: 1048576, // 1MB
			StrictMode:       true,
		},
		PersonaValidation: PersonaValidationConfig{
			ValidatePersonaReferences:  true,
			AllowUnknownPersonas:       false,
			PersonaRefreshInterval:     5 * time.Minute,
			RequirePersonaDescriptions: false,
		},
	}
}

// Validate checks if the configuration is valid
func (sc *SecurityConfig) Validate() error {
	if sc.PathValidation.MaxPathLength <= 0 {
		return &SecurityValidationError{
			Type:         "configuration_error",
			Message:      "max_path_length must be positive",
			Retryable:    false,
			SuggestedFix: "Set max_path_length to a positive value (e.g., 255)",
		}
	}

	if sc.Sanitization.MaxInputLength <= 0 {
		return &SecurityValidationError{
			Type:         "configuration_error",
			Message:      "max_input_length must be positive",
			Retryable:    false,
			SuggestedFix: "Set max_input_length to a positive value (e.g., 10000)",
		}
	}

	if sc.Sanitization.ContentSizeLimit <= 0 {
		return &SecurityValidationError{
			Type:         "configuration_error",
			Message:      "content_size_limit must be positive",
			Retryable:    false,
			SuggestedFix: "Set content_size_limit to a positive value (e.g., 1048576)",
		}
	}

	return nil
}

// IsPathValidationEnabled returns true if path validation is enabled
func (sc *SecurityConfig) IsPathValidationEnabled() bool {
	return sc.Enabled
}

// IsSanitizationEnabled returns true if input sanitization is enabled
func (sc *SecurityConfig) IsSanitizationEnabled() bool {
	return sc.Enabled && sc.Sanitization.EnablePromptInjectionDetection
}

// IsPersonaValidationEnabled returns true if persona validation is enabled
func (sc *SecurityConfig) IsPersonaValidationEnabled() bool {
	return sc.Enabled && sc.PersonaValidation.ValidatePersonaReferences
}