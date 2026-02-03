package contract

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// TemplateConfig defines a structured template that AI must fill
type TemplateConfig struct {
	Type        string                 `json:"type" yaml:"type"`                 // "json", "markdown", "yaml"
	TemplatePath string                `json:"template_path" yaml:"template_path"` // Path to template file
	Required    []string               `json:"required,omitempty" yaml:"required,omitempty"` // Required fields/sections
	Constraints map[string]interface{} `json:"constraints,omitempty" yaml:"constraints,omitempty"` // Field-specific constraints
}

// TemplateValidator enforces structured template compliance
type TemplateValidator struct{}

// Validate ensures output matches the template structure
func (v *TemplateValidator) Validate(cfg ContractConfig, workspacePath string) error {
	// Load template configuration
	var templateCfg TemplateConfig
	if cfg.SchemaPath != "" {
		data, err := os.ReadFile(cfg.SchemaPath)
		if err != nil {
			return fmt.Errorf("failed to read template config: %w", err)
		}
		if err := json.Unmarshal(data, &templateCfg); err != nil {
			return fmt.Errorf("failed to parse template config: %w", err)
		}
	}

	// Load output file
	outputPath := filepath.Join(workspacePath, cfg.Source)
	outputData, err := os.ReadFile(outputPath)
	if err != nil {
		return fmt.Errorf("failed to read output file: %w", err)
	}

	// Validate based on template type
	switch templateCfg.Type {
	case "json":
		return v.validateJSONTemplate(outputData, templateCfg)
	case "markdown":
		return v.validateMarkdownTemplate(string(outputData), templateCfg)
	case "yaml":
		return v.validateYAMLTemplate(outputData, templateCfg)
	default:
		return fmt.Errorf("unsupported template type: %s", templateCfg.Type)
	}
}

// validateJSONTemplate ensures JSON output follows template structure
func (v *TemplateValidator) validateJSONTemplate(data []byte, cfg TemplateConfig) error {
	var output map[string]interface{}
	if err := json.Unmarshal(data, &output); err != nil {
		return &ValidationError{
			ContractType: "template",
			Message:      "invalid JSON format",
			Details:      []string{err.Error()},
			Retryable:    true,
		}
	}

	violations := []string{}

	// Check required fields
	for _, field := range cfg.Required {
		if _, exists := output[field]; !exists {
			violations = append(violations, fmt.Sprintf("missing required field: %s", field))
		}
	}

	// Check field constraints
	for field, constraint := range cfg.Constraints {
		if value, exists := output[field]; exists {
			if err := v.validateConstraint(field, value, constraint); err != nil {
				violations = append(violations, err.Error())
			}
		}
	}

	if len(violations) > 0 {
		return &ValidationError{
			ContractType: "template",
			Message:      "template validation failed",
			Details:      violations,
			Retryable:    true,
		}
	}

	return nil
}

// validateMarkdownTemplate ensures markdown follows required structure
func (v *TemplateValidator) validateMarkdownTemplate(content string, cfg TemplateConfig) error {
	violations := []string{}

	// Check for required sections (headings)
	for _, section := range cfg.Required {
		pattern := regexp.MustCompile(fmt.Sprintf(`(?im)^#+\s+%s`, regexp.QuoteMeta(section)))
		if !pattern.MatchString(content) {
			violations = append(violations, fmt.Sprintf("missing required section: %s", section))
		}
	}

	if len(violations) > 0 {
		return &ValidationError{
			ContractType: "template",
			Message:      "markdown template validation failed",
			Details:      violations,
			Retryable:    true,
		}
	}

	return nil
}

// validateYAMLTemplate ensures YAML follows template structure
func (v *TemplateValidator) validateYAMLTemplate(data []byte, cfg TemplateConfig) error {
	// Similar to JSON validation but for YAML
	// Implementation would use yaml.Unmarshal
	return nil
}

// validateConstraint checks if a value meets its constraint
func (v *TemplateValidator) validateConstraint(field string, value interface{}, constraint interface{}) error {
	constraintMap, ok := constraint.(map[string]interface{})
	if !ok {
		return nil
	}

	// Check string length constraints
	if minLen, exists := constraintMap["min_length"]; exists {
		if str, ok := value.(string); ok {
			if len(str) < int(minLen.(float64)) {
				return fmt.Errorf("field %s too short (min: %v)", field, minLen)
			}
		}
	}

	if maxLen, exists := constraintMap["max_length"]; exists {
		if str, ok := value.(string); ok {
			if len(str) > int(maxLen.(float64)) {
				return fmt.Errorf("field %s too long (max: %v)", field, maxLen)
			}
		}
	}

	// Check pattern constraints
	if pattern, exists := constraintMap["pattern"]; exists {
		if str, ok := value.(string); ok {
			matched, err := regexp.MatchString(pattern.(string), str)
			if err != nil || !matched {
				return fmt.Errorf("field %s does not match pattern: %s", field, pattern)
			}
		}
	}

	// Check enum constraints
	if enum, exists := constraintMap["enum"]; exists {
		if enumList, ok := enum.([]interface{}); ok {
			found := false
			for _, allowed := range enumList {
				if value == allowed {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("field %s has invalid value (must be one of: %v)", field, enum)
			}
		}
	}

	return nil
}

// GenerateTemplate creates a template file with placeholders
func GenerateTemplate(templateType string, fields []string, outputPath string) error {
	var content string

	switch templateType {
	case "json":
		obj := make(map[string]string)
		for _, field := range fields {
			obj[field] = fmt.Sprintf("[TODO: Fill in %s]", field)
		}
		data, err := json.MarshalIndent(obj, "", "  ")
		if err != nil {
			return err
		}
		content = string(data)

	case "markdown":
		var lines []string
		lines = append(lines, "# Output Document\n")
		for _, field := range fields {
			lines = append(lines, fmt.Sprintf("## %s\n", field))
			lines = append(lines, fmt.Sprintf("[TODO: Fill in %s]\n", field))
		}
		content = strings.Join(lines, "\n")

	default:
		return fmt.Errorf("unsupported template type: %s", templateType)
	}

	return os.WriteFile(outputPath, []byte(content), 0644)
}
