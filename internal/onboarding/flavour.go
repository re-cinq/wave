package onboarding

import (
	"os"
	"path/filepath"
)

// FlavourInfo holds detected project flavour metadata.
type FlavourInfo struct {
	Flavour       string
	Language      string
	TestCommand   string
	LintCommand   string
	BuildCommand  string
	FormatCommand string
	SourceGlob    string
}

// FlavourRule defines detection criteria for a project flavour.
type FlavourRule struct {
	// Markers are files/globs that must exist (glob patterns use filepath.Glob).
	Markers []string
	// Excludes are files that must NOT exist for this rule to match.
	Excludes []string
	// Info is returned when this rule matches.
	Info FlavourInfo
}

// flavourRules is the ordered detection matrix. First match wins.
// More specific rules must appear before generic ones.
var flavourRules = []FlavourRule{
	// Kotlin (gradle with .kt files — more specific than java-gradle)
	{
		Markers: []string{"build.gradle.kts", "*.kt"},
		Info: FlavourInfo{
			Flavour:       "kotlin",
			Language:      "kotlin",
			TestCommand:   "gradle test",
			LintCommand:   "gradle detekt",
			BuildCommand:  "gradle build",
			FormatCommand: "gradle ktlintCheck",
			SourceGlob:    "*.kt",
		},
	},
	// Flutter (pubspec.yaml with .metadata marker)
	{
		Markers: []string{"pubspec.yaml", ".metadata"},
		Info: FlavourInfo{
			Flavour:       "flutter",
			Language:      "dart",
			TestCommand:   "flutter test",
			LintCommand:   "dart analyze",
			BuildCommand:  "flutter build",
			FormatCommand: "dart format --set-exit-if-changed .",
			SourceGlob:    "*.dart",
		},
	},
	// Go
	{
		Markers: []string{"go.mod"},
		Info: FlavourInfo{
			Flavour:       "go",
			Language:      "go",
			TestCommand:   "go test ./...",
			LintCommand:   "go vet ./...",
			BuildCommand:  "go build ./...",
			FormatCommand: "gofmt -l .",
			SourceGlob:    "*.go",
		},
	},
	// Rust
	{
		Markers: []string{"Cargo.toml"},
		Info: FlavourInfo{
			Flavour:       "rust",
			Language:      "rust",
			TestCommand:   "cargo test",
			LintCommand:   "cargo clippy -- -D warnings",
			BuildCommand:  "cargo build",
			FormatCommand: "cargo fmt -- --check",
			SourceGlob:    "*.rs",
		},
	},
	// Bun (bun.lock)
	{
		Markers: []string{"package.json", "bun.lock"},
		Info: FlavourInfo{
			Flavour:       "bun",
			Language:      "javascript",
			TestCommand:   "bun test",
			LintCommand:   "bun lint",
			BuildCommand:  "bun run build",
			FormatCommand: "bun format",
			SourceGlob:    "*.{js,jsx,ts,tsx}",
		},
	},
	// Bun (bun.lockb)
	{
		Markers: []string{"package.json", "bun.lockb"},
		Info: FlavourInfo{
			Flavour:       "bun",
			Language:      "javascript",
			TestCommand:   "bun test",
			LintCommand:   "bun lint",
			BuildCommand:  "bun run build",
			FormatCommand: "bun format",
			SourceGlob:    "*.{js,jsx,ts,tsx}",
		},
	},
	// Node pnpm
	{
		Markers: []string{"package.json", "pnpm-lock.yaml"},
		Info: FlavourInfo{
			Flavour:       "node-pnpm",
			Language:      "javascript",
			TestCommand:   "pnpm test",
			LintCommand:   "pnpm lint",
			BuildCommand:  "pnpm build",
			FormatCommand: "pnpm format",
			SourceGlob:    "*.{js,jsx,ts,tsx}",
		},
	},
	// Node yarn
	{
		Markers: []string{"package.json", "yarn.lock"},
		Info: FlavourInfo{
			Flavour:       "node-yarn",
			Language:      "javascript",
			TestCommand:   "yarn test",
			LintCommand:   "yarn lint",
			BuildCommand:  "yarn build",
			FormatCommand: "yarn format",
			SourceGlob:    "*.{js,jsx,ts,tsx}",
		},
	},
	// Node (generic)
	{
		Markers: []string{"package.json"},
		Info: FlavourInfo{
			Flavour:       "node",
			Language:      "javascript",
			TestCommand:   "npm test",
			LintCommand:   "npm run lint",
			BuildCommand:  "npm run build",
			FormatCommand: "npm run format",
			SourceGlob:    "*.{js,jsx,ts,tsx}",
		},
	},
	// Deno (deno.json)
	{
		Markers: []string{"deno.json"},
		Info: FlavourInfo{
			Flavour:       "deno",
			Language:      "typescript",
			TestCommand:   "deno test",
			LintCommand:   "deno lint",
			BuildCommand:  "deno compile",
			FormatCommand: "deno fmt --check",
			SourceGlob:    "*.{ts,tsx}",
		},
	},
	// Deno (deno.jsonc)
	{
		Markers: []string{"deno.jsonc"},
		Info: FlavourInfo{
			Flavour:       "deno",
			Language:      "typescript",
			TestCommand:   "deno test",
			LintCommand:   "deno lint",
			BuildCommand:  "deno compile",
			FormatCommand: "deno fmt --check",
			SourceGlob:    "*.{ts,tsx}",
		},
	},
	// Python (pyproject.toml)
	{
		Markers: []string{"pyproject.toml"},
		Info: FlavourInfo{
			Flavour:       "python",
			Language:      "python",
			TestCommand:   "pytest",
			LintCommand:   "ruff check .",
			BuildCommand:  "",
			FormatCommand: "ruff format --check .",
			SourceGlob:    "*.py",
		},
	},
	// Python legacy (setup.py)
	{
		Markers: []string{"setup.py"},
		Info: FlavourInfo{
			Flavour:       "python-legacy",
			Language:      "python",
			TestCommand:   "python -m pytest",
			LintCommand:   "flake8",
			BuildCommand:  "python setup.py build",
			FormatCommand: "black --check .",
			SourceGlob:    "*.py",
		},
	},
	// Python legacy (requirements.txt, no setup.py)
	{
		Markers:  []string{"requirements.txt"},
		Excludes: []string{"setup.py"},
		Info: FlavourInfo{
			Flavour:       "python-legacy",
			Language:      "python",
			TestCommand:   "python -m pytest",
			LintCommand:   "flake8",
			BuildCommand:  "",
			FormatCommand: "black --check .",
			SourceGlob:    "*.py",
		},
	},
	// C# (.csproj glob)
	{
		Markers: []string{"*.csproj"},
		Info: FlavourInfo{
			Flavour:       "csharp",
			Language:      "csharp",
			TestCommand:   "dotnet test",
			LintCommand:   "dotnet format --verify-no-changes",
			BuildCommand:  "dotnet build",
			FormatCommand: "dotnet format",
			SourceGlob:    "*.cs",
		},
	},
	// C# (.sln glob)
	{
		Markers: []string{"*.sln"},
		Info: FlavourInfo{
			Flavour:       "csharp",
			Language:      "csharp",
			TestCommand:   "dotnet test",
			LintCommand:   "dotnet format --verify-no-changes",
			BuildCommand:  "dotnet build",
			FormatCommand: "dotnet format",
			SourceGlob:    "*.cs",
		},
	},
	// Java Maven
	{
		Markers: []string{"pom.xml"},
		Info: FlavourInfo{
			Flavour:       "java-maven",
			Language:      "java",
			TestCommand:   "mvn test",
			LintCommand:   "",
			BuildCommand:  "mvn package",
			FormatCommand: "",
			SourceGlob:    "*.java",
		},
	},
	// Java Gradle (build.gradle)
	{
		Markers: []string{"build.gradle"},
		Info: FlavourInfo{
			Flavour:       "java-gradle",
			Language:      "java",
			TestCommand:   "gradle test",
			LintCommand:   "",
			BuildCommand:  "gradle build",
			FormatCommand: "",
			SourceGlob:    "*.java",
		},
	},
	// Java Gradle (build.gradle.kts, fallback after kotlin rule above)
	{
		Markers: []string{"build.gradle.kts"},
		Info: FlavourInfo{
			Flavour:       "java-gradle",
			Language:      "java",
			TestCommand:   "gradle test",
			LintCommand:   "",
			BuildCommand:  "gradle build",
			FormatCommand: "",
			SourceGlob:    "*.java",
		},
	},
	// Elixir
	{
		Markers: []string{"mix.exs"},
		Info: FlavourInfo{
			Flavour:       "elixir",
			Language:      "elixir",
			TestCommand:   "mix test",
			LintCommand:   "mix credo",
			BuildCommand:  "mix compile",
			FormatCommand: "mix format --check-formatted",
			SourceGlob:    "*.ex",
		},
	},
	// Dart (pubspec.yaml, no .metadata — flutter already matched above)
	{
		Markers: []string{"pubspec.yaml"},
		Info: FlavourInfo{
			Flavour:       "dart",
			Language:      "dart",
			TestCommand:   "dart test",
			LintCommand:   "dart analyze",
			BuildCommand:  "dart compile exe",
			FormatCommand: "dart format --set-exit-if-changed .",
			SourceGlob:    "*.dart",
		},
	},
	// C++ CMake
	{
		Markers: []string{"CMakeLists.txt"},
		Info: FlavourInfo{
			Flavour:       "cpp-cmake",
			Language:      "cpp",
			TestCommand:   "ctest",
			LintCommand:   "",
			BuildCommand:  "cmake --build .",
			FormatCommand: "clang-format -i",
			SourceGlob:    "*.{cpp,c,h,hpp}",
		},
	},
	// PHP
	{
		Markers: []string{"composer.json"},
		Info: FlavourInfo{
			Flavour:       "php",
			Language:      "php",
			TestCommand:   "vendor/bin/phpunit",
			LintCommand:   "vendor/bin/phpstan analyse",
			BuildCommand:  "",
			FormatCommand: "vendor/bin/php-cs-fixer fix --dry-run",
			SourceGlob:    "*.php",
		},
	},
	// Ruby
	{
		Markers: []string{"Gemfile"},
		Info: FlavourInfo{
			Flavour:       "ruby",
			Language:      "ruby",
			TestCommand:   "bundle exec rspec",
			LintCommand:   "bundle exec rubocop",
			BuildCommand:  "",
			FormatCommand: "bundle exec rubocop -a",
			SourceGlob:    "*.rb",
		},
	},
	// Swift
	{
		Markers: []string{"Package.swift"},
		Info: FlavourInfo{
			Flavour:       "swift",
			Language:      "swift",
			TestCommand:   "swift test",
			LintCommand:   "swiftlint",
			BuildCommand:  "swift build",
			FormatCommand: "swift format",
			SourceGlob:    "*.swift",
		},
	},
	// Zig (build.zig)
	{
		Markers: []string{"build.zig"},
		Info: FlavourInfo{
			Flavour:       "zig",
			Language:      "zig",
			TestCommand:   "zig build test",
			LintCommand:   "",
			BuildCommand:  "zig build",
			FormatCommand: "zig fmt",
			SourceGlob:    "*.zig",
		},
	},
	// Zig (zig.zon)
	{
		Markers: []string{"zig.zon"},
		Info: FlavourInfo{
			Flavour:       "zig",
			Language:      "zig",
			TestCommand:   "zig build test",
			LintCommand:   "",
			BuildCommand:  "zig build",
			FormatCommand: "zig fmt",
			SourceGlob:    "*.zig",
		},
	},
	// Scala
	{
		Markers: []string{"build.sbt"},
		Info: FlavourInfo{
			Flavour:       "scala",
			Language:      "scala",
			TestCommand:   "sbt test",
			LintCommand:   "",
			BuildCommand:  "sbt compile",
			FormatCommand: "sbt scalafmtCheck",
			SourceGlob:    "*.scala",
		},
	},
	// Haskell (cabal glob)
	{
		Markers: []string{"*.cabal"},
		Info: FlavourInfo{
			Flavour:       "haskell",
			Language:      "haskell",
			TestCommand:   "cabal test",
			LintCommand:   "hlint .",
			BuildCommand:  "cabal build",
			FormatCommand: "ormolu --check",
			SourceGlob:    "*.hs",
		},
	},
	// Haskell (cabal.project)
	{
		Markers: []string{"cabal.project"},
		Info: FlavourInfo{
			Flavour:       "haskell",
			Language:      "haskell",
			TestCommand:   "cabal test",
			LintCommand:   "hlint .",
			BuildCommand:  "cabal build",
			FormatCommand: "ormolu --check",
			SourceGlob:    "*.hs",
		},
	},
	// Haskell Stack
	{
		Markers: []string{"stack.yaml"},
		Info: FlavourInfo{
			Flavour:       "haskell-stack",
			Language:      "haskell",
			TestCommand:   "stack test",
			LintCommand:   "hlint .",
			BuildCommand:  "stack build",
			FormatCommand: "ormolu --check",
			SourceGlob:    "*.hs",
		},
	},
	// TypeScript standalone (no package.json)
	{
		Markers:  []string{"tsconfig.json"},
		Excludes: []string{"package.json"},
		Info: FlavourInfo{
			Flavour:       "typescript-standalone",
			Language:      "typescript",
			TestCommand:   "tsc --noEmit",
			LintCommand:   "eslint .",
			BuildCommand:  "tsc",
			FormatCommand: "prettier --check .",
			SourceGlob:    "*.{ts,tsx}",
		},
	},
	// Make (fallback, excludes other common build systems)
	{
		Markers:  []string{"Makefile"},
		Excludes: []string{"go.mod", "Cargo.toml", "package.json", "CMakeLists.txt"},
		Info: FlavourInfo{
			Flavour:       "make",
			Language:      "",
			TestCommand:   "make test",
			LintCommand:   "make lint",
			BuildCommand:  "make build",
			FormatCommand: "make format",
			SourceGlob:    "",
		},
	},
}

// markerExists reports whether a marker file/glob exists in dir.
// If the marker contains '*', filepath.Glob is used; otherwise os.Stat.
func markerExists(dir, marker string) bool {
	if containsGlob(marker) {
		matches, err := filepath.Glob(filepath.Join(dir, marker))
		return err == nil && len(matches) > 0
	}
	_, err := os.Stat(filepath.Join(dir, marker))
	return err == nil
}

// containsGlob reports whether s contains a glob metacharacter.
func containsGlob(s string) bool {
	for _, c := range s {
		if c == '*' || c == '?' || c == '[' {
			return true
		}
	}
	return false
}

// DetectFlavour inspects dir and returns the first matching FlavourInfo,
// or nil if no rule matches. For node-related flavours the language field
// is upgraded to "typescript" when a tsconfig.json is present.
func DetectFlavour(dir string) *FlavourInfo {
	for _, rule := range flavourRules {
		if matchesRule(dir, rule) {
			info := rule.Info // copy

			// Upgrade node flavours to typescript when tsconfig.json exists.
			switch info.Flavour {
			case "node", "node-yarn", "node-pnpm", "bun":
				if markerExists(dir, "tsconfig.json") {
					info.Language = "typescript"
				}
			}

			return &info
		}
	}
	return nil
}

// matchesRule returns true when all markers exist and no excludes exist.
func matchesRule(dir string, rule FlavourRule) bool {
	for _, marker := range rule.Markers {
		if !markerExists(dir, marker) {
			return false
		}
	}
	for _, exclude := range rule.Excludes {
		if markerExists(dir, exclude) {
			return false
		}
	}
	return true
}
