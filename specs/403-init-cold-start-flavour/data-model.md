# Data Model: Init Cold-Start Fix, Flavour Auto-Detection

**Date**: 2026-03-16
**Branch**: `403-init-cold-start-flavour`

## New Package: `internal/flavour`

### Core Types

```go
// DetectionResult holds the result of flavour detection for a project directory.
type DetectionResult struct {
    Flavour       string // e.g., "go", "rust", "bun", "python"
    Language      string // e.g., "go", "rust", "typescript", "python"
    TestCommand   string // e.g., "go test ./..."
    LintCommand   string // e.g., "go vet ./..."
    BuildCommand  string // e.g., "go build ./..."
    FormatCommand string // e.g., "gofmt -l ."
    SourceGlob    string // e.g., "*.go"
}

// MetadataResult holds extracted project metadata.
type MetadataResult struct {
    Name        string // Project name from manifest file
    Description string // Project description from manifest file
}

// DetectionRule maps marker files to a flavour detection result.
type DetectionRule struct {
    Markers []string         // Files to check for (ANY match triggers)
    Result  DetectionResult  // The detection result when matched
}
```

### Detection Matrix (Priority-Ordered)

```go
var rules = []DetectionRule{
    // --- Go ---
    {Markers: []string{"go.mod"}, Result: DetectionResult{
        Flavour: "go", Language: "go",
        TestCommand: "go test ./...", LintCommand: "go vet ./...",
        BuildCommand: "go build ./...", FormatCommand: "gofmt -l .",
        SourceGlob: "*.go",
    }},
    // --- Rust ---
    {Markers: []string{"Cargo.toml"}, Result: DetectionResult{
        Flavour: "rust", Language: "rust",
        TestCommand: "cargo test", LintCommand: "cargo clippy",
        BuildCommand: "cargo build", FormatCommand: "cargo fmt -- --check",
        SourceGlob: "*.rs",
    }},
    // --- Deno (before generic Node) ---
    {Markers: []string{"deno.json", "deno.jsonc"}, Result: DetectionResult{
        Flavour: "deno", Language: "typescript",
        TestCommand: "deno test", LintCommand: "deno lint",
        BuildCommand: "deno compile", FormatCommand: "deno fmt --check",
        SourceGlob: "*.{ts,tsx}",
    }},
    // --- Bun (before generic Node) ---
    {Markers: []string{"bun.lockb", "bun.lock"}, Result: DetectionResult{
        Flavour: "bun", Language: "typescript",
        TestCommand: "bun test", LintCommand: "bun run lint",
        BuildCommand: "bun run build", FormatCommand: "",
        SourceGlob: "*.{ts,tsx}",
    }},
    // --- pnpm ---
    {Markers: []string{"pnpm-lock.yaml"}, Result: DetectionResult{
        Flavour: "pnpm", Language: "javascript",
        TestCommand: "pnpm test", LintCommand: "pnpm run lint",
        BuildCommand: "pnpm run build", FormatCommand: "",
        SourceGlob: "*.{js,jsx}",
    }},
    // --- Yarn ---
    {Markers: []string{"yarn.lock"}, Result: DetectionResult{
        Flavour: "yarn", Language: "javascript",
        TestCommand: "yarn test", LintCommand: "yarn run lint",
        BuildCommand: "yarn run build", FormatCommand: "",
        SourceGlob: "*.{js,jsx}",
    }},
    // --- npm (generic Node, lowest Node priority) ---
    {Markers: []string{"package.json"}, Result: DetectionResult{
        Flavour: "node", Language: "javascript",
        TestCommand: "npm test", LintCommand: "npm run lint",
        BuildCommand: "npm run build", FormatCommand: "",
        SourceGlob: "*.{js,jsx}",
    }},
    // --- Python (modern) ---
    {Markers: []string{"pyproject.toml"}, Result: DetectionResult{
        Flavour: "python", Language: "python",
        TestCommand: "pytest", LintCommand: "ruff check .",
        BuildCommand: "", FormatCommand: "ruff format --check .",
        SourceGlob: "*.py",
    }},
    // --- Python (legacy) ---
    {Markers: []string{"setup.py"}, Result: DetectionResult{
        Flavour: "python", Language: "python",
        TestCommand: "pytest", LintCommand: "ruff check .",
        BuildCommand: "python setup.py build", FormatCommand: "ruff format --check .",
        SourceGlob: "*.py",
    }},
    // --- C# ---
    {Markers: []string{"*.csproj", "*.sln"}, Result: DetectionResult{
        Flavour: "dotnet", Language: "csharp",
        TestCommand: "dotnet test", LintCommand: "dotnet format --verify-no-changes",
        BuildCommand: "dotnet build", FormatCommand: "dotnet format --verify-no-changes",
        SourceGlob: "*.cs",
    }},
    // --- Java (Maven) ---
    {Markers: []string{"pom.xml"}, Result: DetectionResult{
        Flavour: "maven", Language: "java",
        TestCommand: "mvn test", LintCommand: "mvn checkstyle:check",
        BuildCommand: "mvn package", FormatCommand: "",
        SourceGlob: "*.java",
    }},
    // --- Java (Gradle) ---
    {Markers: []string{"build.gradle", "build.gradle.kts"}, Result: DetectionResult{
        Flavour: "gradle", Language: "java",
        TestCommand: "./gradlew test", LintCommand: "./gradlew check",
        BuildCommand: "./gradlew build", FormatCommand: "",
        SourceGlob: "*.java",
    }},
    // --- Kotlin ---
    {Markers: []string{"build.gradle.kts"}, Result: DetectionResult{
        Flavour: "kotlin", Language: "kotlin",
        TestCommand: "./gradlew test", LintCommand: "./gradlew ktlintCheck",
        BuildCommand: "./gradlew build", FormatCommand: "./gradlew ktlintFormat",
        SourceGlob: "*.kt",
    }},
    // --- Elixir ---
    {Markers: []string{"mix.exs"}, Result: DetectionResult{
        Flavour: "elixir", Language: "elixir",
        TestCommand: "mix test", LintCommand: "mix credo",
        BuildCommand: "mix compile", FormatCommand: "mix format --check-formatted",
        SourceGlob: "*.{ex,exs}",
    }},
    // --- Dart/Flutter ---
    {Markers: []string{"pubspec.yaml"}, Result: DetectionResult{
        Flavour: "dart", Language: "dart",
        TestCommand: "dart test", LintCommand: "dart analyze",
        BuildCommand: "dart compile exe", FormatCommand: "dart format --set-exit-if-changed .",
        SourceGlob: "*.dart",
    }},
    // --- C++ (CMake) ---
    {Markers: []string{"CMakeLists.txt"}, Result: DetectionResult{
        Flavour: "cmake", Language: "cpp",
        TestCommand: "ctest", LintCommand: "",
        BuildCommand: "cmake --build build", FormatCommand: "",
        SourceGlob: "*.{cpp,hpp,cc,h}",
    }},
    // --- Make ---
    {Markers: []string{"Makefile"}, Result: DetectionResult{
        Flavour: "make", Language: "",
        TestCommand: "make test", LintCommand: "make lint",
        BuildCommand: "make build", FormatCommand: "",
        SourceGlob: "",
    }},
    // --- PHP ---
    {Markers: []string{"composer.json"}, Result: DetectionResult{
        Flavour: "php", Language: "php",
        TestCommand: "vendor/bin/phpunit", LintCommand: "vendor/bin/phpstan analyse",
        BuildCommand: "", FormatCommand: "vendor/bin/php-cs-fixer fix --dry-run",
        SourceGlob: "*.php",
    }},
    // --- Ruby ---
    {Markers: []string{"Gemfile"}, Result: DetectionResult{
        Flavour: "ruby", Language: "ruby",
        TestCommand: "bundle exec rspec", LintCommand: "bundle exec rubocop",
        BuildCommand: "", FormatCommand: "bundle exec rubocop -A --fail-level error",
        SourceGlob: "*.rb",
    }},
    // --- Swift ---
    {Markers: []string{"Package.swift"}, Result: DetectionResult{
        Flavour: "swift", Language: "swift",
        TestCommand: "swift test", LintCommand: "swift package diagnose-api-breaking-changes",
        BuildCommand: "swift build", FormatCommand: "swift-format lint -r .",
        SourceGlob: "*.swift",
    }},
    // --- Zig ---
    {Markers: []string{"build.zig"}, Result: DetectionResult{
        Flavour: "zig", Language: "zig",
        TestCommand: "zig build test", LintCommand: "",
        BuildCommand: "zig build", FormatCommand: "zig fmt --check .",
        SourceGlob: "*.zig",
    }},
    // --- Scala (sbt) ---
    {Markers: []string{"build.sbt"}, Result: DetectionResult{
        Flavour: "scala", Language: "scala",
        TestCommand: "sbt test", LintCommand: "sbt scalafmtCheck",
        BuildCommand: "sbt compile", FormatCommand: "sbt scalafmt",
        SourceGlob: "*.scala",
    }},
    // --- Haskell (Cabal) ---
    {Markers: []string{"*.cabal"}, Result: DetectionResult{
        Flavour: "cabal", Language: "haskell",
        TestCommand: "cabal test", LintCommand: "hlint .",
        BuildCommand: "cabal build", FormatCommand: "",
        SourceGlob: "*.hs",
    }},
    // --- Haskell (Stack) ---
    {Markers: []string{"stack.yaml"}, Result: DetectionResult{
        Flavour: "stack", Language: "haskell",
        TestCommand: "stack test", LintCommand: "hlint .",
        BuildCommand: "stack build", FormatCommand: "",
        SourceGlob: "*.hs",
    }},
    // --- TypeScript standalone (tsconfig but no package.json) ---
    // NOTE: This is checked AFTER all Node variants. If package.json exists,
    // a Node flavour will match first. This catches standalone tsc projects.
    {Markers: []string{"tsconfig.json"}, Result: DetectionResult{
        Flavour: "typescript", Language: "typescript",
        TestCommand: "", LintCommand: "tsc --noEmit",
        BuildCommand: "tsc", FormatCommand: "",
        SourceGlob: "*.{ts,tsx}",
    }},
}
```

**Note on marker matching**: Most markers are exact filenames. Glob patterns like `*.csproj` and `*.cabal` require a directory scan check (any file matching the pattern).

### Node Refinement

For Node-based flavours (`node`, `bun`, `pnpm`, `yarn`), the `Detect` function should refine the result by:
1. Checking `tsconfig.json` to override `Language` to `"typescript"` and `SourceGlob` to `"*.{ts,tsx}"`
2. Reading `package.json` scripts to derive actual test/lint/build commands (reuse logic from existing `detectNodeProject`)

## Modified Types

### `manifest.Project` (internal/manifest/types.go)

```go
type Project struct {
    Language      string `yaml:"language,omitempty"`
    Flavour       string `yaml:"flavour,omitempty"`        // NEW
    TestCommand   string `yaml:"test_command,omitempty"`
    LintCommand   string `yaml:"lint_command,omitempty"`
    BuildCommand  string `yaml:"build_command,omitempty"`
    FormatCommand string `yaml:"format_command,omitempty"` // NEW
    SourceGlob    string `yaml:"source_glob,omitempty"`
}
```

### `onboarding.WizardResult` (internal/onboarding/onboarding.go)

```go
type WizardResult struct {
    Adapter       string
    Model         string
    Flavour       string // NEW
    TestCommand   string
    LintCommand   string
    BuildCommand  string
    FormatCommand string // NEW
    Language      string
    SourceGlob    string
    Pipelines     []string
    Skills        []string
    Dependencies  []DependencyStatus
}
```

## Function Signatures

### `internal/flavour` Package

```go
// Detect scans the given directory for marker files and returns the first
// matching flavour detection result. Returns a zero-value DetectionResult
// if no markers match.
func Detect(dir string) DetectionResult

// DetectMetadata extracts project name and description from language-specific
// manifest files in the given directory. Returns zero-value MetadataResult
// if no metadata can be extracted.
func DetectMetadata(dir string) MetadataResult
```

### Modified Functions

```go
// ProjectVars — add two new entries
func (p *Project) ProjectVars() map[string]string {
    // ... existing ...
    if p.Flavour != "" {
        vars["project.flavour"] = p.Flavour
    }
    if p.FormatCommand != "" {
        vars["project.format_command"] = p.FormatCommand
    }
    return vars
}
```

## Integration Points

| Caller | Current | After |
|--------|---------|-------|
| `init.go:detectProject()` | Inline 5-language switch | `flavour.Detect(".")` + convert to map |
| `onboarding/steps.go:detectProjectType()` | Inline 5-language switch | `flavour.Detect(".")` + convert to map |
| `init.go:createDefaultManifest()` | No flavour/format | Include `flavour` and `format_command` |
| `onboarding.go:buildManifest()` | No flavour/format | Include `flavour` and `format_command` |
| `init.go:runInit()` | No git check | Pre-wizard git bootstrap |
| `init.go:runWizardInit()` | No git check | Pre-wizard git bootstrap |
| `init.go:getFilteredAssets()` | No forge filter | Add `FilterPipelinesByForge` call |
| `init.go:printInitSuccess()` | Static next-steps | Dynamic suggestion based on project state |
| `init.go:printWizardSuccess()` | Static next-steps | Dynamic suggestion based on project state |
