package onboarding

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// ComposeService represents a service discovered in a Docker compose file.
type ComposeService struct {
	Name  string
	Build string // build context path, empty if image-only
	Image string
}

// ProjectMetadata holds the extracted name and description of a project.
type ProjectMetadata struct {
	Name        string
	Description string
	Services    []ComposeService // populated when compose file found
	SubProjects []SubProject     // nested language manifests in subdirs
}

// SubProject represents a nested project discovered in a subdirectory.
type SubProject struct {
	Path     string // relative path from root
	Name     string
	Language string // go, rust, node, python, etc.
}

// ExtractProjectMetadata attempts to extract project name and description from
// language-specific manifest files found in dir. It checks files in a defined
// priority order and returns the first successful result. Also discovers Docker
// compose services and nested sub-projects. Falls back to README heading for
// project name. Returns an empty ProjectMetadata if nothing is found.
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

	var meta ProjectMetadata
	for _, check := range checks {
		path := filepath.Join(dir, check.filename)
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		if result := check.fn(string(data)); result.Name != "" {
			meta = result
			break
		}
	}

	// Discover Docker compose services
	meta.Services = parseComposeFile(dir)

	// Discover nested sub-projects
	meta.SubProjects = scanNestedManifests(dir)

	// Fall back to README heading for project name
	if meta.Name == "" {
		meta.Name = parseREADME(dir)
	}

	// Final fallback: use directory name
	if meta.Name == "" {
		meta.Name = filepath.Base(dir)
		if meta.Name == "." || meta.Name == "/" {
			if abs, err := filepath.Abs(dir); err == nil {
				meta.Name = filepath.Base(abs)
			}
		}
	}

	return meta
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

// parseComposeFile parses a Docker compose file and returns the list of
// services it defines. It handles both docker-compose.yml and compose.yml.
// Uses lightweight YAML scanning (no full parser) to extract service names,
// build contexts, and image references.
func parseComposeFile(dir string) []ComposeService {
	var composeData []byte
	for _, name := range []string{"compose.yml", "compose.yaml", "docker-compose.yml", "docker-compose.yaml"} {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err == nil {
			composeData = data
			break
		}
	}
	if composeData == nil {
		return nil
	}

	return parseComposeServices(string(composeData))
}

// parseComposeServices extracts service definitions from compose YAML content.
// It looks for the top-level "services:" key and parses direct children as
// service names, extracting build: and image: values.
func parseComposeServices(content string) []ComposeService {
	lines := strings.Split(content, "\n")
	var services []ComposeService

	inServices := false
	var currentService *ComposeService

	for _, line := range lines {
		// Skip comments and empty lines
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		// Detect top-level "services:" key (no leading whitespace)
		if !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") {
			if strings.HasPrefix(trimmed, "services:") {
				inServices = true
				continue
			}
			// Another top-level key ends the services block
			if inServices {
				break
			}
			continue
		}

		if !inServices {
			continue
		}

		indent := len(line) - len(strings.TrimLeft(line, " \t"))

		// Service name: exactly 2 spaces (or 1 tab) of indent, ends with ':'
		if indent == 2 && strings.HasSuffix(trimmed, ":") {
			if currentService != nil {
				services = append(services, *currentService)
			}
			name := strings.TrimSuffix(trimmed, ":")
			currentService = &ComposeService{Name: name}
			continue
		}

		// Service properties: deeper indent
		if currentService != nil && indent > 2 {
			if strings.HasPrefix(trimmed, "build:") {
				val := strings.TrimSpace(strings.TrimPrefix(trimmed, "build:"))
				if val != "" {
					currentService.Build = val
				} else {
					currentService.Build = "."
				}
			}
			if strings.HasPrefix(trimmed, "image:") {
				currentService.Image = strings.TrimSpace(strings.TrimPrefix(trimmed, "image:"))
			}
		}
	}
	if currentService != nil {
		services = append(services, *currentService)
	}
	return services
}

// scanNestedManifests looks for language-specific manifest files in immediate
// subdirectories of dir. This discovers sub-projects in monorepos and Docker
// compose setups where each service has its own language manifest.
func scanNestedManifests(dir string) []SubProject {
	type manifestDef struct {
		filename string
		language string
	}
	manifests := []manifestDef{
		{"go.mod", "go"},
		{"Cargo.toml", "rust"},
		{"package.json", "node"},
		{"pyproject.toml", "python"},
		{"requirements.txt", "python"},
		{"composer.json", "php"},
		{"pubspec.yaml", "dart"},
		{"mix.exs", "elixir"},
		{"Package.swift", "swift"},
		{"Gemfile", "ruby"},
		{"build.gradle", "java"},
		{"pom.xml", "java"},
		{"CMakeLists.txt", "cpp"},
		{"Makefile", "make"},
	}

	// Directories to scan for sub-projects
	scanDirs := []string{"services", "apps", "packages", "libs", "modules", "crates", "cmd", "pkg", "src"}

	var subProjects []SubProject
	seen := make(map[string]bool)

	for _, scanDir := range scanDirs {
		base := filepath.Join(dir, scanDir)
		entries, err := os.ReadDir(base)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			subDir := filepath.Join(base, entry.Name())
			relPath := filepath.Join(scanDir, entry.Name())
			if seen[relPath] {
				continue
			}
			for _, mf := range manifests {
				if _, err := os.Stat(filepath.Join(subDir, mf.filename)); err == nil {
					meta := ExtractProjectMetadata(subDir)
					name := meta.Name
					if name == "" {
						name = entry.Name()
					}
					subProjects = append(subProjects, SubProject{
						Path:     relPath,
						Name:     name,
						Language: mf.language,
					})
					seen[relPath] = true
					break
				}
			}
		}
	}

	return subProjects
}

// parseREADME extracts the project name from a README file's first heading.
// It checks README.md, README.rst, and README.txt.
func parseREADME(dir string) string {
	for _, name := range []string{"README.md", "readme.md", "Readme.md", "README.rst", "README.txt", "README"} {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			continue
		}
		content := string(data)
		for _, line := range strings.Split(content, "\n") {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" {
				continue
			}
			// Markdown heading: # Title
			if strings.HasPrefix(trimmed, "# ") {
				title := strings.TrimSpace(strings.TrimPrefix(trimmed, "#"))
				if title != "" {
					return title
				}
			}
			// RST heading: underlined with === or ---
			// The title is the line before, but for simplicity we take the first non-empty line
			// if it's followed by a line of === or ---
			break // only check first non-empty line
		}

		// For RST: check if second non-empty line is all = or -
		nonEmpty := 0
		var firstLine string
		for _, line := range strings.Split(content, "\n") {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" {
				continue
			}
			nonEmpty++
			if nonEmpty == 1 {
				firstLine = trimmed
			}
			if nonEmpty == 2 {
				if isRSTUnderline(trimmed) && firstLine != "" {
					return firstLine
				}
				break
			}
		}
	}
	return ""
}

// isRSTUnderline returns true if the line consists entirely of = or - characters
// (reStructuredText heading underline).
func isRSTUnderline(line string) bool {
	if len(line) < 3 {
		return false
	}
	ch := line[0]
	if ch != '=' && ch != '-' {
		return false
	}
	for _, c := range line {
		if byte(c) != ch {
			return false
		}
	}
	return true
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
