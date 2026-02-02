# Data Model: Critical Security Vulnerability Fixes

**Phase 1 Data Model** | **Date**: 2026-02-02
**Feature**: Security validation entities and structures

## Core Entities

### SecurityViolationEvent

**Purpose**: Records detected security attempts for auditing and monitoring

**Fields**:
- `ID`: string (UUID) - Unique identifier for the violation event
- `Timestamp`: time.Time - When the violation was detected
- `Type`: string - Category of violation (path_traversal, prompt_injection, invalid_persona, etc.)
- `Source`: string - Source of the violation (schema_path, user_input, meta_pipeline, etc.)
- `SanitizedDetails`: string - Safe description of the violation without exposing sensitive data
- `Severity`: string - LOW/MEDIUM/HIGH/CRITICAL
- `Blocked`: bool - Whether the operation was successfully blocked
- `UserID`: string (optional) - User context if available

**Validation Rules**:
- Type must be from predefined enum
- SanitizedDetails must not contain sensitive paths or content
- Severity must match type-specific rules

**State Transitions**: N/A (immutable audit records)

### SchemaValidationResult

**Purpose**: Contains outcome of schema validation with security considerations

**Fields**:
- `SchemaPath`: string - Original requested schema path
- `ValidatedPath`: string - Cleaned and validated path actually used
- `Content`: string - Sanitized schema content safe for AI processing
- `SecurityFlags`: []string - List of security concerns detected
- `IsValid`: bool - Whether schema passed validation
- `ErrorMessage`: string - Human-readable error if validation failed
- `SanitizationActions`: []string - List of sanitization operations performed

**Validation Rules**:
- ValidatedPath must be within approved directories
- Content must be valid JSON
- SecurityFlags must be from predefined security concern types

**Relationships**:
- Related to SecurityViolationEvent if violations detected

### PersonaReference

**Purpose**: Links pipeline steps to validated manifest personas

**Fields**:
- `StepID`: string - Pipeline step identifier
- `PersonaName`: string - Requested persona name
- `IsValid`: bool - Whether persona exists in current manifest
- `AvailablePersonas`: []string - List of valid personas from manifest
- `SuggestedAlternative`: string (optional) - Closest valid persona if invalid requested

**Validation Rules**:
- PersonaName must exist in manifest if IsValid is true
- AvailablePersonas must reflect current manifest state
- SuggestedAlternative must be from AvailablePersonas

**Relationships**:
- Maps to manifest persona definitions
- References pipeline step configuration

### InputSanitizationRecord

**Purpose**: Tracks sanitization actions for user input and content

**Fields**:
- `InputHash`: string - Hash of original input for tracking (not the input itself)
- `InputType`: string - Type of input (task_description, schema_content, pipeline_yaml, etc.)
- `SanitizationRules`: []string - List of sanitization rules applied
- `ChangesDetected`: bool - Whether any modifications were made
- `SanitizedLength`: int - Length of content after sanitization
- `OriginalLength`: int - Original content length
- `RiskScore`: int - Calculated risk score (0-100)

**Validation Rules**:
- InputHash must be SHA-256 hash
- InputType must be from predefined enum
- RiskScore must be 0-100
- SanitizedLength must be <= OriginalLength

**State Transitions**: N/A (immutable sanitization records)

## Security Configuration

### PathValidationConfig

**Purpose**: Configuration for path traversal prevention

**Fields**:
- `ApprovedDirectories`: []string - Allowlisted base directories for schema files
- `MaxPathLength`: int - Maximum allowed path length
- `AllowSymlinks`: bool - Whether to follow symbolic links
- `RequireRelativePaths`: bool - Whether to require relative paths only

### SanitizationConfig

**Purpose**: Configuration for input sanitization

**Fields**:
- `MaxInputLength`: int - Maximum allowed input length
- `EnablePromptInjectionDetection`: bool - Whether to scan for prompt injection
- `PromptInjectionPatterns`: []string - Regex patterns for prompt injection detection
- `ContentSizeLimit`: int - Maximum content size for processing
- `StrictMode`: bool - Whether to use strict validation

### PersonaValidationConfig

**Purpose**: Configuration for persona validation in meta-pipelines

**Fields**:
- `ValidatePersonaReferences`: bool - Whether to validate persona references
- `AllowUnknownPersonas`: bool - Whether to allow unknown personas (for testing)
- `PersonaRefreshInterval`: duration - How often to refresh persona list from manifest
- `RequirePersonaDescriptions`: bool - Whether personas must have descriptions

## Error Types

### SecurityValidationError

**Purpose**: Structured error for security validation failures

**Fields**:
- `Type`: string - Error type (path_traversal, prompt_injection, etc.)
- `Message`: string - Human-readable error message
- `Details`: map[string]interface{} - Additional error context (sanitized)
- `Retryable`: bool - Whether operation can be retried
- `SuggestedFix`: string - Suggested remediation action

## Integration Points

### Pipeline Integration

- SecurityValidationError integrates with existing pipeline error handling
- SecurityViolationEvent uses existing Wave event emission system
- PathValidationConfig extends manifest configuration structure

### Logging Integration

- SecurityViolationEvent records flow through existing audit logging
- Sanitization records available for debugging and compliance
- Error details sanitized to prevent log injection attacks

### Testing Integration

- All entities support table-driven tests with realistic attack scenarios
- Mock configurations for testing edge cases
- Validation helpers for security test assertions