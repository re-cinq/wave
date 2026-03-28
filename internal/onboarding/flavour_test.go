package onboarding

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createFile writes an empty file at path, creating parent directories as needed.
func createFile(t *testing.T, path string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte{}, 0o644))
}

func TestDetectFlavour(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(dir string)
		wantNil     bool
		wantFlavour string
		wantLang    string
	}{
		// ----------------------------------------------------------------
		// Empty directory
		// ----------------------------------------------------------------
		{
			name:    "empty directory returns nil",
			setup:   func(dir string) {},
			wantNil: true,
		},

		// ----------------------------------------------------------------
		// Go
		// ----------------------------------------------------------------
		{
			name: "go — go.mod present",
			setup: func(dir string) {
				createFile(t, filepath.Join(dir, "go.mod"))
			},
			wantFlavour: "go",
			wantLang:    "go",
		},

		// ----------------------------------------------------------------
		// Rust
		// ----------------------------------------------------------------
		{
			name: "rust — Cargo.toml present",
			setup: func(dir string) {
				createFile(t, filepath.Join(dir, "Cargo.toml"))
			},
			wantFlavour: "rust",
			wantLang:    "rust",
		},

		// ----------------------------------------------------------------
		// Node variants — priority ordering
		// ----------------------------------------------------------------
		{
			name: "bun — bun.lockb + package.json wins over plain node",
			setup: func(dir string) {
				createFile(t, filepath.Join(dir, "package.json"))
				createFile(t, filepath.Join(dir, "bun.lockb"))
			},
			wantFlavour: "bun",
			wantLang:    "javascript",
		},
		{
			name: "bun — bun.lock + package.json",
			setup: func(dir string) {
				createFile(t, filepath.Join(dir, "package.json"))
				createFile(t, filepath.Join(dir, "bun.lock"))
			},
			wantFlavour: "bun",
			wantLang:    "javascript",
		},
		{
			name: "node-pnpm — pnpm-lock.yaml + package.json wins over plain node",
			setup: func(dir string) {
				createFile(t, filepath.Join(dir, "package.json"))
				createFile(t, filepath.Join(dir, "pnpm-lock.yaml"))
			},
			wantFlavour: "node-pnpm",
			wantLang:    "javascript",
		},
		{
			name: "node-yarn — yarn.lock + package.json wins over plain node",
			setup: func(dir string) {
				createFile(t, filepath.Join(dir, "package.json"))
				createFile(t, filepath.Join(dir, "yarn.lock"))
			},
			wantFlavour: "node-yarn",
			wantLang:    "javascript",
		},
		{
			name: "node — plain package.json only",
			setup: func(dir string) {
				createFile(t, filepath.Join(dir, "package.json"))
			},
			wantFlavour: "node",
			wantLang:    "javascript",
		},

		// ----------------------------------------------------------------
		// Node TypeScript upgrade
		// ----------------------------------------------------------------
		{
			name: "node + tsconfig.json → language upgraded to typescript",
			setup: func(dir string) {
				createFile(t, filepath.Join(dir, "package.json"))
				createFile(t, filepath.Join(dir, "tsconfig.json"))
			},
			wantFlavour: "node",
			wantLang:    "typescript",
		},
		{
			name: "node-yarn + tsconfig.json → language upgraded to typescript",
			setup: func(dir string) {
				createFile(t, filepath.Join(dir, "package.json"))
				createFile(t, filepath.Join(dir, "yarn.lock"))
				createFile(t, filepath.Join(dir, "tsconfig.json"))
			},
			wantFlavour: "node-yarn",
			wantLang:    "typescript",
		},
		{
			name: "node-pnpm + tsconfig.json → language upgraded to typescript",
			setup: func(dir string) {
				createFile(t, filepath.Join(dir, "package.json"))
				createFile(t, filepath.Join(dir, "pnpm-lock.yaml"))
				createFile(t, filepath.Join(dir, "tsconfig.json"))
			},
			wantFlavour: "node-pnpm",
			wantLang:    "typescript",
		},
		{
			name: "bun + tsconfig.json → language upgraded to typescript",
			setup: func(dir string) {
				createFile(t, filepath.Join(dir, "package.json"))
				createFile(t, filepath.Join(dir, "bun.lockb"))
				createFile(t, filepath.Join(dir, "tsconfig.json"))
			},
			wantFlavour: "bun",
			wantLang:    "typescript",
		},

		// ----------------------------------------------------------------
		// Deno
		// ----------------------------------------------------------------
		{
			name: "deno — deno.json present",
			setup: func(dir string) {
				createFile(t, filepath.Join(dir, "deno.json"))
			},
			wantFlavour: "deno",
			wantLang:    "typescript",
		},
		{
			name: "deno — deno.jsonc present",
			setup: func(dir string) {
				createFile(t, filepath.Join(dir, "deno.jsonc"))
			},
			wantFlavour: "deno",
			wantLang:    "typescript",
		},

		// ----------------------------------------------------------------
		// Python
		// ----------------------------------------------------------------
		{
			name: "python — pyproject.toml present",
			setup: func(dir string) {
				createFile(t, filepath.Join(dir, "pyproject.toml"))
			},
			wantFlavour: "python",
			wantLang:    "python",
		},
		{
			name: "python-legacy — setup.py present",
			setup: func(dir string) {
				createFile(t, filepath.Join(dir, "setup.py"))
			},
			wantFlavour: "python-legacy",
			wantLang:    "python",
		},
		{
			name: "python-legacy — requirements.txt without setup.py",
			setup: func(dir string) {
				createFile(t, filepath.Join(dir, "requirements.txt"))
			},
			wantFlavour: "python-legacy",
			wantLang:    "python",
		},
		{
			name: "requirements.txt excluded when setup.py present — falls through to setup.py rule",
			setup: func(dir string) {
				createFile(t, filepath.Join(dir, "requirements.txt"))
				createFile(t, filepath.Join(dir, "setup.py"))
			},
			// setup.py rule fires first (it appears before requirements.txt rule in flavourRules)
			wantFlavour: "python-legacy",
			wantLang:    "python",
		},

		// ----------------------------------------------------------------
		// C#
		// ----------------------------------------------------------------
		{
			name: "csharp — *.csproj glob matches App.csproj",
			setup: func(dir string) {
				createFile(t, filepath.Join(dir, "App.csproj"))
			},
			wantFlavour: "csharp",
			wantLang:    "csharp",
		},
		{
			name: "csharp — *.sln glob matches Solution.sln",
			setup: func(dir string) {
				createFile(t, filepath.Join(dir, "Solution.sln"))
			},
			wantFlavour: "csharp",
			wantLang:    "csharp",
		},

		// ----------------------------------------------------------------
		// Java
		// ----------------------------------------------------------------
		{
			name: "java-maven — pom.xml present",
			setup: func(dir string) {
				createFile(t, filepath.Join(dir, "pom.xml"))
			},
			wantFlavour: "java-maven",
			wantLang:    "java",
		},
		{
			name: "java-gradle — build.gradle present",
			setup: func(dir string) {
				createFile(t, filepath.Join(dir, "build.gradle"))
			},
			wantFlavour: "java-gradle",
			wantLang:    "java",
		},
		{
			name: "java-gradle — build.gradle.kts without .kt files",
			setup: func(dir string) {
				createFile(t, filepath.Join(dir, "build.gradle.kts"))
				// No *.kt files — kotlin rule requires both markers
			},
			wantFlavour: "java-gradle",
			wantLang:    "java",
		},

		// ----------------------------------------------------------------
		// Kotlin — priority over java-gradle
		// ----------------------------------------------------------------
		{
			name: "kotlin — build.gradle.kts + *.kt file wins over java-gradle",
			setup: func(dir string) {
				createFile(t, filepath.Join(dir, "build.gradle.kts"))
				createFile(t, filepath.Join(dir, "Main.kt"))
			},
			wantFlavour: "kotlin",
			wantLang:    "kotlin",
		},

		// ----------------------------------------------------------------
		// Elixir
		// ----------------------------------------------------------------
		{
			name: "elixir — mix.exs present",
			setup: func(dir string) {
				createFile(t, filepath.Join(dir, "mix.exs"))
			},
			wantFlavour: "elixir",
			wantLang:    "elixir",
		},

		// ----------------------------------------------------------------
		// Dart / Flutter — priority ordering
		// ----------------------------------------------------------------
		{
			name: "flutter — pubspec.yaml + .metadata wins over dart",
			setup: func(dir string) {
				createFile(t, filepath.Join(dir, "pubspec.yaml"))
				createFile(t, filepath.Join(dir, ".metadata"))
			},
			wantFlavour: "flutter",
			wantLang:    "dart",
		},
		{
			name: "dart — pubspec.yaml without .metadata",
			setup: func(dir string) {
				createFile(t, filepath.Join(dir, "pubspec.yaml"))
			},
			wantFlavour: "dart",
			wantLang:    "dart",
		},

		// ----------------------------------------------------------------
		// C++ CMake
		// ----------------------------------------------------------------
		{
			name: "cpp-cmake — CMakeLists.txt present",
			setup: func(dir string) {
				createFile(t, filepath.Join(dir, "CMakeLists.txt"))
			},
			wantFlavour: "cpp-cmake",
			wantLang:    "cpp",
		},

		// ----------------------------------------------------------------
		// PHP
		// ----------------------------------------------------------------
		{
			name: "php — composer.json present",
			setup: func(dir string) {
				createFile(t, filepath.Join(dir, "composer.json"))
			},
			wantFlavour: "php",
			wantLang:    "php",
		},

		// ----------------------------------------------------------------
		// Ruby
		// ----------------------------------------------------------------
		{
			name: "ruby — Gemfile present",
			setup: func(dir string) {
				createFile(t, filepath.Join(dir, "Gemfile"))
			},
			wantFlavour: "ruby",
			wantLang:    "ruby",
		},

		// ----------------------------------------------------------------
		// Swift
		// ----------------------------------------------------------------
		{
			name: "swift — Package.swift present",
			setup: func(dir string) {
				createFile(t, filepath.Join(dir, "Package.swift"))
			},
			wantFlavour: "swift",
			wantLang:    "swift",
		},

		// ----------------------------------------------------------------
		// Zig
		// ----------------------------------------------------------------
		{
			name: "zig — build.zig present",
			setup: func(dir string) {
				createFile(t, filepath.Join(dir, "build.zig"))
			},
			wantFlavour: "zig",
			wantLang:    "zig",
		},
		{
			name: "zig — zig.zon present",
			setup: func(dir string) {
				createFile(t, filepath.Join(dir, "zig.zon"))
			},
			wantFlavour: "zig",
			wantLang:    "zig",
		},

		// ----------------------------------------------------------------
		// Scala
		// ----------------------------------------------------------------
		{
			name: "scala — build.sbt present",
			setup: func(dir string) {
				createFile(t, filepath.Join(dir, "build.sbt"))
			},
			wantFlavour: "scala",
			wantLang:    "scala",
		},

		// ----------------------------------------------------------------
		// Haskell
		// ----------------------------------------------------------------
		{
			name: "haskell — *.cabal glob matches MyProject.cabal",
			setup: func(dir string) {
				createFile(t, filepath.Join(dir, "MyProject.cabal"))
			},
			wantFlavour: "haskell",
			wantLang:    "haskell",
		},
		{
			name: "haskell — cabal.project present",
			setup: func(dir string) {
				createFile(t, filepath.Join(dir, "cabal.project"))
			},
			wantFlavour: "haskell",
			wantLang:    "haskell",
		},
		{
			name: "haskell-stack — stack.yaml present",
			setup: func(dir string) {
				createFile(t, filepath.Join(dir, "stack.yaml"))
			},
			wantFlavour: "haskell-stack",
			wantLang:    "haskell",
		},

		// ----------------------------------------------------------------
		// TypeScript standalone
		// ----------------------------------------------------------------
		{
			name: "typescript-standalone — tsconfig.json without package.json",
			setup: func(dir string) {
				createFile(t, filepath.Join(dir, "tsconfig.json"))
			},
			wantFlavour: "typescript-standalone",
			wantLang:    "typescript",
		},
		{
			name: "typescript-standalone not matched when package.json also present",
			setup: func(dir string) {
				createFile(t, filepath.Join(dir, "tsconfig.json"))
				createFile(t, filepath.Join(dir, "package.json"))
			},
			// package.json (node) rule fires before typescript-standalone
			wantFlavour: "node",
			wantLang:    "typescript", // upgraded because tsconfig.json present
		},

		// ----------------------------------------------------------------
		// Make (fallback)
		// ----------------------------------------------------------------
		{
			name: "make — Makefile alone",
			setup: func(dir string) {
				createFile(t, filepath.Join(dir, "Makefile"))
			},
			wantFlavour: "make",
			wantLang:    "",
		},
		{
			name: "make excluded when go.mod present",
			setup: func(dir string) {
				createFile(t, filepath.Join(dir, "Makefile"))
				createFile(t, filepath.Join(dir, "go.mod"))
			},
			// go rule matches first; Makefile exclude also prevents make rule
			wantFlavour: "go",
			wantLang:    "go",
		},
		{
			name: "make excluded when Cargo.toml present",
			setup: func(dir string) {
				createFile(t, filepath.Join(dir, "Makefile"))
				createFile(t, filepath.Join(dir, "Cargo.toml"))
			},
			wantFlavour: "rust",
			wantLang:    "rust",
		},
		{
			name: "make excluded when package.json present",
			setup: func(dir string) {
				createFile(t, filepath.Join(dir, "Makefile"))
				createFile(t, filepath.Join(dir, "package.json"))
			},
			wantFlavour: "node",
			wantLang:    "javascript",
		},
		{
			name: "make excluded when CMakeLists.txt present",
			setup: func(dir string) {
				createFile(t, filepath.Join(dir, "Makefile"))
				createFile(t, filepath.Join(dir, "CMakeLists.txt"))
			},
			wantFlavour: "cpp-cmake",
			wantLang:    "cpp",
		},

		// ----------------------------------------------------------------
		// Priority: go.mod wins when present alongside other markers
		// ----------------------------------------------------------------
		{
			name: "go.mod wins over Cargo.toml (first match in flavourRules is kotlin → flutter → go)",
			setup: func(dir string) {
				createFile(t, filepath.Join(dir, "go.mod"))
				createFile(t, filepath.Join(dir, "Cargo.toml"))
			},
			// go rule appears before rust in flavourRules
			wantFlavour: "go",
			wantLang:    "go",
		},
	}

	// Docker compose variants
	composeFiles := []string{"compose.yml", "compose.yaml", "docker-compose.yml", "docker-compose.yaml"}
	for _, cf := range composeFiles {
		name := cf
		tests = append(tests, struct {
			name        string
			setup       func(string)
			wantNil     bool
			wantFlavour string
			wantLang    string
		}{
			name: "docker-compose via " + name,
			setup: func(dir string) {
				createFile(t, filepath.Join(dir, name))
			},
			wantFlavour: "docker-compose",
			wantLang:    "",
		})
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			tc.setup(dir)

			got := DetectFlavour(dir)

			if tc.wantNil {
				assert.Nil(t, got)
				return
			}

			require.NotNil(t, got)
			assert.Equal(t, tc.wantFlavour, got.Flavour)
			assert.Equal(t, tc.wantLang, got.Language)
		})
	}
}
