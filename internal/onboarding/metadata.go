package onboarding

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// ProjectMetadata holds the extracted name and description of a project.
type ProjectMetadata struct {
	Name        string
	Description string
}

// ExtractProjectMetadata attempts to extract project name and description from
// language-specific manifest files found in dir. It checks files in a defined
// priority order and returns the first successful result. Returns an empty
// ProjectMetadata if nothing is found or on any parse error.
func ExtractProjectMetadata(dir string) ProjectMetadata {
	type extractor func(string) ProjectMetadata

	checks := []struct {
		filename string
		fn       extractor
	}{
		{"go.mod", extractGoMod},
		{"Cargo.toml", extractCargoToml},
		{"package.json", extractPackageJSON},
		{"pyproject.toml", extractPyprojectToml},
		{"composer.json", extractComposerJSON},
		{"pubspec.yaml", extractPubspecYaml},
		{"mix.exs", extractMixExs},
		{"Package.swift", extractPackageSwift},
	}

	for _, check := range checks {
		path := filepath.Join(dir, check.filename)
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		if meta := check.fn(string(data)); meta.Name != "" {
			return meta
		}
	}

	return ProjectMetadata{}
}

// extractGoMod parses a go.mod file and extracts the module name as the last
// segment of the module path.
func extractGoMod(content string) ProjectMetadata {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "module ") {
			continue
		}
		modulePath := strings.TrimSpace(strings.TrimPrefix(line, "module"))
		if modulePath == "" {
			return ProjectMetadata{}
		}
		parts := strings.Split(modulePath, "/")
		name := parts[len(parts)-1]
		if name == "" {
			return ProjectMetadata{}
		}
		return ProjectMetadata{Name: name}
	}
	return ProjectMetadata{}
}

// extractCargoToml parses a Cargo.toml file and extracts the package name from
// the [package] section.
func extractCargoToml(content string) ProjectMetadata {
	inPackage := false
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "[package]" {
			inPackage = true
			continue
		}
		if inPackage && strings.HasPrefix(trimmed, "[") {
			// Entered a new section — stop looking.
			break
		}
		if !inPackage {
			continue
		}
		if !strings.HasPrefix(trimmed, "name") {
			continue
		}
		name := extractTomlStringValue(trimmed, "name")
		if name != "" {
			return ProjectMetadata{Name: name}
		}
	}
	return ProjectMetadata{}
}

// extractPackageJSON parses a package.json file and extracts the name and
// description fields.
func extractPackageJSON(content string) ProjectMetadata {
	var pkg struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := json.Unmarshal([]byte(content), &pkg); err != nil {
		return ProjectMetadata{}
	}
	return ProjectMetadata{Name: pkg.Name, Description: pkg.Description}
}

// extractPyprojectToml parses a pyproject.toml file and extracts the package
// name from the [project] section.
func extractPyprojectToml(content string) ProjectMetadata {
	inProject := false
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "[project]" {
			inProject = true
			continue
		}
		if inProject && strings.HasPrefix(trimmed, "[") {
			break
		}
		if !inProject {
			continue
		}
		if !strings.HasPrefix(trimmed, "name") {
			continue
		}
		name := extractTomlStringValue(trimmed, "name")
		if name != "" {
			return ProjectMetadata{Name: name}
		}
	}
	return ProjectMetadata{}
}

// extractComposerJSON parses a composer.json file and extracts the name and
// description fields.
func extractComposerJSON(content string) ProjectMetadata {
	var pkg struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := json.Unmarshal([]byte(content), &pkg); err != nil {
		return ProjectMetadata{}
	}
	return ProjectMetadata{Name: pkg.Name, Description: pkg.Description}
}

// extractPubspecYaml parses a pubspec.yaml file by scanning top-level name:
// and description: lines without a YAML parser.
func extractPubspecYaml(content string) ProjectMetadata {
	var meta ProjectMetadata
	for _, line := range strings.Split(content, "\n") {
		// Only consider top-level keys (no leading whitespace).
		if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
			continue
		}
		if strings.HasPrefix(line, "name:") {
			value := strings.TrimSpace(strings.TrimPrefix(line, "name:"))
			value = strings.Trim(value, `"'`)
			meta.Name = value
		}
		if strings.HasPrefix(line, "description:") {
			value := strings.TrimSpace(strings.TrimPrefix(line, "description:"))
			value = strings.Trim(value, `"'`)
			meta.Description = value
		}
	}
	return meta
}

// extractMixExs parses a mix.exs file and extracts the app atom value from the
// project/0 function.
func extractMixExs(content string) ProjectMetadata {
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.Contains(trimmed, "app:") {
			continue
		}
		// Match patterns like: app: :my_app or app: :my_app,
		idx := strings.Index(trimmed, "app:")
		if idx < 0 {
			continue
		}
		rest := strings.TrimSpace(trimmed[idx+len("app:"):])
		if !strings.HasPrefix(rest, ":") {
			continue
		}
		rest = rest[1:] // strip leading ':'
		// Trim trailing comma and whitespace.
		rest = strings.TrimRight(rest, ", \t")
		if rest != "" {
			return ProjectMetadata{Name: rest}
		}
	}
	return ProjectMetadata{}
}

// extractPackageSwift parses a Package.swift file and extracts the name
// argument from the Package(...) initialiser.
func extractPackageSwift(content string) ProjectMetadata {
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.Contains(trimmed, "name:") {
			continue
		}
		idx := strings.Index(trimmed, "name:")
		if idx < 0 {
			continue
		}
		rest := strings.TrimSpace(trimmed[idx+len("name:"):])
		// Value should be a quoted string.
		if len(rest) == 0 || rest[0] != '"' {
			continue
		}
		rest = rest[1:] // strip opening '"'
		end := strings.IndexByte(rest, '"')
		if end < 0 {
			continue
		}
		name := rest[:end]
		if name != "" {
			return ProjectMetadata{Name: name}
		}
	}
	return ProjectMetadata{}
}

// extractTomlStringValue extracts the string value from a TOML key = "value"
// line for the given key. Returns empty string if the pattern is not matched.
func extractTomlStringValue(line, key string) string {
	// Expect: key = "value" or key="value"
	rest := strings.TrimSpace(strings.TrimPrefix(line, key))
	if !strings.HasPrefix(rest, "=") {
		return ""
	}
	rest = strings.TrimSpace(rest[1:])
	if len(rest) == 0 || rest[0] != '"' {
		return ""
	}
	rest = rest[1:] // strip opening '"'
	end := strings.IndexByte(rest, '"')
	if end < 0 {
		return ""
	}
	return rest[:end]
}
