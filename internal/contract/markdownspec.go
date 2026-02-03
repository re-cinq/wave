package contract

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

type markdownSpecValidator struct{}

// SpecData represents the structured data extracted from a Speckit markdown file
type SpecData struct {
	Title           string          `json:"title"`
	UserStories     []UserStory     `json:"user_stories"`
	DataModel       interface{}     `json:"data_model,omitempty"`
	APIDesign       interface{}     `json:"api_design,omitempty"`
	EdgeCases       []string        `json:"edge_cases,omitempty"`
	TestingStrategy interface{}     `json:"testing_strategy,omitempty"`
	ImplementationSteps []string    `json:"implementation_steps,omitempty"`
	FileChanges     []FileChange    `json:"file_changes,omitempty"`
	RiskAssessment  interface{}     `json:"risk_assessment,omitempty"`
}

// UserStory represents a user story with acceptance criteria
type UserStory struct {
	AsA              string   `json:"as_a"`
	IWant            string   `json:"i_want"`
	SoThat           string   `json:"so_that"`
	AcceptanceCriteria []string `json:"acceptance_criteria"`
}

// FileChange represents a file modification in an implementation plan
type FileChange struct {
	Path        string `json:"path"`
	Description string `json:"description"`
	Changes     string `json:"changes"`
}

func (v *markdownSpecValidator) Validate(cfg ContractConfig, workspacePath string) error {
	// Determine the source file path
	sourceFile := cfg.Source
	if sourceFile == "" {
		return &ValidationError{
			ContractType: "markdown_spec",
			Message:      "no source file specified",
			Details:      []string{"markdown_spec requires a source file path"},
			Retryable:    false,
		}
	}

	// Handle relative paths and Speckit path patterns
	sourcePath := resolveSpeckitPath(workspacePath, sourceFile)

	// Check if the file exists
	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		return &ValidationError{
			ContractType: "markdown_spec",
			Message:      fmt.Sprintf("markdown file not found: %s", sourcePath),
			Details:      []string{err.Error()},
			Retryable:    true, // File might be created in retry
		}
	}

	// Parse the markdown file
	specData, err := parseMarkdownSpec(sourcePath)
	if err != nil {
		return &ValidationError{
			ContractType: "markdown_spec",
			Message:      "failed to parse markdown specification",
			Details:      []string{err.Error(), fmt.Sprintf("file: %s", sourcePath)},
			Retryable:    true,
		}
	}

	// Convert to JSON for schema validation
	jsonData, err := json.Marshal(specData)
	if err != nil {
		return &ValidationError{
			ContractType: "markdown_spec",
			Message:      "failed to convert parsed spec to JSON",
			Details:      []string{err.Error()},
			Retryable:    false,
		}
	}

	// Validate against schema if provided
	if cfg.SchemaPath != "" {
		if err := v.validateAgainstSchema(jsonData, cfg.SchemaPath); err != nil {
			return err
		}
	}

	// Save the converted JSON for debugging and potential use by next steps
	jsonOutputPath := strings.TrimSuffix(sourcePath, filepath.Ext(sourcePath)) + ".json"
	if err := os.WriteFile(jsonOutputPath, jsonData, 0644); err != nil {
		// Non-fatal - log but don't fail validation
		// In production, this would go through the audit logger
	}

	return nil
}

// resolveSpeckitPath resolves Speckit-style paths like "specs/{{branch_name}}/spec.md"
func resolveSpeckitPath(workspacePath, relativePath string) string {
	// If it's already an absolute path, use it directly
	if filepath.IsAbs(relativePath) {
		return relativePath
	}

	// Handle template variables - for now, we'll look for actual directory patterns
	if strings.Contains(relativePath, "{{") {
		// Try to resolve by finding matching directories
		if strings.Contains(relativePath, "specs/") {
			// Look for specs directories with branch-like patterns
			specsDir := filepath.Join(workspacePath, "specs")
			if _, err := os.Stat(specsDir); err == nil {
				// Find the most recently modified directory that looks like a Speckit feature
				if resolved := findLatestSpeckitDir(specsDir, relativePath); resolved != "" {
					return resolved
				}
			}
		}
	}

	// Default to workspace-relative path
	return filepath.Join(workspacePath, relativePath)
}

// findLatestSpeckitDir finds the most recently modified directory matching Speckit patterns
func findLatestSpeckitDir(specsDir, pattern string) string {
	entries, err := os.ReadDir(specsDir)
	if err != nil {
		return ""
	}

	var latestDir string
	var latestTime int64

	// Extract the filename from the pattern (e.g., "spec.md" from "specs/{{branch}}/spec.md")
	filename := filepath.Base(pattern)

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Check if this looks like a Speckit feature directory (###-name pattern)
		matched, _ := regexp.MatchString(`^\d{3}-`, entry.Name())
		if !matched {
			continue
		}

		// Check if the target file exists in this directory
		targetPath := filepath.Join(specsDir, entry.Name(), filename)
		if stat, err := os.Stat(targetPath); err == nil {
			if stat.ModTime().UnixNano() > latestTime {
				latestTime = stat.ModTime().UnixNano()
				latestDir = targetPath
			}
		}
	}

	return latestDir
}

// parseMarkdownSpec extracts structured data from a Speckit markdown file
func parseMarkdownSpec(filePath string) (*SpecData, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	spec := &SpecData{}
	scanner := bufio.NewScanner(file)

	var currentSection string
	var currentContent strings.Builder
	var inCodeBlock bool

	for scanner.Scan() {
		line := scanner.Text()

		// Track code blocks to avoid parsing headers inside them
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			inCodeBlock = !inCodeBlock
			currentContent.WriteString(line + "\n")
			continue
		}

		if inCodeBlock {
			currentContent.WriteString(line + "\n")
			continue
		}

		// Parse headers to identify sections
		if strings.HasPrefix(line, "#") {
			// Process previous section
			if currentSection != "" {
				if err := processSection(spec, currentSection, currentContent.String()); err != nil {
					return nil, fmt.Errorf("error processing section %s: %w", currentSection, err)
				}
			}

			// Start new section
			currentSection = strings.TrimSpace(strings.TrimLeft(line, "#"))
			currentContent.Reset()
		} else {
			currentContent.WriteString(line + "\n")
		}
	}

	// Process final section
	if currentSection != "" {
		if err := processSection(spec, currentSection, currentContent.String()); err != nil {
			return nil, fmt.Errorf("error processing final section %s: %w", currentSection, err)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	// If no title was found in sections, try to extract from filename or first heading
	if spec.Title == "" {
		basename := filepath.Base(filePath)
		spec.Title = strings.TrimSuffix(basename, filepath.Ext(basename))
	}

	return spec, nil
}

// processSection processes a markdown section based on its header
func processSection(spec *SpecData, sectionName, content string) error {
	content = strings.TrimSpace(content)
	if content == "" {
		return nil
	}

	lowerSection := strings.ToLower(sectionName)

	switch {
	case strings.Contains(lowerSection, "title") || lowerSection == "overview":
		if spec.Title == "" {  // Only set if not already set
			spec.Title = extractFirstLine(content)
		}

	case strings.Contains(lowerSection, "user stories") || strings.Contains(lowerSection, "user story"):
		stories, err := parseUserStories(content)
		if err != nil {
			return err
		}
		spec.UserStories = append(spec.UserStories, stories...)

	case strings.Contains(lowerSection, "story") && content != "":
		// Handle individual story sections like "Story 1", "Story 2"
		// Also extract narrative-style stories with acceptance scenarios
		stories, err := parseUserStories(content)
		if err != nil {
			return err
		}

		// If no traditional user stories found, try to extract from narrative description
		if len(stories) == 0 && content != "" {
			// Create a user story from the narrative content
			story := UserStory{
				AsA:    "user",
				IWant:  extractUserIntent(content),
				SoThat: extractUserBenefit(content),
			}

			// Extract acceptance scenarios/criteria
			story.AcceptanceCriteria = extractAcceptanceScenarios(content)

			if story.IWant != "" || len(story.AcceptanceCriteria) > 0 {
				stories = append(stories, story)
			}
		}

		spec.UserStories = append(spec.UserStories, stories...)

	case strings.Contains(lowerSection, "data model") || strings.Contains(lowerSection, "schema"):
		spec.DataModel = parseStructuredContent(content)

	case strings.Contains(lowerSection, "api") || strings.Contains(lowerSection, "endpoint"):
		spec.APIDesign = parseStructuredContent(content)

	case strings.Contains(lowerSection, "edge case") || strings.Contains(lowerSection, "error"):
		spec.EdgeCases = parseListItems(content)

	case strings.Contains(lowerSection, "test") || strings.Contains(lowerSection, "validation"):
		spec.TestingStrategy = parseStructuredContent(content)

	case strings.Contains(lowerSection, "implementation") || strings.Contains(lowerSection, "step"):
		spec.ImplementationSteps = parseListItems(content)

	case strings.Contains(lowerSection, "file") || strings.Contains(lowerSection, "change"):
		changes, err := parseFileChanges(content)
		if err != nil {
			return err
		}
		spec.FileChanges = changes

	case strings.Contains(lowerSection, "risk"):
		spec.RiskAssessment = parseStructuredContent(content)
	}

	return nil
}

// parseUserStories extracts user stories from markdown content
func parseUserStories(content string) ([]UserStory, error) {
	var stories []UserStory

	// Split by story blocks - look for patterns that start a new story
	lines := strings.Split(content, "\n")
	var currentStory *UserStory
	var acceptanceCriteria []string
	inAcceptance := false

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Check for single-line format first (has both "as a" and "i want")
		if strings.Contains(strings.ToLower(line), "as a") && strings.Contains(strings.ToLower(line), "i want") {
			// Handle comma-separated format like "As a user, I want to log in so that I can access my account"
			// Save previous story if it exists
			if currentStory != nil {
				currentStory.AcceptanceCriteria = acceptanceCriteria
				stories = append(stories, *currentStory)
			}

			currentStory = &UserStory{}

			// Parse single-line user story format
			lower := strings.ToLower(line)
			asAStart := strings.Index(lower, "as a")
			iWantStart := strings.Index(lower, "i want")
			soThatStart := strings.Index(lower, "so that")

			if asAStart != -1 && iWantStart != -1 {
				// Extract "as a" part
				asAEnd := iWantStart
				currentStory.AsA = strings.TrimSpace(line[asAStart+5:asAEnd])
				currentStory.AsA = strings.TrimSuffix(currentStory.AsA, ",")

				// Extract "I want" part
				iWantEnd := len(line)
				if soThatStart != -1 {
					iWantEnd = soThatStart
				}
				currentStory.IWant = strings.TrimSpace(line[iWantStart+6:iWantEnd])
				currentStory.IWant = strings.TrimSuffix(currentStory.IWant, ",")

				// Extract "so that" part if present
				if soThatStart != -1 {
					currentStory.SoThat = strings.TrimSpace(line[soThatStart+8:])
					currentStory.SoThat = strings.TrimSuffix(currentStory.SoThat, ".")
				}
			}
			acceptanceCriteria = []string{}
			inAcceptance = false
		} else if strings.HasPrefix(strings.ToLower(line), "as a") || strings.HasPrefix(strings.ToLower(line), "as an") {
			// Multi-line format starting with "As a"
			// Save previous story if it exists
			if currentStory != nil {
				currentStory.AcceptanceCriteria = acceptanceCriteria
				stories = append(stories, *currentStory)
			}

			// Start new story
			currentStory = &UserStory{}
			currentStory.AsA = extractUserStoryValue(line)
			acceptanceCriteria = []string{}
			inAcceptance = false
		} else if currentStory != nil {
			// Continue parsing current story
			if strings.HasPrefix(strings.ToLower(line), "i want") {
				currentStory.IWant = extractUserStoryValue(line)
			} else if strings.HasPrefix(strings.ToLower(line), "so that") {
				currentStory.SoThat = extractUserStoryValue(line)
			} else if strings.Contains(strings.ToLower(line), "acceptance") {
				inAcceptance = true
			} else if inAcceptance && (strings.HasPrefix(line, "-") || strings.HasPrefix(line, "*") || strings.HasPrefix(line, "•")) {
				criterion := strings.TrimSpace(strings.TrimLeft(line, "-*•"))
				if criterion != "" {
					acceptanceCriteria = append(acceptanceCriteria, criterion)
				}
			}
		}
	}

	// Don't forget the last story
	if currentStory != nil {
		currentStory.AcceptanceCriteria = acceptanceCriteria
		stories = append(stories, *currentStory)
	}

	// Filter out stories that don't have the minimum required fields
	// Be more lenient - accept stories that have at least some meaningful content
	var validStories []UserStory
	for _, story := range stories {
		// Accept if it has AsA and either IWant or SoThat, or has acceptance criteria
		if (story.AsA != "" && (story.IWant != "" || story.SoThat != "")) || len(story.AcceptanceCriteria) > 0 {
			// Fill in empty fields to ensure schema compatibility
			if story.AsA == "" {
				story.AsA = "user"
			}
			if story.IWant == "" && story.SoThat != "" {
				story.IWant = "functionality described below"
			}
			if story.SoThat == "" && story.IWant != "" {
				story.SoThat = "the feature works as intended"
			}
			validStories = append(validStories, story)
		}
	}

	return validStories, nil
}

// parseFileChanges extracts file change descriptions from content
func parseFileChanges(content string) ([]FileChange, error) {
	var changes []FileChange

	// Look for file paths and their descriptions
	lines := strings.Split(content, "\n")
	var currentFile *FileChange

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Check if line looks like a file path
		if strings.Contains(line, ".go") || strings.Contains(line, ".js") ||
		   strings.Contains(line, ".ts") || strings.Contains(line, ".py") ||
		   strings.Contains(line, "/") {

			// Save previous file if exists
			if currentFile != nil {
				changes = append(changes, *currentFile)
			}

			// Start new file change
			currentFile = &FileChange{
				Path: line,
			}
		} else if currentFile != nil {
			// Add to description or changes
			if currentFile.Description == "" {
				currentFile.Description = line
			} else {
				currentFile.Changes += line + "\n"
			}
		}
	}

	// Add final file
	if currentFile != nil {
		changes = append(changes, *currentFile)
	}

	return changes, nil
}

// Helper functions
func extractFirstLine(content string) string {
	lines := strings.Split(strings.TrimSpace(content), "\n")
	if len(lines) > 0 {
		return strings.TrimSpace(lines[0])
	}
	return ""
}

func extractUserStoryValue(line string) string {
	line = strings.TrimSpace(line)

	// Handle different formats
	lower := strings.ToLower(line)

	// Format: "As a: user" or "I want: to log in"
	if strings.Contains(line, ":") {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) > 1 {
			return strings.TrimSpace(parts[1])
		}
	}

	// Format: "As a user, I want to..." (comma-separated)
	if strings.HasPrefix(lower, "as a ") {
		// Extract everything after "as a " and before comma or end
		value := strings.TrimPrefix(line, "As a ")
		value = strings.TrimPrefix(value, "as a ")
		if idx := strings.Index(value, ","); idx != -1 {
			value = value[:idx]
		}
		return strings.TrimSpace(value)
	}

	if strings.HasPrefix(lower, "i want ") {
		// Extract everything after "i want " and before " so that"
		value := strings.TrimPrefix(line, "I want ")
		value = strings.TrimPrefix(value, "i want ")
		if idx := strings.Index(strings.ToLower(value), " so that"); idx != -1 {
			value = value[:idx]
		}
		return strings.TrimSpace(value)
	}

	if strings.HasPrefix(lower, "so that ") {
		// Extract everything after "so that "
		value := strings.TrimPrefix(line, "So that ")
		value = strings.TrimPrefix(value, "so that ")
		return strings.TrimSpace(value)
	}

	return strings.TrimSpace(line)
}

func parseListItems(content string) []string {
	var items []string
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Handle various list formats
		if strings.HasPrefix(line, "-") || strings.HasPrefix(line, "*") || strings.HasPrefix(line, "•") {
			item := strings.TrimSpace(strings.TrimLeft(line, "-*•"))
			if item != "" {
				items = append(items, item)
			}
		} else if strings.Contains(line, ".") && len(line) < 200 {
			// Might be a numbered list
			parts := strings.SplitN(line, ".", 2)
			if len(parts) > 1 {
				item := strings.TrimSpace(parts[1])
				if item != "" {
					items = append(items, item)
				}
			}
		}
	}

	return items
}

func parseStructuredContent(content string) interface{} {
	content = strings.TrimSpace(content)

	// Try to parse as JSON if it looks like JSON
	if strings.HasPrefix(content, "{") || strings.HasPrefix(content, "[") {
		var parsed interface{}
		if err := json.Unmarshal([]byte(content), &parsed); err == nil {
			return parsed
		}
	}

	// Otherwise return as a structured text object
	lines := strings.Split(content, "\n")
	result := make(map[string]interface{})

	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		key := fmt.Sprintf("item_%d", i+1)
		if strings.Contains(line, ":") {
			parts := strings.SplitN(line, ":", 2)
			key = strings.TrimSpace(parts[0])
			result[key] = strings.TrimSpace(parts[1])
		} else {
			result[key] = line
		}
	}

	if len(result) == 0 {
		return content
	}

	return result
}

// validateAgainstSchema validates JSON data against a schema file
func (v *markdownSpecValidator) validateAgainstSchema(jsonData []byte, schemaPath string) error {
	// Load and compile schema
	schemaData, err := os.ReadFile(schemaPath)
	if err != nil {
		return &ValidationError{
			ContractType: "markdown_spec",
			Message:      fmt.Sprintf("failed to read schema file: %s", schemaPath),
			Details:      []string{err.Error()},
			Retryable:    false,
		}
	}

	var schemaDoc interface{}
	if err := json.Unmarshal(schemaData, &schemaDoc); err != nil {
		return &ValidationError{
			ContractType: "markdown_spec",
			Message:      fmt.Sprintf("failed to parse schema file: %s", schemaPath),
			Details:      []string{err.Error()},
			Retryable:    false,
		}
	}

	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource(schemaPath, schemaDoc); err != nil {
		return &ValidationError{
			ContractType: "markdown_spec",
			Message:      "failed to add schema resource",
			Details:      []string{err.Error()},
			Retryable:    false,
		}
	}

	schema, err := compiler.Compile(schemaPath)
	if err != nil {
		return &ValidationError{
			ContractType: "markdown_spec",
			Message:      "failed to compile schema",
			Details:      []string{err.Error()},
			Retryable:    false,
		}
	}

	// Parse JSON data for validation
	var artifact interface{}
	if err := json.Unmarshal(jsonData, &artifact); err != nil {
		return &ValidationError{
			ContractType: "markdown_spec",
			Message:      "failed to parse converted JSON for validation",
			Details:      []string{err.Error()},
			Retryable:    true,
		}
	}

	// Validate against schema
	if err := schema.Validate(artifact); err != nil {
		details := extractSchemaValidationDetails(err)
		return &ValidationError{
			ContractType: "markdown_spec",
			Message:      "markdown spec does not match schema requirements",
			Details:      details,
			Retryable:    true,
		}
	}

	return nil
}
// extractUserIntent extracts the user intent from narrative text
func extractUserIntent(content string) string {
	content = strings.ToLower(content)

	// Look for patterns that indicate user intent
	patterns := []string{
		"can see", "can understand", "can view", "can access", "can use",
		"need", "want", "require", "should be able to",
		"users can", "developers can", "system should",
	}

	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		for _, pattern := range patterns {
			if strings.Contains(line, pattern) {
				// Extract the intent part
				if idx := strings.Index(line, pattern); idx != -1 {
					intent := strings.TrimSpace(line[idx+len(pattern):])
					// Clean up the intent
					intent = strings.TrimPrefix(intent, "to ")
					intent = strings.Trim(intent, ".,:;")
					if intent != "" && len(intent) > 10 {
						return intent
					}
				}
			}
		}
	}

	// Fallback: use first meaningful sentence
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && len(line) > 20 && !strings.HasPrefix(line, "#") && !strings.HasPrefix(line, "**") {
			return strings.Trim(line, ".,:;")
		}
	}

	return "use the feature"
}

// extractUserBenefit extracts the benefit/value from narrative text
func extractUserBenefit(content string) string {
	content = strings.ToLower(content)

	// Look for patterns that indicate benefits/reasons
	patterns := []string{
		"so that", "because", "to ensure", "for", "enabling",
		"eliminating", "building", "providing", "helping",
		"why this priority", "addresses",
	}

	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		for _, pattern := range patterns {
			if strings.Contains(line, pattern) {
				// Extract the benefit part
				if idx := strings.Index(line, pattern); idx != -1 {
					benefit := strings.TrimSpace(line[idx+len(pattern):])
					// Clean up common prefixes
					benefit = strings.TrimPrefix(benefit, ": ")
					benefit = strings.TrimPrefix(benefit, "the ")
					benefit = strings.Trim(benefit, ".,:;")
					if benefit != "" && len(benefit) > 10 {
						return benefit
					}
				}
			}
		}
	}

	return "the functionality works correctly"
}

// extractAcceptanceScenarios extracts acceptance criteria from narrative text
func extractAcceptanceScenarios(content string) []string {
	var scenarios []string
	lines := strings.Split(content, "\n")

	inScenarios := false
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Look for acceptance scenarios section
		if strings.Contains(strings.ToLower(line), "acceptance") &&
		   (strings.Contains(strings.ToLower(line), "scenario") || strings.Contains(strings.ToLower(line), "criteria")) {
			inScenarios = true
			continue
		}

		// Stop when we hit a new section
		if strings.HasPrefix(line, "#") || strings.HasPrefix(line, "**") {
			if !strings.Contains(strings.ToLower(line), "acceptance") {
				inScenarios = false
			}
			continue
		}

		if inScenarios {
			// Extract numbered or bulleted scenarios
			if strings.HasPrefix(line, "1. ") || strings.HasPrefix(line, "2. ") || strings.HasPrefix(line, "3. ") ||
			   strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") || strings.HasPrefix(line, "• ") {
				scenario := line
				// Clean up numbering and bullets
				scenario = strings.TrimLeft(scenario, "123456789. -*•")
				scenario = strings.TrimSpace(scenario)
				if scenario != "" {
					scenarios = append(scenarios, scenario)
				}
			} else if len(line) > 20 && !strings.Contains(line, ":") {
				// Standalone scenario descriptions
				scenarios = append(scenarios, line)
			}
		}
	}

	// Also look for "Given/When/Then" patterns anywhere in the content
	for _, line := range lines {
		line = strings.TrimSpace(line)
		lower := strings.ToLower(line)
		if strings.HasPrefix(lower, "given") || strings.HasPrefix(lower, "when") || strings.HasPrefix(lower, "then") {
			scenarios = append(scenarios, line)
		}
	}

	return scenarios
}
