package contract

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// VerificationGate runs automated verification checks on pipeline outputs
type VerificationGate struct{}

func (g *VerificationGate) Name() string {
	return "verification"
}

// Check runs comprehensive verification checks
func (g *VerificationGate) Check(workspacePath string, config QualityGateConfig) ([]QualityViolation, error) {
	violations := []QualityViolation{}

	// Get verification rules from parameters
	rules, ok := config.Parameters["rules"].([]interface{})
	if !ok || len(rules) == 0 {
		return violations, nil
	}

	for _, rule := range rules {
		ruleMap, ok := rule.(map[string]interface{})
		if !ok {
			continue
		}

		ruleType, _ := ruleMap["type"].(string)
		switch ruleType {
		case "link_validation":
			v := g.checkLinks(workspacePath, ruleMap)
			violations = append(violations, v...)
		case "code_compilation":
			v := g.checkCodeCompilation(workspacePath, ruleMap)
			violations = append(violations, v...)
		case "test_execution":
			v := g.checkTestExecution(workspacePath, ruleMap)
			violations = append(violations, v...)
		case "file_existence":
			v := g.checkFileExistence(workspacePath, ruleMap)
			violations = append(violations, v...)
		case "cross_reference":
			v := g.checkCrossReferences(workspacePath, ruleMap)
			violations = append(violations, v...)
		}
	}

	return violations, nil
}

// checkLinks validates that all referenced links are valid
func (g *VerificationGate) checkLinks(workspacePath string, rule map[string]interface{}) []QualityViolation {
	violations := []QualityViolation{}

	target, _ := rule["target"].(string)
	if target == "" {
		return violations
	}

	filePath := filepath.Join(workspacePath, target)
	data, err := os.ReadFile(filePath)
	if err != nil {
		return violations
	}

	var output map[string]interface{}
	if err := json.Unmarshal(data, &output); err != nil {
		return violations
	}

	// Check for URL fields
	brokenLinks := []string{}
	g.findURLs(output, "", &brokenLinks)

	if len(brokenLinks) > 0 {
		violations = append(violations, QualityViolation{
			Gate:     "verification",
			Severity: "warning",
			Message:  fmt.Sprintf("Found %d potentially invalid URLs", len(brokenLinks)),
			Details:  brokenLinks,
			Suggestions: []string{
				"Verify all URLs are accessible",
				"Check for typos in issue/PR numbers",
			},
		})
	}

	return violations
}

// findURLs recursively searches for URL fields in JSON
func (g *VerificationGate) findURLs(obj interface{}, path string, brokenLinks *[]string) {
	switch v := obj.(type) {
	case map[string]interface{}:
		for key, value := range v {
			newPath := key
			if path != "" {
				newPath = path + "." + key
			}
			// Check if this is a URL field
			if strings.Contains(strings.ToLower(key), "url") || strings.Contains(strings.ToLower(key), "link") {
				if str, ok := value.(string); ok && str != "" {
					// Basic URL validation
					if !strings.HasPrefix(str, "http://") && !strings.HasPrefix(str, "https://") {
						*brokenLinks = append(*brokenLinks, fmt.Sprintf("%s: invalid URL format", newPath))
					}
				}
			}
			g.findURLs(value, newPath, brokenLinks)
		}
	case []interface{}:
		for i, item := range v {
			g.findURLs(item, fmt.Sprintf("%s[%d]", path, i), brokenLinks)
		}
	}
}

// checkCodeCompilation verifies that generated code compiles
func (g *VerificationGate) checkCodeCompilation(workspacePath string, rule map[string]interface{}) []QualityViolation {
	violations := []QualityViolation{}

	command, _ := rule["command"].(string)
	if command == "" {
		command = "go build ./..."
	}

	// Split command into parts
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return violations
	}

	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Dir = workspacePath
	output, err := cmd.CombinedOutput()

	if err != nil {
		violations = append(violations, QualityViolation{
			Gate:     "verification",
			Severity: "error",
			Message:  "Code compilation failed",
			Details:  []string{string(output)},
			Suggestions: []string{
				"Fix compilation errors before proceeding",
				"Ensure all imports are correct",
				"Check for syntax errors",
			},
		})
	}

	return violations
}

// checkTestExecution verifies that tests pass
func (g *VerificationGate) checkTestExecution(workspacePath string, rule map[string]interface{}) []QualityViolation {
	violations := []QualityViolation{}

	command, _ := rule["command"].(string)
	if command == "" {
		command = "go test ./..."
	}

	parts := strings.Fields(command)
	if len(parts) == 0 {
		return violations
	}

	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Dir = workspacePath
	output, err := cmd.CombinedOutput()

	if err != nil {
		violations = append(violations, QualityViolation{
			Gate:     "verification",
			Severity: "error",
			Message:  "Test execution failed",
			Details:  []string{string(output)},
			Suggestions: []string{
				"Fix failing tests before proceeding",
				"Ensure test data is properly set up",
				"Check for race conditions with -race flag",
			},
		})
	}

	return violations
}

// checkFileExistence verifies required files exist
func (g *VerificationGate) checkFileExistence(workspacePath string, rule map[string]interface{}) []QualityViolation {
	violations := []QualityViolation{}

	files, ok := rule["files"].([]interface{})
	if !ok {
		return violations
	}

	missingFiles := []string{}
	for _, file := range files {
		fileName, _ := file.(string)
		filePath := filepath.Join(workspacePath, fileName)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			missingFiles = append(missingFiles, fileName)
		}
	}

	if len(missingFiles) > 0 {
		violations = append(violations, QualityViolation{
			Gate:     "verification",
			Severity: "error",
			Message:  "Required files missing",
			Details:  missingFiles,
			Suggestions: []string{
				"Generate all required files",
				"Check file paths are correct",
			},
		})
	}

	return violations
}

// checkCrossReferences verifies that cross-references in documents are valid
func (g *VerificationGate) checkCrossReferences(workspacePath string, rule map[string]interface{}) []QualityViolation {
	violations := []QualityViolation{}

	target, _ := rule["target"].(string)
	if target == "" {
		return violations
	}

	filePath := filepath.Join(workspacePath, target)
	data, err := os.ReadFile(filePath)
	if err != nil {
		return violations
	}

	var output map[string]interface{}
	if err := json.Unmarshal(data, &output); err != nil {
		return violations
	}

	// Check for issue number references
	if relatedIssues, ok := output["related_issues"].([]interface{}); ok {
		if len(relatedIssues) == 0 && shouldHaveRelations(output) {
			violations = append(violations, QualityViolation{
				Gate:     "verification",
				Severity: "warning",
				Message:  "No related issues specified but cross-references expected",
				Suggestions: []string{
					"Link to related issues for context",
					"Add 'Closes #123' references where appropriate",
				},
			})
		}
	}

	return violations
}

// shouldHaveRelations determines if output should have cross-references
func shouldHaveRelations(output map[string]interface{}) bool {
	// Check if this is a PR or issue-related output
	if _, hasPR := output["pr_number"]; hasPR {
		return true
	}
	if _, hasIssue := output["issue_number"]; hasIssue {
		return true
	}
	return false
}

// OutputValidator validates the entire output package for consistency
type OutputValidator struct{}

// ValidatePackage runs comprehensive validation on all outputs
func (v *OutputValidator) ValidatePackage(workspacePath string, expectedOutputs []string) error {
	violations := []string{}

	// Check all expected outputs exist
	for _, output := range expectedOutputs {
		filePath := filepath.Join(workspacePath, output)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			violations = append(violations, fmt.Sprintf("missing expected output: %s", output))
		}
	}

	// Check for unexpected extra files (potential AI hallucination)
	files, _ := os.ReadDir(workspacePath)
	outputMap := make(map[string]bool)
	for _, out := range expectedOutputs {
		outputMap[out] = true
	}

	for _, file := range files {
		if !outputMap[file.Name()] && !file.IsDir() {
			// Allow certain common files
			if file.Name() != ".gitkeep" && !strings.HasPrefix(file.Name(), ".") {
				violations = append(violations, fmt.Sprintf("unexpected output file: %s", file.Name()))
			}
		}
	}

	if len(violations) > 0 {
		return &ValidationError{
			ContractType: "output_package",
			Message:      "output package validation failed",
			Details:      violations,
			Retryable:    false,
		}
	}

	return nil
}
