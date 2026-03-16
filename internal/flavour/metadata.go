package flavour

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// MetadataResult holds extracted project metadata.
type MetadataResult struct {
	Name        string
	Description string
}

// DetectMetadata extracts project name and description from language-specific
// manifest files in the given directory. Returns zero-value MetadataResult
// if no metadata can be extracted; falls back to directory name for the project name.
func DetectMetadata(dir string) MetadataResult {
	// Try go.mod
	if result, ok := detectGoMetadata(dir); ok {
		return result
	}

	// Try package.json
	if result, ok := detectPackageJSONMetadata(dir); ok {
		return result
	}

	// Try Cargo.toml
	if result, ok := detectCargoMetadata(dir); ok {
		return result
	}

	// Try pyproject.toml
	if result, ok := detectPyprojectMetadata(dir); ok {
		return result
	}

	// Fallback: use directory name
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return MetadataResult{}
	}
	return MetadataResult{Name: filepath.Base(absDir)}
}

func detectGoMetadata(dir string) (MetadataResult, bool) {
	data, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		return MetadataResult{}, false
	}

	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "module ") {
			modulePath := strings.TrimPrefix(line, "module ")
			modulePath = strings.TrimSpace(modulePath)
			// Extract repo name from last path segment
			parts := strings.Split(modulePath, "/")
			name := parts[len(parts)-1]
			return MetadataResult{Name: name}, true
		}
	}
	return MetadataResult{}, false
}

func detectPackageJSONMetadata(dir string) (MetadataResult, bool) {
	data, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		return MetadataResult{}, false
	}

	var pkg struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return MetadataResult{}, false
	}
	if pkg.Name == "" {
		return MetadataResult{}, false
	}
	return MetadataResult{Name: pkg.Name, Description: pkg.Description}, true
}

func detectCargoMetadata(dir string) (MetadataResult, bool) {
	data, err := os.ReadFile(filepath.Join(dir, "Cargo.toml"))
	if err != nil {
		return MetadataResult{}, false
	}

	result := MetadataResult{}
	inPackage := false
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "[package]" {
			inPackage = true
			continue
		}
		if strings.HasPrefix(line, "[") {
			inPackage = false
			continue
		}
		if !inPackage {
			continue
		}
		if strings.HasPrefix(line, "name") {
			result.Name = extractTOMLStringValue(line)
		}
		if strings.HasPrefix(line, "description") {
			result.Description = extractTOMLStringValue(line)
		}
	}

	if result.Name == "" {
		return MetadataResult{}, false
	}
	return result, true
}

func detectPyprojectMetadata(dir string) (MetadataResult, bool) {
	data, err := os.ReadFile(filepath.Join(dir, "pyproject.toml"))
	if err != nil {
		return MetadataResult{}, false
	}

	result := MetadataResult{}
	inSection := false
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "[project]" || line == "[tool.poetry]" {
			inSection = true
			continue
		}
		if strings.HasPrefix(line, "[") {
			inSection = false
			continue
		}
		if !inSection {
			continue
		}
		if strings.HasPrefix(line, "name") {
			result.Name = extractTOMLStringValue(line)
		}
		if strings.HasPrefix(line, "description") {
			result.Description = extractTOMLStringValue(line)
		}
	}

	if result.Name == "" {
		return MetadataResult{}, false
	}
	return result, true
}

// extractTOMLStringValue extracts a string value from a TOML key = "value" line.
func extractTOMLStringValue(line string) string {
	parts := strings.SplitN(line, "=", 2)
	if len(parts) != 2 {
		return ""
	}
	val := strings.TrimSpace(parts[1])
	val = strings.Trim(val, "\"")
	return val
}
