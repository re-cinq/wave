package flavour

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// DetectionResult holds the result of flavour detection for a project directory.
type DetectionResult struct {
	Flavour       string
	Language      string
	TestCommand   string
	LintCommand   string
	BuildCommand  string
	FormatCommand string
	SourceGlob    string
}

// DetectionRule maps marker files to a flavour detection result.
type DetectionRule struct {
	Markers []string
	Result  DetectionResult
}

// rules is the priority-ordered list of detection rules.
// More specific markers (e.g., bun.lock) come before generic ones (e.g., package.json).
var rules = []DetectionRule{
	// Go
	{Markers: []string{"go.mod"}, Result: DetectionResult{
		Flavour: "go", Language: "go",
		TestCommand: "go test ./...", LintCommand: "go vet ./...",
		BuildCommand: "go build ./...", FormatCommand: "gofmt -l .",
		SourceGlob: "*.go",
	}},
	// Rust
	{Markers: []string{"Cargo.toml"}, Result: DetectionResult{
		Flavour: "rust", Language: "rust",
		TestCommand: "cargo test", LintCommand: "cargo clippy",
		BuildCommand: "cargo build", FormatCommand: "cargo fmt -- --check",
		SourceGlob: "*.rs",
	}},
	// Deno (before generic Node)
	{Markers: []string{"deno.json", "deno.jsonc"}, Result: DetectionResult{
		Flavour: "deno", Language: "typescript",
		TestCommand: "deno test", LintCommand: "deno lint",
		BuildCommand: "deno compile", FormatCommand: "deno fmt --check",
		SourceGlob: "*.{ts,tsx}",
	}},
	// Bun (before generic Node)
	{Markers: []string{"bun.lockb", "bun.lock"}, Result: DetectionResult{
		Flavour: "bun", Language: "typescript",
		TestCommand: "bun test", LintCommand: "bun run lint",
		BuildCommand: "bun run build", FormatCommand: "",
		SourceGlob: "*.{ts,tsx}",
	}},
	// pnpm
	{Markers: []string{"pnpm-lock.yaml"}, Result: DetectionResult{
		Flavour: "pnpm", Language: "javascript",
		TestCommand: "pnpm test", LintCommand: "pnpm run lint",
		BuildCommand: "pnpm run build", FormatCommand: "",
		SourceGlob: "*.{js,jsx}",
	}},
	// Yarn
	{Markers: []string{"yarn.lock"}, Result: DetectionResult{
		Flavour: "yarn", Language: "javascript",
		TestCommand: "yarn test", LintCommand: "yarn run lint",
		BuildCommand: "yarn run build", FormatCommand: "",
		SourceGlob: "*.{js,jsx}",
	}},
	// npm (generic Node, lowest Node priority)
	{Markers: []string{"package.json"}, Result: DetectionResult{
		Flavour: "node", Language: "javascript",
		TestCommand: "npm test", LintCommand: "npm run lint",
		BuildCommand: "npm run build", FormatCommand: "",
		SourceGlob: "*.{js,jsx}",
	}},
	// Python (modern)
	{Markers: []string{"pyproject.toml"}, Result: DetectionResult{
		Flavour: "python", Language: "python",
		TestCommand: "pytest", LintCommand: "ruff check .",
		BuildCommand: "", FormatCommand: "ruff format --check .",
		SourceGlob: "*.py",
	}},
	// Python (legacy)
	{Markers: []string{"setup.py"}, Result: DetectionResult{
		Flavour: "python", Language: "python",
		TestCommand: "pytest", LintCommand: "ruff check .",
		BuildCommand: "python setup.py build", FormatCommand: "ruff format --check .",
		SourceGlob: "*.py",
	}},
	// C#
	{Markers: []string{"*.csproj", "*.sln"}, Result: DetectionResult{
		Flavour: "dotnet", Language: "csharp",
		TestCommand: "dotnet test", LintCommand: "dotnet format --verify-no-changes",
		BuildCommand: "dotnet build", FormatCommand: "dotnet format --verify-no-changes",
		SourceGlob: "*.cs",
	}},
	// Java (Maven)
	{Markers: []string{"pom.xml"}, Result: DetectionResult{
		Flavour: "maven", Language: "java",
		TestCommand: "mvn test", LintCommand: "mvn checkstyle:check",
		BuildCommand: "mvn package", FormatCommand: "",
		SourceGlob: "*.java",
	}},
	// Java (Gradle) — NOTE: build.gradle.kts also matches Kotlin below, but Gradle (Java) comes first.
	// If a project has ONLY build.gradle.kts without .kt source files, they'll get gradle/java.
	{Markers: []string{"build.gradle", "build.gradle.kts"}, Result: DetectionResult{
		Flavour: "gradle", Language: "java",
		TestCommand: "./gradlew test", LintCommand: "./gradlew check",
		BuildCommand: "./gradlew build", FormatCommand: "",
		SourceGlob: "*.java",
	}},
	// Elixir
	{Markers: []string{"mix.exs"}, Result: DetectionResult{
		Flavour: "elixir", Language: "elixir",
		TestCommand: "mix test", LintCommand: "mix credo",
		BuildCommand: "mix compile", FormatCommand: "mix format --check-formatted",
		SourceGlob: "*.{ex,exs}",
	}},
	// Dart/Flutter
	{Markers: []string{"pubspec.yaml"}, Result: DetectionResult{
		Flavour: "dart", Language: "dart",
		TestCommand: "dart test", LintCommand: "dart analyze",
		BuildCommand: "dart compile exe", FormatCommand: "dart format --set-exit-if-changed .",
		SourceGlob: "*.dart",
	}},
	// C++ (CMake)
	{Markers: []string{"CMakeLists.txt"}, Result: DetectionResult{
		Flavour: "cmake", Language: "cpp",
		TestCommand: "ctest", LintCommand: "",
		BuildCommand: "cmake --build build", FormatCommand: "",
		SourceGlob: "*.{cpp,hpp,cc,h}",
	}},
	// Make
	{Markers: []string{"Makefile"}, Result: DetectionResult{
		Flavour: "make", Language: "",
		TestCommand: "make test", LintCommand: "make lint",
		BuildCommand: "make build", FormatCommand: "",
		SourceGlob: "",
	}},
	// PHP
	{Markers: []string{"composer.json"}, Result: DetectionResult{
		Flavour: "php", Language: "php",
		TestCommand: "vendor/bin/phpunit", LintCommand: "vendor/bin/phpstan analyse",
		BuildCommand: "", FormatCommand: "vendor/bin/php-cs-fixer fix --dry-run",
		SourceGlob: "*.php",
	}},
	// Ruby
	{Markers: []string{"Gemfile"}, Result: DetectionResult{
		Flavour: "ruby", Language: "ruby",
		TestCommand: "bundle exec rspec", LintCommand: "bundle exec rubocop",
		BuildCommand: "", FormatCommand: "bundle exec rubocop -A --fail-level error",
		SourceGlob: "*.rb",
	}},
	// Swift
	{Markers: []string{"Package.swift"}, Result: DetectionResult{
		Flavour: "swift", Language: "swift",
		TestCommand: "swift test", LintCommand: "swift package diagnose-api-breaking-changes",
		BuildCommand: "swift build", FormatCommand: "swift-format lint -r .",
		SourceGlob: "*.swift",
	}},
	// Zig
	{Markers: []string{"build.zig"}, Result: DetectionResult{
		Flavour: "zig", Language: "zig",
		TestCommand: "zig build test", LintCommand: "",
		BuildCommand: "zig build", FormatCommand: "zig fmt --check .",
		SourceGlob: "*.zig",
	}},
	// Scala (sbt)
	{Markers: []string{"build.sbt"}, Result: DetectionResult{
		Flavour: "scala", Language: "scala",
		TestCommand: "sbt test", LintCommand: "sbt scalafmtCheck",
		BuildCommand: "sbt compile", FormatCommand: "sbt scalafmt",
		SourceGlob: "*.scala",
	}},
	// Haskell (Cabal)
	{Markers: []string{"*.cabal"}, Result: DetectionResult{
		Flavour: "cabal", Language: "haskell",
		TestCommand: "cabal test", LintCommand: "hlint .",
		BuildCommand: "cabal build", FormatCommand: "",
		SourceGlob: "*.hs",
	}},
	// Haskell (Stack)
	{Markers: []string{"stack.yaml"}, Result: DetectionResult{
		Flavour: "stack", Language: "haskell",
		TestCommand: "stack test", LintCommand: "hlint .",
		BuildCommand: "stack build", FormatCommand: "",
		SourceGlob: "*.hs",
	}},
	// TypeScript standalone (tsconfig but no package.json)
	{Markers: []string{"tsconfig.json"}, Result: DetectionResult{
		Flavour: "typescript", Language: "typescript",
		TestCommand: "", LintCommand: "tsc --noEmit",
		BuildCommand: "tsc", FormatCommand: "",
		SourceGlob: "*.{ts,tsx}",
	}},
}

// nodeFlavours is the set of flavours that are Node-based and support refinement.
var nodeFlavours = map[string]bool{
	"node": true,
	"bun":  true,
	"pnpm": true,
	"yarn": true,
}

// Detect scans the given directory for marker files and returns the first
// matching flavour detection result. Returns a zero-value DetectionResult
// if no markers match.
func Detect(dir string) DetectionResult {
	for _, rule := range rules {
		if matchesAny(dir, rule.Markers) {
			result := rule.Result
			// Apply Node refinement for Node-based flavours
			if nodeFlavours[result.Flavour] {
				refineNodeResult(dir, &result)
			}
			return result
		}
	}
	return DetectionResult{}
}

// matchesAny checks if any of the marker patterns exist in the directory.
// Supports both exact filenames and glob patterns (e.g., "*.csproj").
func matchesAny(dir string, markers []string) bool {
	for _, marker := range markers {
		if isGlobPattern(marker) {
			matches, err := filepath.Glob(filepath.Join(dir, marker))
			if err == nil && len(matches) > 0 {
				return true
			}
		} else {
			if _, err := os.Stat(filepath.Join(dir, marker)); err == nil {
				return true
			}
		}
	}
	return false
}

// isGlobPattern returns true if the string contains glob metacharacters.
func isGlobPattern(s string) bool {
	for _, c := range s {
		if c == '*' || c == '?' || c == '[' {
			return true
		}
	}
	return false
}

// refineNodeResult checks for tsconfig.json and reads package.json scripts
// to refine the detection result for Node-based projects.
func refineNodeResult(dir string, result *DetectionResult) {
	// Check for TypeScript
	if _, err := os.Stat(filepath.Join(dir, "tsconfig.json")); err == nil {
		result.Language = "typescript"
		result.SourceGlob = "*.{ts,tsx}"
	}

	// Read package.json scripts to derive actual commands
	data, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		return
	}

	var pkg struct {
		Scripts map[string]string `json:"scripts"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return
	}

	runner := result.Flavour
	if runner == "node" {
		runner = "npm"
	}

	runCmd := func(script string) string {
		if runner == "npm" {
			return "npm run " + script
		}
		return runner + " run " + script
	}

	if _, ok := pkg.Scripts["test"]; ok {
		if runner == "npm" {
			result.TestCommand = "npm test"
		} else {
			result.TestCommand = runner + " test"
		}
	}
	if _, ok := pkg.Scripts["lint"]; ok {
		result.LintCommand = runCmd("lint")
	}
	if _, ok := pkg.Scripts["build"]; ok {
		result.BuildCommand = runCmd("build")
	}
	if _, ok := pkg.Scripts["format"]; ok {
		result.FormatCommand = runCmd("format")
	}
}
