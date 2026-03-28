package skill

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// Skill represents a parsed SKILL.md file per the Agent Skills Specification.
type Skill struct {
	Name          string
	Description   string
	Body          string
	License       string
	Compatibility string
	CheckCommand  string
	Metadata      map[string]string
	AllowedTools  []string
	SourcePath    string
	ResourcePaths []string
}

// frontmatter is the raw YAML structure for SKILL.md frontmatter.
type frontmatter struct {
	Name          string            `yaml:"name"`
	Description   string            `yaml:"description"`
	License       string            `yaml:"license,omitempty"`
	Compatibility string            `yaml:"compatibility,omitempty"`
	CheckCommand  string            `yaml:"check_command,omitempty"`
	Metadata      map[string]string `yaml:"metadata,omitempty"`
	AllowedTools  string            `yaml:"allowed-tools,omitempty"`
}

var nameRegex = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`)

// ValidateName checks if a skill name conforms to the naming rules.
func ValidateName(name string) error {
	if name == "" {
		return &ParseError{Field: "name", Constraint: "required"}
	}
	if len(name) > 64 {
		return &ParseError{Field: "name", Constraint: "max 64 characters", Value: name}
	}
	if !nameRegex.MatchString(name) {
		return &ParseError{Field: "name", Constraint: "must match ^[a-z0-9]([a-z0-9-]*[a-z0-9])?$", Value: name}
	}
	return nil
}

// splitFrontmatter splits SKILL.md content into YAML frontmatter and markdown body.
func splitFrontmatter(data []byte) (yamlBlock []byte, body string, err error) {
	const delimiter = "---"

	s := string(data)

	// Must start with ---
	if !strings.HasPrefix(s, delimiter) {
		return nil, "", &ParseError{Field: "frontmatter", Constraint: "must start with ---"}
	}

	// Find closing ---
	rest := s[len(delimiter):]
	// Skip the newline after opening ---
	switch {
	case len(rest) == 0:
		return nil, "", &ParseError{Field: "frontmatter", Constraint: "unterminated frontmatter"}
	case rest[0] == '\n':
		rest = rest[1:]
	case len(rest) > 1 && rest[0] == '\r' && rest[1] == '\n':
		rest = rest[2:]
	}

	idx := strings.Index(rest, "\n"+delimiter)
	if idx < 0 {
		// Check if it ends with --- (no trailing newline)
		if strings.HasSuffix(rest, delimiter) {
			yamlContent := rest[:len(rest)-len(delimiter)]
			return []byte(yamlContent), "", nil
		}
		return nil, "", &ParseError{Field: "frontmatter", Constraint: "unterminated frontmatter"}
	}

	yamlContent := rest[:idx]
	remaining := rest[idx+1+len(delimiter):]

	// Strip leading newline from body
	if len(remaining) > 0 && remaining[0] == '\n' {
		remaining = remaining[1:]
	} else if len(remaining) > 1 && remaining[0] == '\r' && remaining[1] == '\n' {
		remaining = remaining[2:]
	}

	return []byte(yamlContent), remaining, nil
}

// ValidateFields checks name, description, and compatibility constraints.
// Used by both Parse (via validateFrontmatter) and Serialize for consistent validation.
func ValidateFields(name, description, compatibility string) error {
	if err := ValidateName(name); err != nil {
		return err
	}
	if description == "" {
		return &ParseError{Field: "description", Constraint: "required"}
	}
	if len(description) > 1024 {
		return &ParseError{Field: "description", Constraint: "max 1024 characters", Value: description[:50] + "..."}
	}
	if compatibility != "" && len(compatibility) > 500 {
		return &ParseError{Field: "compatibility", Constraint: "max 500 characters"}
	}
	return nil
}

// validateFrontmatter validates the parsed frontmatter fields.
func validateFrontmatter(fm *frontmatter) error {
	return ValidateFields(fm.Name, fm.Description, fm.Compatibility)
}

// parseFrontmatter splits and validates frontmatter, returning the parsed
// frontmatter and the markdown body. Splits only once.
func parseFrontmatter(data []byte) (frontmatter, string, error) {
	yamlBlock, body, err := splitFrontmatter(data)
	if err != nil {
		return frontmatter{}, "", err
	}

	var fm frontmatter
	if err := yaml.Unmarshal(yamlBlock, &fm); err != nil {
		return frontmatter{}, "", fmt.Errorf("failed to parse YAML frontmatter: %w", err)
	}

	if err := validateFrontmatter(&fm); err != nil {
		return frontmatter{}, "", err
	}

	return fm, body, nil
}

// frontmatterToSkill converts validated frontmatter to a Skill.
func frontmatterToSkill(fm frontmatter, body string) Skill {
	var allowedTools []string
	if fm.AllowedTools != "" {
		allowedTools = strings.Fields(fm.AllowedTools)
	}

	return Skill{
		Name:          fm.Name,
		Description:   fm.Description,
		Body:          body,
		License:       fm.License,
		Compatibility: fm.Compatibility,
		CheckCommand:  fm.CheckCommand,
		Metadata:      fm.Metadata,
		AllowedTools:  allowedTools,
	}
}

// Parse parses SKILL.md content from raw bytes.
func Parse(data []byte) (Skill, error) {
	fm, body, err := parseFrontmatter(data)
	if err != nil {
		return Skill{}, err
	}

	return frontmatterToSkill(fm, body), nil
}

// ParseMetadata parses only the frontmatter from SKILL.md content.
// The returned Skill has an empty Body field.
func ParseMetadata(data []byte) (Skill, error) {
	fm, _, err := parseFrontmatter(data)
	if err != nil {
		return Skill{}, err
	}

	return frontmatterToSkill(fm, ""), nil
}

// Serialize converts a Skill back to SKILL.md format.
func Serialize(skill Skill) ([]byte, error) {
	if err := ValidateFields(skill.Name, skill.Description, skill.Compatibility); err != nil {
		return nil, err
	}

	fm := frontmatter{
		Name:          skill.Name,
		Description:   skill.Description,
		License:       skill.License,
		Compatibility: skill.Compatibility,
		CheckCommand:  skill.CheckCommand,
		Metadata:      skill.Metadata,
	}
	if len(skill.AllowedTools) > 0 {
		fm.AllowedTools = strings.Join(skill.AllowedTools, " ")
	}

	var buf bytes.Buffer
	buf.WriteString("---\n")

	yamlData, err := yaml.Marshal(&fm)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal frontmatter: %w", err)
	}
	buf.Write(yamlData)
	buf.WriteString("---\n")

	if skill.Body != "" {
		buf.WriteString(skill.Body)
	}

	return buf.Bytes(), nil
}
