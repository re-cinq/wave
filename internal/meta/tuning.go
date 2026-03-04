package meta

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// SizeClass categorizes repository size.
type SizeClass string

const (
	SizeSmall    SizeClass = "small"    // < 100 source files
	SizeMedium   SizeClass = "medium"   // 100-1000 source files
	SizeLarge    SizeClass = "large"    // > 1000 source files
	SizeMonorepo SizeClass = "monorepo" // Multiple independent modules
)

// CodebaseProfile represents the analysis of a repository's characteristics.
type CodebaseProfile struct {
	Language     string    `json:"language"`
	Framework    string    `json:"framework"`
	TestCommand  string    `json:"test_command"`
	BuildCommand string    `json:"build_command"`
	SourceGlob   string    `json:"source_glob"`
	Size         SizeClass `json:"size"`
	Structure    string    `json:"structure"` // "single", "monorepo", "workspace"
	PackageCount int       `json:"package_count"`
}

// langConfig holds detection metadata for a single language.
type langConfig struct {
	language     string
	testCommand  string
	buildCommand string
	sourceGlob   string
	extensions   []string
}

// AnalyzeCodebase inspects the given directory and returns a CodebaseProfile.
// It detects language, framework, test infrastructure, and project structure
// by examining marker files and directory structure.
func AnalyzeCodebase(rootDir string) CodebaseProfile {
	profile := CodebaseProfile{
		Language:     "unknown",
		Structure:    "single",
		PackageCount: 1,
	}

	lang := detectLanguage(rootDir)
	profile.Language = lang.language
	profile.TestCommand = lang.testCommand
	profile.BuildCommand = lang.buildCommand
	profile.SourceGlob = lang.sourceGlob

	// Framework detection (JS/TS only).
	if profile.Language == "javascript" {
		profile.Framework = detectFramework(rootDir)
	}

	// Structure and monorepo detection.
	structure, isMonorepo := detectStructure(rootDir, profile.Language)
	profile.Structure = structure

	// Package count.
	profile.PackageCount = countPackages(rootDir, lang)

	// Size classification.
	fileCount := countSourceFiles(rootDir, lang.extensions)
	profile.Size = classifySize(fileCount)
	if isMonorepo {
		profile.Size = SizeMonorepo
	}

	return profile
}

// detectLanguage checks marker files in priority order to determine the project language.
func detectLanguage(rootDir string) langConfig {
	checks := []struct {
		marker string
		config langConfig
	}{
		{
			marker: "go.mod",
			config: langConfig{
				language:     "go",
				testCommand:  "go test ./...",
				buildCommand: "go build ./...",
				sourceGlob:   "**/*.go",
				extensions:   []string{".go"},
			},
		},
		{
			marker: "package.json",
			config: langConfig{
				language:     "javascript",
				testCommand:  "npm test",
				buildCommand: "npm run build",
				sourceGlob:   "**/*.{js,ts}",
				extensions:   []string{".js", ".ts", ".jsx", ".tsx"},
			},
		},
		{
			marker: "Cargo.toml",
			config: langConfig{
				language:     "rust",
				testCommand:  "cargo test",
				buildCommand: "cargo build",
				sourceGlob:   "**/*.rs",
				extensions:   []string{".rs"},
			},
		},
		{
			marker: "pyproject.toml",
			config: langConfig{
				language:     "python",
				testCommand:  "pytest",
				buildCommand: "",
				sourceGlob:   "**/*.py",
				extensions:   []string{".py"},
			},
		},
		{
			marker: "setup.py",
			config: langConfig{
				language:     "python",
				testCommand:  "pytest",
				buildCommand: "",
				sourceGlob:   "**/*.py",
				extensions:   []string{".py"},
			},
		},
		{
			marker: "Gemfile",
			config: langConfig{
				language:     "ruby",
				testCommand:  "bundle exec rspec",
				buildCommand: "",
				sourceGlob:   "**/*.rb",
				extensions:   []string{".rb"},
			},
		},
		{
			marker: "pom.xml",
			config: langConfig{
				language:     "java",
				testCommand:  "mvn test",
				buildCommand: "mvn package",
				sourceGlob:   "**/*.java",
				extensions:   []string{".java"},
			},
		},
		{
			marker: "build.gradle",
			config: langConfig{
				language:     "java",
				testCommand:  "gradle test",
				buildCommand: "gradle build",
				sourceGlob:   "**/*.java",
				extensions:   []string{".java"},
			},
		},
	}

	for _, check := range checks {
		if fileExists(filepath.Join(rootDir, check.marker)) {
			return check.config
		}
	}

	return langConfig{language: "unknown"}
}

// detectFramework parses package.json dependencies to detect JS/TS frameworks.
func detectFramework(rootDir string) string {
	pkgPath := filepath.Join(rootDir, "package.json")
	data, err := os.ReadFile(pkgPath)
	if err != nil {
		return ""
	}

	var pkg struct {
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return ""
	}

	// Check both dependencies and devDependencies.
	allDeps := make(map[string]struct{})
	for dep := range pkg.Dependencies {
		allDeps[dep] = struct{}{}
	}
	for dep := range pkg.DevDependencies {
		allDeps[dep] = struct{}{}
	}

	// Check in priority order — more specific frameworks first.
	frameworkChecks := []struct {
		dep       string
		framework string
	}{
		{"next", "nextjs"},
		{"react", "react"},
		{"vue", "vue"},
		{"express", "express"},
	}

	for _, check := range frameworkChecks {
		if _, ok := allDeps[check.dep]; ok {
			return check.framework
		}
	}

	return ""
}

// detectStructure determines whether the project is single, monorepo, or workspace.
// Returns the structure string and whether it's a monorepo.
func detectStructure(rootDir string, language string) (string, bool) {
	switch language {
	case "go":
		// Multiple go.mod files at different levels → monorepo.
		goModCount := countMarkerFiles(rootDir, "go.mod")
		if goModCount > 1 {
			return "monorepo", true
		}

	case "javascript":
		// Check for workspaces field in package.json.
		pkgPath := filepath.Join(rootDir, "package.json")
		data, err := os.ReadFile(pkgPath)
		if err == nil {
			var pkg map[string]json.RawMessage
			if err := json.Unmarshal(data, &pkg); err == nil {
				if _, hasWorkspaces := pkg["workspaces"]; hasWorkspaces {
					return "workspace", true
				}
			}
		}

	case "rust":
		// Check for [workspace] in root Cargo.toml.
		cargoPath := filepath.Join(rootDir, "Cargo.toml")
		data, err := os.ReadFile(cargoPath)
		if err == nil {
			if strings.Contains(string(data), "[workspace]") {
				return "workspace", true
			}
		}
	}

	return "single", false
}

// countPackages counts the number of packages/modules in the project.
func countPackages(rootDir string, lang langConfig) int {
	switch lang.language {
	case "go":
		// Count directories containing .go files.
		return countDirsContainingExt(rootDir, ".go")

	case "javascript":
		// Count directories containing package.json.
		return countMarkerFiles(rootDir, "package.json")

	default:
		// For single-project languages, just return 1.
		// For monorepo-like structures, this would be detected elsewhere.
		return 1
	}
}

// countSourceFiles counts files with the given extensions in the directory tree.
func countSourceFiles(rootDir string, extensions []string) int {
	if len(extensions) == 0 {
		return 0
	}

	extSet := make(map[string]struct{}, len(extensions))
	for _, ext := range extensions {
		extSet[ext] = struct{}{}
	}

	count := 0
	_ = filepath.WalkDir(rootDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // Skip inaccessible paths.
		}
		if d.IsDir() {
			// Skip hidden directories and common non-source directories.
			name := d.Name()
			if strings.HasPrefix(name, ".") || name == "node_modules" || name == "vendor" || name == "target" {
				return filepath.SkipDir
			}
			return nil
		}
		ext := filepath.Ext(path)
		if _, ok := extSet[ext]; ok {
			count++
		}
		return nil
	})

	return count
}

// classifySize returns a SizeClass based on file count.
func classifySize(fileCount int) SizeClass {
	switch {
	case fileCount > 1000:
		return SizeLarge
	case fileCount >= 100:
		return SizeMedium
	default:
		return SizeSmall
	}
}

// countMarkerFiles counts occurrences of a specific filename in the directory tree.
func countMarkerFiles(rootDir string, filename string) int {
	count := 0
	_ = filepath.WalkDir(rootDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			name := d.Name()
			if strings.HasPrefix(name, ".") || name == "node_modules" || name == "vendor" || name == "target" {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Name() == filename {
			count++
		}
		return nil
	})
	return count
}

// countDirsContainingExt counts unique directories that contain at least one file with the given extension.
func countDirsContainingExt(rootDir string, ext string) int {
	seen := make(map[string]struct{})
	_ = filepath.WalkDir(rootDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			name := d.Name()
			if strings.HasPrefix(name, ".") || name == "node_modules" || name == "vendor" || name == "target" {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) == ext {
			dir := filepath.Dir(path)
			seen[dir] = struct{}{}
		}
		return nil
	})
	return len(seen)
}

// fileExists checks if a file exists at the given path.
func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}
