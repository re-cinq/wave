package doctor

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// ProjectProfile is the structured output of a deep project scan.
type ProjectProfile struct {
	FilesScanned   int             `json:"files_scanned"`
	Languages      []LanguageInfo  `json:"languages"`
	BuildSystem    BuildSystemInfo `json:"build_system"`
	TestRunner     TestRunnerInfo  `json:"test_runner"`
	LintTools      []LintToolInfo  `json:"lint_tools"`
	CIPlatform     CIPlatformInfo  `json:"ci_platform"`
	MonorepoLayout *MonorepoInfo   `json:"monorepo,omitempty"`
	Frameworks     []FrameworkInfo `json:"frameworks"`
	Conventions    ConventionInfo  `json:"conventions"`
	HasClaudeMD    bool            `json:"has_claude_md"`
	HasDocker      bool            `json:"has_docker"`
}

// LanguageInfo describes a detected programming language.
type LanguageInfo struct {
	Name       string   `json:"name"`
	Extensions []string `json:"extensions"`
	FileCount  int      `json:"file_count"`
	Percentage float64  `json:"percentage"`
}

// BuildSystemInfo describes the detected build system.
type BuildSystemInfo struct {
	Name    string   `json:"name"`
	File    string   `json:"file"`
	Targets []string `json:"targets,omitempty"`
}

// TestRunnerInfo describes the detected test runner.
type TestRunnerInfo struct {
	Command    string `json:"command"`
	Source     string `json:"source"`
	Confidence string `json:"confidence"`
}

// LintToolInfo describes a detected linting tool.
type LintToolInfo struct {
	Name       string `json:"name"`
	ConfigFile string `json:"config_file"`
	Command    string `json:"command"`
}

// CIPlatformInfo describes the detected CI platform.
type CIPlatformInfo struct {
	Name       string   `json:"name"`
	ConfigPath string   `json:"config_path"`
	Workflows  []string `json:"workflows,omitempty"`
}

// MonorepoInfo describes a detected monorepo layout.
type MonorepoInfo struct {
	Type       string   `json:"type"`
	ConfigFile string   `json:"config_file"`
	Packages   []string `json:"packages,omitempty"`
}

// FrameworkInfo describes a detected framework.
type FrameworkInfo struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
	Source  string `json:"source"`
}

// ConventionInfo describes detected project conventions.
type ConventionInfo struct {
	CommitFormat    string `json:"commit_format,omitempty"`
	HasPRTemplate   bool   `json:"has_pr_template"`
	BranchNaming    string `json:"branch_naming,omitempty"`
	HasEditorConfig bool   `json:"has_editorconfig"`
}

// ScanOption configures scanning behavior.
type ScanOption func(*scanConfig)

type scanConfig struct {
	maxFiles int
	skipAI   bool
	runCmd   func(name string, args ...string) ([]byte, error)
}

// WithMaxFiles sets the maximum number of files to count during scanning.
func WithMaxFiles(n int) ScanOption {
	return func(c *scanConfig) {
		c.maxFiles = n
	}
}

// WithSkipAI disables AI-powered analysis, using only deterministic scanning.
func WithSkipAI() ScanOption {
	return func(c *scanConfig) {
		c.skipAI = true
	}
}

// WithRunCmd sets the command runner for external commands (e.g. git log).
func WithRunCmd(fn func(name string, args ...string) ([]byte, error)) ScanOption {
	return func(c *scanConfig) {
		c.runCmd = fn
	}
}

func defaultRunCmd(name string, args ...string) ([]byte, error) {
	return exec.Command(name, args...).Output()
}

// skipDirs contains directories to skip during file scanning.
var skipDirs = map[string]bool{
	".git":         true,
	"node_modules": true,
	"vendor":       true,
	".wave":        true,
	"__pycache__":  true,
}

// extToLanguage maps file extensions to language names.
var extToLanguage = map[string]string{
	".go":     "Go",
	".rs":     "Rust",
	".py":     "Python",
	".js":     "JavaScript",
	".jsx":    "JavaScript",
	".ts":     "TypeScript",
	".tsx":    "TypeScript",
	".java":   "Java",
	".kt":     "Kotlin",
	".rb":     "Ruby",
	".php":    "PHP",
	".c":      "C",
	".h":      "C",
	".cpp":    "C++",
	".cc":     "C++",
	".cxx":    "C++",
	".hpp":    "C++",
	".cs":     "C#",
	".swift":  "Swift",
	".m":      "Objective-C",
	".scala":  "Scala",
	".zig":    "Zig",
	".lua":    "Lua",
	".r":      "R",
	".R":      "R",
	".pl":     "Perl",
	".ex":     "Elixir",
	".exs":    "Elixir",
	".erl":    "Erlang",
	".hs":     "Haskell",
	".clj":    "Clojure",
	".dart":   "Dart",
	".vue":    "Vue",
	".svelte": "Svelte",
	".sh":     "Shell",
	".bash":   "Shell",
	".zsh":    "Shell",
}

// languageTestDefaults maps languages to default test commands.
var languageTestDefaults = map[string]string{
	"Go":         "go test ./...",
	"Rust":       "cargo test",
	"Python":     "pytest",
	"JavaScript": "npm test",
	"TypeScript": "npm test",
	"Java":       "mvn test",
	"Ruby":       "bundle exec rspec",
	"PHP":        "phpunit",
	"C#":         "dotnet test",
	"Elixir":     "mix test",
	"Dart":       "dart test",
}

// ScanProject performs a deep deterministic scan of the project directory.
func ScanProject(dir string, opts ...ScanOption) (*ProjectProfile, error) {
	cfg := &scanConfig{
		maxFiles: 10000,
		runCmd:   defaultRunCmd,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	profile := &ProjectProfile{}

	// 1. Scan languages
	profile.Languages = scanLanguages(dir, cfg.maxFiles)

	// 2. Detect build system
	profile.BuildSystem = detectBuildSystem(dir)

	// 3. Detect CI platform
	profile.CIPlatform = detectCIPlatform(dir)

	// 4. Detect test runner
	profile.TestRunner = detectTestRunner(dir, profile)

	// 5. Detect lint tools
	profile.LintTools = detectLintTools(dir)

	// 6. Detect monorepo
	profile.MonorepoLayout = detectMonorepo(dir)

	// 7. Detect frameworks
	profile.Frameworks = detectFrameworks(dir)

	// 8. Detect conventions
	profile.Conventions = detectConventions(dir, cfg)

	// 9. HasClaudeMD
	if _, err := os.Stat(filepath.Join(dir, "AGENTS.md")); err == nil {
		profile.HasClaudeMD = true
	}

	// 10. HasDocker
	profile.HasDocker = detectDocker(dir)

	// Compute total files scanned from language file counts
	for _, lang := range profile.Languages {
		profile.FilesScanned += lang.FileCount
	}

	return profile, nil
}

// scanLanguages walks the directory and counts files by language.
func scanLanguages(dir string, maxFiles int) []LanguageInfo {
	// Map: language -> set of extensions
	langExts := make(map[string]map[string]bool)
	// Map: language -> file count
	langCount := make(map[string]int)
	total := 0

	_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if skipDirs[d.Name()] {
				return fs.SkipDir
			}
			return nil
		}
		if total >= maxFiles {
			return fs.SkipAll
		}
		ext := filepath.Ext(d.Name())
		if ext == "" {
			return nil
		}
		lang, ok := extToLanguage[ext]
		if !ok {
			return nil
		}
		total++
		langCount[lang]++
		if langExts[lang] == nil {
			langExts[lang] = make(map[string]bool)
		}
		langExts[lang][ext] = true
		return nil
	})

	if total == 0 {
		return nil
	}

	var langs []LanguageInfo
	for lang, count := range langCount {
		exts := make([]string, 0, len(langExts[lang]))
		for ext := range langExts[lang] {
			exts = append(exts, ext)
		}
		sort.Strings(exts)
		pct := float64(count) / float64(total) * 100
		// Round to 1 decimal place.
		pct = float64(int(pct*10+0.5)) / 10
		langs = append(langs, LanguageInfo{
			Name:       lang,
			Extensions: exts,
			FileCount:  count,
			Percentage: pct,
		})
	}

	// Sort by file count descending, then by name ascending for stability.
	sort.Slice(langs, func(i, j int) bool {
		if langs[i].FileCount != langs[j].FileCount {
			return langs[i].FileCount > langs[j].FileCount
		}
		return langs[i].Name < langs[j].Name
	})

	return langs
}

// detectBuildSystem detects the primary build system from well-known files.
func detectBuildSystem(dir string) BuildSystemInfo {
	// Check in priority order.
	checks := []struct {
		file   string
		name   string
		parser func(path string) []string
	}{
		{"Makefile", "make", parseMakefileTargets},
		{"Taskfile.yml", "task", nil},
		{"justfile", "just", nil},
		{"package.json", "npm", parsePackageJSONScripts},
		{"build.gradle", "gradle", nil},
		{"build.gradle.kts", "gradle", nil},
		{"pom.xml", "maven", nil},
		{"CMakeLists.txt", "cmake", nil},
		{"Cargo.toml", "cargo", nil},
	}

	for _, c := range checks {
		path := filepath.Join(dir, c.file)
		if _, err := os.Stat(path); err == nil {
			info := BuildSystemInfo{
				Name: c.name,
				File: c.file,
			}
			if c.parser != nil {
				info.Targets = c.parser(path)
			}
			return info
		}
	}

	return BuildSystemInfo{Name: "none"}
}

// parseMakefileTargets extracts targets from a Makefile.
func parseMakefileTargets(path string) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	targetSet := make(map[string]bool)
	scanner := bufio.NewScanner(bytes.NewReader(data))

	// Pattern for .PHONY: declarations.
	phonyRe := regexp.MustCompile(`^\.PHONY:\s*(.+)$`)
	// Pattern for target: declarations.
	targetRe := regexp.MustCompile(`^([a-zA-Z_][a-zA-Z0-9_-]*)\s*:`)

	for scanner.Scan() {
		line := scanner.Text()

		if m := phonyRe.FindStringSubmatch(line); m != nil {
			for _, t := range strings.Fields(m[1]) {
				targetSet[t] = true
			}
			continue
		}

		if m := targetRe.FindStringSubmatch(line); m != nil {
			targetSet[m[1]] = true
		}
	}

	targets := make([]string, 0, len(targetSet))
	for t := range targetSet {
		targets = append(targets, t)
	}
	sort.Strings(targets)
	return targets
}

// parsePackageJSONScripts extracts script names from package.json.
func parsePackageJSONScripts(path string) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	var pkg struct {
		Scripts map[string]string `json:"scripts"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil
	}

	scripts := make([]string, 0, len(pkg.Scripts))
	for name := range pkg.Scripts {
		scripts = append(scripts, name)
	}
	sort.Strings(scripts)
	return scripts
}

// detectCIPlatform detects CI configuration from well-known directories.
func detectCIPlatform(dir string) CIPlatformInfo {
	// GitHub Actions
	ghDir := filepath.Join(dir, ".github", "workflows")
	if entries, err := os.ReadDir(ghDir); err == nil {
		var workflows []string
		for _, e := range entries {
			if !e.IsDir() {
				ext := filepath.Ext(e.Name())
				if ext == ".yml" || ext == ".yaml" {
					workflows = append(workflows, e.Name())
				}
			}
		}
		if len(workflows) > 0 {
			sort.Strings(workflows)
			return CIPlatformInfo{
				Name:       "github-actions",
				ConfigPath: ".github/workflows",
				Workflows:  workflows,
			}
		}
	}

	// GitLab CI
	if _, err := os.Stat(filepath.Join(dir, ".gitlab-ci.yml")); err == nil {
		return CIPlatformInfo{
			Name:       "gitlab-ci",
			ConfigPath: ".gitlab-ci.yml",
		}
	}

	// Bitbucket Pipelines
	if _, err := os.Stat(filepath.Join(dir, "bitbucket-pipelines.yml")); err == nil {
		return CIPlatformInfo{
			Name:       "bitbucket-pipelines",
			ConfigPath: "bitbucket-pipelines.yml",
		}
	}

	// CircleCI
	if _, err := os.Stat(filepath.Join(dir, ".circleci", "config.yml")); err == nil {
		return CIPlatformInfo{
			Name:       "circleci",
			ConfigPath: ".circleci/config.yml",
		}
	}

	return CIPlatformInfo{Name: "none"}
}

// detectTestRunner determines the test command from multiple sources.
func detectTestRunner(dir string, profile *ProjectProfile) TestRunnerInfo {
	// Priority 1: CI config test commands.
	if profile.CIPlatform.Name == "github-actions" {
		if cmd := extractTestCommandFromGHActions(dir); cmd != "" {
			return TestRunnerInfo{
				Command:    cmd,
				Source:     "ci",
				Confidence: "high",
			}
		}
	}

	// Priority 2: Build system test targets.
	if profile.BuildSystem.Name == "make" {
		for _, t := range profile.BuildSystem.Targets {
			if t == "test" || t == "check" || t == "spec" {
				return TestRunnerInfo{
					Command:    "make " + t,
					Source:     "makefile",
					Confidence: "high",
				}
			}
		}
	}

	if profile.BuildSystem.Name == "task" {
		return TestRunnerInfo{
			Command:    "task test",
			Source:     "taskfile",
			Confidence: "medium",
		}
	}

	// Priority 3: package.json scripts.test
	if profile.BuildSystem.Name == "npm" {
		for _, t := range profile.BuildSystem.Targets {
			if t == "test" {
				return TestRunnerInfo{
					Command:    "npm test",
					Source:     "package.json",
					Confidence: "high",
				}
			}
		}
	}

	// Priority 4: Language defaults.
	if len(profile.Languages) > 0 {
		primary := profile.Languages[0].Name
		if cmd, ok := languageTestDefaults[primary]; ok {
			return TestRunnerInfo{
				Command:    cmd,
				Source:     "heuristic",
				Confidence: "low",
			}
		}
	}

	return TestRunnerInfo{
		Command:    "",
		Source:     "none",
		Confidence: "low",
	}
}

// extractTestCommandFromGHActions scans GitHub Actions workflow files for test commands.
func extractTestCommandFromGHActions(dir string) string {
	ghDir := filepath.Join(dir, ".github", "workflows")
	entries, err := os.ReadDir(ghDir)
	if err != nil {
		return ""
	}

	// Common test command patterns, ordered longest-first to avoid
	// substring false positives (e.g. "cargo test" contains "go test").
	testPatterns := []string{
		"python -m pytest",
		"bundle exec rspec",
		"cargo test",
		"dotnet test",
		"gradle test",
		"make check",
		"make test",
		"pnpm test",
		"yarn test",
		"npm test",
		"go test",
		"mvn test",
		"pytest",
	}

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := filepath.Ext(e.Name())
		if ext != ".yml" && ext != ".yaml" {
			continue
		}

		data, err := os.ReadFile(filepath.Join(ghDir, e.Name()))
		if err != nil {
			continue
		}

		// Parse YAML to find run steps.
		var workflow map[string]interface{}
		if err := yaml.Unmarshal(data, &workflow); err != nil {
			continue
		}

		runs := extractRunSteps(workflow)
		for _, run := range runs {
			for _, pat := range testPatterns {
				if containsTestCommand(run, pat) {
					return pat
				}
			}
		}
	}

	return ""
}

// containsTestCommand checks if a run step contains the test command pattern,
// ensuring it's a whole word match (not a substring of another command).
// For example, "cargo test" should NOT match "go test".
func containsTestCommand(run, pattern string) bool {
	idx := strings.Index(run, pattern)
	if idx < 0 {
		return false
	}
	// Check that the character before the match is a word boundary.
	if idx > 0 {
		prev := run[idx-1]
		if prev != ' ' && prev != '\t' && prev != '\n' && prev != ';' && prev != '&' {
			return false
		}
	}
	return true
}

// extractRunSteps recursively finds all "run:" values in a YAML structure.
func extractRunSteps(data interface{}) []string {
	var runs []string

	switch v := data.(type) {
	case map[string]interface{}:
		for key, val := range v {
			if key == "run" {
				if s, ok := val.(string); ok {
					runs = append(runs, s)
				}
			} else {
				runs = append(runs, extractRunSteps(val)...)
			}
		}
	case []interface{}:
		for _, item := range v {
			runs = append(runs, extractRunSteps(item)...)
		}
	}

	return runs
}

// detectLintTools checks for lint tool configuration files.
func detectLintTools(dir string) []LintToolInfo {
	var tools []LintToolInfo

	// eslint
	eslintFiles := []string{".eslintrc", ".eslintrc.js", ".eslintrc.cjs", ".eslintrc.json", ".eslintrc.yml", ".eslintrc.yaml", "eslint.config.js", "eslint.config.mjs", "eslint.config.cjs"}
	for _, f := range eslintFiles {
		if _, err := os.Stat(filepath.Join(dir, f)); err == nil {
			tools = append(tools, LintToolInfo{Name: "eslint", ConfigFile: f, Command: "eslint"})
			break
		}
	}

	// golangci-lint
	golangciFiles := []string{".golangci.yml", ".golangci.yaml", ".golangci.json", ".golangci.toml"}
	for _, f := range golangciFiles {
		if _, err := os.Stat(filepath.Join(dir, f)); err == nil {
			tools = append(tools, LintToolInfo{Name: "golangci-lint", ConfigFile: f, Command: "golangci-lint run"})
			break
		}
	}

	// rustfmt
	if _, err := os.Stat(filepath.Join(dir, "rustfmt.toml")); err == nil {
		tools = append(tools, LintToolInfo{Name: "rustfmt", ConfigFile: "rustfmt.toml", Command: "cargo fmt"})
	} else if _, err := os.Stat(filepath.Join(dir, ".rustfmt.toml")); err == nil {
		tools = append(tools, LintToolInfo{Name: "rustfmt", ConfigFile: ".rustfmt.toml", Command: "cargo fmt"})
	}

	// clippy (detected via Cargo.toml presence + Rust files)
	// Not adding here as it has no dedicated config file.

	// prettier
	prettierFiles := []string{".prettierrc", ".prettierrc.js", ".prettierrc.cjs", ".prettierrc.json", ".prettierrc.yml", ".prettierrc.yaml", ".prettierrc.toml", "prettier.config.js", "prettier.config.cjs"}
	for _, f := range prettierFiles {
		if _, err := os.Stat(filepath.Join(dir, f)); err == nil {
			tools = append(tools, LintToolInfo{Name: "prettier", ConfigFile: f, Command: "prettier"})
			break
		}
	}

	// ruff
	if _, err := os.Stat(filepath.Join(dir, "ruff.toml")); err == nil {
		tools = append(tools, LintToolInfo{Name: "ruff", ConfigFile: "ruff.toml", Command: "ruff check"})
	} else if hasRuffInPyproject(filepath.Join(dir, "pyproject.toml")) {
		tools = append(tools, LintToolInfo{Name: "ruff", ConfigFile: "pyproject.toml", Command: "ruff check"})
	}

	// flake8
	if _, err := os.Stat(filepath.Join(dir, ".flake8")); err == nil {
		tools = append(tools, LintToolInfo{Name: "flake8", ConfigFile: ".flake8", Command: "flake8"})
	}

	// biome
	if _, err := os.Stat(filepath.Join(dir, "biome.json")); err == nil {
		tools = append(tools, LintToolInfo{Name: "biome", ConfigFile: "biome.json", Command: "biome check"})
	} else if _, err := os.Stat(filepath.Join(dir, "biome.jsonc")); err == nil {
		tools = append(tools, LintToolInfo{Name: "biome", ConfigFile: "biome.jsonc", Command: "biome check"})
	}

	return tools
}

// hasRuffInPyproject checks if pyproject.toml contains [tool.ruff].
func hasRuffInPyproject(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return strings.Contains(string(data), "[tool.ruff]")
}

// detectMonorepo checks for workspace/monorepo configuration.
func detectMonorepo(dir string) *MonorepoInfo {
	// nx
	if _, err := os.Stat(filepath.Join(dir, "nx.json")); err == nil {
		return &MonorepoInfo{
			Type:       "nx",
			ConfigFile: "nx.json",
			Packages:   findNXPackages(dir),
		}
	}

	// turborepo
	if _, err := os.Stat(filepath.Join(dir, "turbo.json")); err == nil {
		return &MonorepoInfo{
			Type:       "turborepo",
			ConfigFile: "turbo.json",
			Packages:   findWorkspacePackagesFromPackageJSON(dir),
		}
	}

	// pnpm workspaces
	if _, err := os.Stat(filepath.Join(dir, "pnpm-workspace.yaml")); err == nil {
		return &MonorepoInfo{
			Type:       "pnpm-workspaces",
			ConfigFile: "pnpm-workspace.yaml",
			Packages:   findPNPMPackages(dir),
		}
	}

	// lerna
	if _, err := os.Stat(filepath.Join(dir, "lerna.json")); err == nil {
		return &MonorepoInfo{
			Type:       "lerna",
			ConfigFile: "lerna.json",
		}
	}

	// go workspace
	if _, err := os.Stat(filepath.Join(dir, "go.work")); err == nil {
		return &MonorepoInfo{
			Type:       "go-workspace",
			ConfigFile: "go.work",
			Packages:   parseGoWorkModules(dir),
		}
	}

	// cargo workspace
	if hasCargoWorkspace(filepath.Join(dir, "Cargo.toml")) {
		return &MonorepoInfo{
			Type:       "cargo-workspace",
			ConfigFile: "Cargo.toml",
			Packages:   parseCargoWorkspaceMembers(filepath.Join(dir, "Cargo.toml")),
		}
	}

	// yarn workspaces (detected from package.json)
	if hasYarnWorkspaces(filepath.Join(dir, "package.json")) {
		return &MonorepoInfo{
			Type:       "yarn-workspaces",
			ConfigFile: "package.json",
			Packages:   findWorkspacePackagesFromPackageJSON(dir),
		}
	}

	// docker compose (multi-service)
	for _, name := range []string{"compose.yml", "compose.yaml", "docker-compose.yml", "docker-compose.yaml"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
			return &MonorepoInfo{
				Type:       "docker-compose",
				ConfigFile: name,
				Packages:   findComposeServices(dir, name),
			}
		}
	}

	return nil
}

// findComposeServices extracts service names with build contexts from a compose file.
func findComposeServices(dir, filename string) []string {
	data, err := os.ReadFile(filepath.Join(dir, filename))
	if err != nil {
		return nil
	}

	lines := strings.Split(string(data), "\n")
	var services []string
	inServices := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		// Top-level key
		if !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") {
			if strings.HasPrefix(trimmed, "services:") {
				inServices = true
				continue
			}
			if inServices {
				break
			}
			continue
		}

		if !inServices {
			continue
		}

		indent := len(line) - len(strings.TrimLeft(line, " \t"))
		if indent == 2 && strings.HasSuffix(trimmed, ":") {
			services = append(services, strings.TrimSuffix(trimmed, ":"))
		}
	}
	return services
}

// findNXPackages finds packages in an NX monorepo by checking common dirs.
func findNXPackages(dir string) []string {
	return findSubdirsWithFile(dir, []string{"packages", "apps", "libs"}, "package.json", 20)
}

// findPNPMPackages parses pnpm-workspace.yaml for package globs and resolves them.
func findPNPMPackages(dir string) []string {
	data, err := os.ReadFile(filepath.Join(dir, "pnpm-workspace.yaml"))
	if err != nil {
		return nil
	}

	var ws struct {
		Packages []string `yaml:"packages"`
	}
	if err := yaml.Unmarshal(data, &ws); err != nil {
		return nil
	}

	return resolveWorkspaceGlobs(dir, ws.Packages, 20)
}

// findWorkspacePackagesFromPackageJSON reads workspaces from package.json.
func findWorkspacePackagesFromPackageJSON(dir string) []string {
	data, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		return nil
	}

	var pkg struct {
		Workspaces interface{} `json:"workspaces"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil
	}

	var globs []string
	switch v := pkg.Workspaces.(type) {
	case []interface{}:
		for _, item := range v {
			if s, ok := item.(string); ok {
				globs = append(globs, s)
			}
		}
	case map[string]interface{}:
		// Yarn v2+ format: { packages: [...] }
		if pkgs, ok := v["packages"]; ok {
			if arr, ok := pkgs.([]interface{}); ok {
				for _, item := range arr {
					if s, ok := item.(string); ok {
						globs = append(globs, s)
					}
				}
			}
		}
	}

	return resolveWorkspaceGlobs(dir, globs, 20)
}

// resolveWorkspaceGlobs expands workspace glob patterns into directory names.
func resolveWorkspaceGlobs(dir string, globs []string, limit int) []string {
	var packages []string
	for _, g := range globs {
		// Normalize: strip trailing /** to /* (filepath.Glob only does single-level).
		pattern := strings.TrimSuffix(g, "/**")
		if pattern == g {
			// No change; keep the original (may already have /* or be a literal path).
		} else {
			pattern += "/*"
		}

		// Ensure the pattern has a wildcard so it enumerates children.
		// If the user wrote "packages" (no glob), append "/*" to list its children.
		if !strings.ContainsAny(pattern, "*?[") {
			pattern += "/*"
		}

		matches, err := filepath.Glob(filepath.Join(dir, pattern))
		if err != nil {
			continue
		}
		for _, m := range matches {
			info, err := os.Stat(m)
			if err != nil || !info.IsDir() {
				continue
			}
			rel, err := filepath.Rel(dir, m)
			if err != nil {
				continue
			}
			packages = append(packages, rel)
			if len(packages) >= limit {
				return packages
			}
		}
	}
	sort.Strings(packages)
	return packages
}

// findSubdirsWithFile finds subdirectories containing a specific file.
func findSubdirsWithFile(dir string, searchDirs []string, targetFile string, limit int) []string {
	var packages []string
	for _, sd := range searchDirs {
		sdPath := filepath.Join(dir, sd)
		entries, err := os.ReadDir(sdPath)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			if _, err := os.Stat(filepath.Join(sdPath, e.Name(), targetFile)); err == nil {
				packages = append(packages, filepath.Join(sd, e.Name()))
				if len(packages) >= limit {
					return packages
				}
			}
		}
	}
	return packages
}

// parseGoWorkModules reads module directories from go.work.
func parseGoWorkModules(dir string) []string {
	data, err := os.ReadFile(filepath.Join(dir, "go.work"))
	if err != nil {
		return nil
	}

	var modules []string
	scanner := bufio.NewScanner(bytes.NewReader(data))
	inUseBlock := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "use (" {
			inUseBlock = true
			continue
		}
		if inUseBlock {
			if line == ")" {
				inUseBlock = false
				continue
			}
			mod := strings.TrimSpace(line)
			if mod != "" && !strings.HasPrefix(mod, "//") {
				modules = append(modules, mod)
			}
		}
		// Single-line use directive.
		if strings.HasPrefix(line, "use ") && !strings.Contains(line, "(") {
			mod := strings.TrimSpace(strings.TrimPrefix(line, "use "))
			if mod != "" {
				modules = append(modules, mod)
			}
		}
	}
	return modules
}

// hasCargoWorkspace checks if a Cargo.toml contains [workspace].
func hasCargoWorkspace(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return strings.Contains(string(data), "[workspace]")
}

// parseCargoWorkspaceMembers extracts workspace members from Cargo.toml.
func parseCargoWorkspaceMembers(path string) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	var members []string
	scanner := bufio.NewScanner(bytes.NewReader(data))
	inMembers := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "members") && strings.Contains(line, "[") {
			inMembers = true
			// Check for inline members on the same line.
			if idx := strings.Index(line, "["); idx >= 0 {
				rest := line[idx+1:]
				if endIdx := strings.Index(rest, "]"); endIdx >= 0 {
					// Inline array.
					items := rest[:endIdx]
					for _, item := range strings.Split(items, ",") {
						item = strings.TrimSpace(item)
						item = strings.Trim(item, `"'`)
						if item != "" {
							members = append(members, item)
						}
					}
					inMembers = false
				}
			}
			continue
		}
		if inMembers {
			if strings.Contains(line, "]") {
				// Might have an item before the closing bracket.
				if idx := strings.Index(line, "]"); idx > 0 {
					item := strings.TrimSpace(line[:idx])
					item = strings.Trim(item, `"',`)
					if item != "" {
						members = append(members, item)
					}
				}
				inMembers = false
				continue
			}
			item := strings.Trim(line, `"', `)
			if item != "" && !strings.HasPrefix(item, "#") {
				members = append(members, item)
			}
		}
	}

	if len(members) > 20 {
		members = members[:20]
	}
	return members
}

// hasYarnWorkspaces checks if package.json has a workspaces field.
func hasYarnWorkspaces(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	var pkg map[string]interface{}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return false
	}
	_, ok := pkg["workspaces"]
	return ok
}

// detectFrameworks detects application frameworks from config files and dependency manifests.
func detectFrameworks(dir string) []FrameworkInfo {
	var frameworks []FrameworkInfo

	// Config-file-based detection.
	configChecks := []struct {
		patterns []string
		name     string
	}{
		{[]string{"next.config.js", "next.config.mjs", "next.config.ts"}, "Next.js"},
		{[]string{"nuxt.config.js", "nuxt.config.ts"}, "Nuxt"},
		{[]string{"angular.json"}, "Angular"},
		{[]string{"svelte.config.js", "svelte.config.ts"}, "Svelte"},
		{[]string{"vite.config.js", "vite.config.ts", "vite.config.mjs"}, "Vite"},
	}

	for _, check := range configChecks {
		for _, pat := range check.patterns {
			if _, err := os.Stat(filepath.Join(dir, pat)); err == nil {
				frameworks = append(frameworks, FrameworkInfo{
					Name:   check.name,
					Source: pat,
				})
				break
			}
		}
	}

	// Go module framework detection.
	frameworks = append(frameworks, detectGoFrameworks(dir)...)

	// Python framework detection.
	frameworks = append(frameworks, detectPythonFrameworks(dir)...)

	return frameworks
}

// detectGoFrameworks parses go.mod for known framework imports.
func detectGoFrameworks(dir string) []FrameworkInfo {
	data, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		return nil
	}

	goFrameworks := []struct {
		module string
		name   string
	}{
		{"github.com/gin-gonic/gin", "Gin"},
		{"github.com/labstack/echo", "Echo"},
		{"github.com/gofiber/fiber", "Fiber"},
		{"github.com/go-chi/chi", "Chi"},
		{"github.com/gorilla/mux", "Gorilla Mux"},
		{"github.com/spf13/cobra", "Cobra"},
		{"github.com/urfave/cli", "urfave/cli"},
	}

	content := string(data)
	var frameworks []FrameworkInfo

	for _, gf := range goFrameworks {
		if strings.Contains(content, gf.module) {
			// Try to extract version.
			version := extractGoModVersion(content, gf.module)
			frameworks = append(frameworks, FrameworkInfo{
				Name:    gf.name,
				Version: version,
				Source:  "go.mod",
			})
		}
	}

	return frameworks
}

// extractGoModVersion extracts the version of a module from go.mod content.
func extractGoModVersion(content, module string) string {
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.Contains(line, module) {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				return parts[len(parts)-1]
			}
		}
	}
	return ""
}

// detectPythonFrameworks checks requirements.txt and pyproject.toml for Python frameworks.
func detectPythonFrameworks(dir string) []FrameworkInfo {
	pyFrameworks := []struct {
		pkg  string
		name string
	}{
		{"django", "Django"},
		{"flask", "Flask"},
		{"fastapi", "FastAPI"},
		{"starlette", "Starlette"},
		{"tornado", "Tornado"},
		{"aiohttp", "aiohttp"},
	}

	var frameworks []FrameworkInfo
	found := make(map[string]bool)

	// Check requirements.txt.
	if data, err := os.ReadFile(filepath.Join(dir, "requirements.txt")); err == nil {
		content := strings.ToLower(string(data))
		for _, pf := range pyFrameworks {
			if strings.Contains(content, pf.pkg) && !found[pf.name] {
				found[pf.name] = true
				frameworks = append(frameworks, FrameworkInfo{
					Name:   pf.name,
					Source: "requirements.txt",
				})
			}
		}
	}

	// Check pyproject.toml.
	if data, err := os.ReadFile(filepath.Join(dir, "pyproject.toml")); err == nil {
		content := strings.ToLower(string(data))
		for _, pf := range pyFrameworks {
			if strings.Contains(content, pf.pkg) && !found[pf.name] {
				found[pf.name] = true
				frameworks = append(frameworks, FrameworkInfo{
					Name:   pf.name,
					Source: "pyproject.toml",
				})
			}
		}
	}

	return frameworks
}

// detectConventions checks for project conventions.
func detectConventions(dir string, cfg *scanConfig) ConventionInfo {
	conv := ConventionInfo{
		CommitFormat: "unknown",
		BranchNaming: "unknown",
	}

	// PR template.
	prTemplateFiles := []string{
		".github/pull_request_template.md",
		".github/PULL_REQUEST_TEMPLATE.md",
		"docs/pull_request_template.md",
	}
	for _, f := range prTemplateFiles {
		if _, err := os.Stat(filepath.Join(dir, f)); err == nil {
			conv.HasPRTemplate = true
			break
		}
	}

	// Also check the template directory.
	templateDir := filepath.Join(dir, ".github", "PULL_REQUEST_TEMPLATE")
	if entries, err := os.ReadDir(templateDir); err == nil && len(entries) > 0 {
		conv.HasPRTemplate = true
	}

	// EditorConfig.
	if _, err := os.Stat(filepath.Join(dir, ".editorconfig")); err == nil {
		conv.HasEditorConfig = true
	}

	// Conventional commits: analyze git log.
	conv.CommitFormat = detectCommitFormat(cfg)

	// Branch naming: analyze git branches.
	conv.BranchNaming = detectBranchNaming(cfg)

	return conv
}

// detectCommitFormat uses git log to detect if the project uses conventional commits.
func detectCommitFormat(cfg *scanConfig) string {
	out, err := cfg.runCmd("git", "log", "--oneline", "-20", "--format=%s")
	if err != nil {
		return "unknown"
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) == 0 || (len(lines) == 1 && lines[0] == "") {
		return "unknown"
	}

	conventionalRe := regexp.MustCompile(`^(feat|fix|docs|refactor|test|chore|style|perf|build|ci|revert)(\(.+\))?!?:\s`)
	conventional := 0
	for _, line := range lines {
		if conventionalRe.MatchString(line) {
			conventional++
		}
	}

	ratio := float64(conventional) / float64(len(lines))
	if ratio >= 0.5 {
		return "conventional"
	}

	return "unknown"
}

// detectBranchNaming uses git branch to detect naming conventions.
func detectBranchNaming(cfg *scanConfig) string {
	out, err := cfg.runCmd("git", "branch", "-r", "--format=%(refname:short)")
	if err != nil {
		return "unknown"
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) == 0 || (len(lines) == 1 && lines[0] == "") {
		return "unknown"
	}

	featureSlash := 0
	for _, line := range lines {
		branch := strings.TrimPrefix(line, "origin/")
		if strings.HasPrefix(branch, "feature/") || strings.HasPrefix(branch, "feat/") ||
			strings.HasPrefix(branch, "fix/") || strings.HasPrefix(branch, "bugfix/") ||
			strings.HasPrefix(branch, "hotfix/") || strings.HasPrefix(branch, "release/") ||
			strings.HasPrefix(branch, "chore/") {
			featureSlash++
		}
	}

	ratio := float64(featureSlash) / float64(len(lines))
	if ratio >= 0.3 {
		return "feature/*"
	}

	return "unknown"
}

// detectDocker checks for Docker-related files.
func detectDocker(dir string) bool {
	dockerFiles := []string{
		"Dockerfile",
		"docker-compose.yml",
		"docker-compose.yaml",
		"compose.yml",
		"compose.yaml",
	}
	for _, f := range dockerFiles {
		if _, err := os.Stat(filepath.Join(dir, f)); err == nil {
			return true
		}
	}

	// Check for .devcontainer/
	if _, err := os.Stat(filepath.Join(dir, ".devcontainer")); err == nil {
		return true
	}

	return false
}
