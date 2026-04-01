package doctor

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeFile is a test helper that creates a file with the given content,
// creating parent directories as needed.
func writeFile(t *testing.T, dir, relPath, content string) {
	t.Helper()
	path := filepath.Join(dir, relPath)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))
}

func TestScanProject_GoProject(t *testing.T) {
	dir := t.TempDir()

	// go.mod with cobra dependency.
	writeFile(t, dir, "go.mod", `module github.com/example/myapp

go 1.22

require (
	github.com/spf13/cobra v1.8.0
	github.com/go-chi/chi/v5 v5.0.10
)
`)

	// Go source files.
	writeFile(t, dir, "main.go", `package main

func main() {}
`)
	writeFile(t, dir, "cmd/root.go", `package cmd
`)
	writeFile(t, dir, "internal/server/server.go", `package server
`)
	writeFile(t, dir, "internal/handler/handler.go", `package handler
`)

	// Makefile with targets.
	writeFile(t, dir, "Makefile", `.PHONY: build test lint clean

build:
	go build ./...

test:
	go test -race ./...

lint:
	golangci-lint run

clean:
	rm -rf bin/
`)

	// golangci-lint config.
	writeFile(t, dir, ".golangci.yml", `linters:
  enable:
    - govet
    - errcheck
`)

	// GitHub Actions CI.
	writeFile(t, dir, ".github/workflows/ci.yml", `name: CI
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - run: go test -race ./...
      - run: golangci-lint run
`)

	// CLAUDE.md.
	writeFile(t, dir, "AGENTS.md", `# Project guidelines
`)

	profile, err := ScanProject(dir, WithRunCmd(func(name string, args ...string) ([]byte, error) {
		return nil, fmt.Errorf("not available in test")
	}))
	require.NoError(t, err)

	// Languages.
	require.NotEmpty(t, profile.Languages)
	assert.Equal(t, "Go", profile.Languages[0].Name)
	assert.Equal(t, 4, profile.Languages[0].FileCount)
	assert.Equal(t, 100.0, profile.Languages[0].Percentage)
	assert.Contains(t, profile.Languages[0].Extensions, ".go")

	// Build system.
	assert.Equal(t, "make", profile.BuildSystem.Name)
	assert.Equal(t, "Makefile", profile.BuildSystem.File)
	assert.Contains(t, profile.BuildSystem.Targets, "build")
	assert.Contains(t, profile.BuildSystem.Targets, "test")
	assert.Contains(t, profile.BuildSystem.Targets, "lint")
	assert.Contains(t, profile.BuildSystem.Targets, "clean")

	// Test runner — CI takes priority.
	assert.Equal(t, "go test", profile.TestRunner.Command)
	assert.Equal(t, "ci", profile.TestRunner.Source)
	assert.Equal(t, "high", profile.TestRunner.Confidence)

	// Lint tools.
	require.Len(t, profile.LintTools, 1)
	assert.Equal(t, "golangci-lint", profile.LintTools[0].Name)
	assert.Equal(t, ".golangci.yml", profile.LintTools[0].ConfigFile)

	// CI platform.
	assert.Equal(t, "github-actions", profile.CIPlatform.Name)
	assert.Equal(t, ".github/workflows", profile.CIPlatform.ConfigPath)
	assert.Contains(t, profile.CIPlatform.Workflows, "ci.yml")

	// Frameworks.
	var cobraFound, chiFound bool
	for _, fw := range profile.Frameworks {
		if fw.Name == "Cobra" {
			cobraFound = true
			assert.Equal(t, "v1.8.0", fw.Version)
			assert.Equal(t, "go.mod", fw.Source)
		}
		if fw.Name == "Chi" {
			chiFound = true
			assert.Equal(t, "go.mod", fw.Source)
		}
	}
	assert.True(t, cobraFound, "expected Cobra framework to be detected")
	assert.True(t, chiFound, "expected Chi framework to be detected")

	// Monorepo — should be nil.
	assert.Nil(t, profile.MonorepoLayout)

	// CLAUDE.md.
	assert.True(t, profile.HasClaudeMD)

	// Docker.
	assert.False(t, profile.HasDocker)
}

func TestScanProject_NodeJSProject(t *testing.T) {
	dir := t.TempDir()

	// package.json with scripts.
	writeFile(t, dir, "package.json", `{
  "name": "my-next-app",
  "scripts": {
    "dev": "next dev",
    "build": "next build",
    "start": "next start",
    "test": "jest",
    "lint": "eslint ."
  },
  "dependencies": {
    "next": "14.0.0",
    "react": "18.2.0"
  }
}
`)

	// Next.js config.
	writeFile(t, dir, "next.config.js", `/** @type {import('next').NextConfig} */
module.exports = {}
`)

	// ESLint config.
	writeFile(t, dir, ".eslintrc.json", `{
  "extends": "next/core-web-vitals"
}
`)

	// Source files.
	writeFile(t, dir, "src/page.tsx", `export default function Home() {}`)
	writeFile(t, dir, "src/layout.tsx", `export default function Layout() {}`)
	writeFile(t, dir, "src/components/Button.tsx", `export function Button() {}`)
	writeFile(t, dir, "lib/utils.ts", `export function cn() {}`)
	writeFile(t, dir, "scripts/deploy.js", `console.log("deploy")`)

	// GitHub Actions CI.
	writeFile(t, dir, ".github/workflows/test.yml", `name: Test
on: [push]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - run: npm test
`)

	profile, err := ScanProject(dir, WithRunCmd(func(name string, args ...string) ([]byte, error) {
		return nil, fmt.Errorf("not available in test")
	}))
	require.NoError(t, err)

	// Languages — should have TypeScript and JavaScript.
	langNames := make(map[string]bool)
	for _, l := range profile.Languages {
		langNames[l.Name] = true
	}
	assert.True(t, langNames["TypeScript"], "expected TypeScript")
	assert.True(t, langNames["JavaScript"], "expected JavaScript")

	// Build system.
	assert.Equal(t, "npm", profile.BuildSystem.Name)
	assert.Equal(t, "package.json", profile.BuildSystem.File)
	assert.Contains(t, profile.BuildSystem.Targets, "test")
	assert.Contains(t, profile.BuildSystem.Targets, "build")
	assert.Contains(t, profile.BuildSystem.Targets, "lint")

	// Test runner — CI takes priority.
	assert.Equal(t, "npm test", profile.TestRunner.Command)
	assert.Equal(t, "ci", profile.TestRunner.Source)
	assert.Equal(t, "high", profile.TestRunner.Confidence)

	// Lint tools.
	require.NotEmpty(t, profile.LintTools)
	assert.Equal(t, "eslint", profile.LintTools[0].Name)

	// CI platform.
	assert.Equal(t, "github-actions", profile.CIPlatform.Name)
	assert.Contains(t, profile.CIPlatform.Workflows, "test.yml")

	// Frameworks — Next.js.
	var nextFound bool
	for _, fw := range profile.Frameworks {
		if fw.Name == "Next.js" {
			nextFound = true
			assert.Equal(t, "next.config.js", fw.Source)
		}
	}
	assert.True(t, nextFound, "expected Next.js framework to be detected")

	// HasClaudeMD.
	assert.False(t, profile.HasClaudeMD)
}

func TestScanProject_RustProject(t *testing.T) {
	dir := t.TempDir()

	// Cargo.toml with workspace.
	writeFile(t, dir, "Cargo.toml", `[workspace]
members = [
    "crate-a",
    "crate-b",
    "crate-c",
]

[workspace.dependencies]
serde = "1.0"
`)

	// Workspace member crates.
	writeFile(t, dir, "crate-a/Cargo.toml", `[package]
name = "crate-a"
`)
	writeFile(t, dir, "crate-a/src/lib.rs", `pub fn hello() {}`)

	writeFile(t, dir, "crate-b/Cargo.toml", `[package]
name = "crate-b"
`)
	writeFile(t, dir, "crate-b/src/main.rs", `fn main() {}`)

	writeFile(t, dir, "crate-c/Cargo.toml", `[package]
name = "crate-c"
`)
	writeFile(t, dir, "crate-c/src/lib.rs", `pub fn world() {}`)

	// rustfmt.toml.
	writeFile(t, dir, "rustfmt.toml", `max_width = 100
edition = "2021"
`)

	// GitHub Actions CI.
	writeFile(t, dir, ".github/workflows/ci.yml", `name: CI
on: [push]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - run: cargo test --workspace
`)

	profile, err := ScanProject(dir, WithRunCmd(func(name string, args ...string) ([]byte, error) {
		return nil, fmt.Errorf("not available in test")
	}))
	require.NoError(t, err)

	// Languages.
	require.NotEmpty(t, profile.Languages)
	assert.Equal(t, "Rust", profile.Languages[0].Name)
	assert.Contains(t, profile.Languages[0].Extensions, ".rs")

	// Build system — Cargo.
	assert.Equal(t, "cargo", profile.BuildSystem.Name)
	assert.Equal(t, "Cargo.toml", profile.BuildSystem.File)

	// Test runner — CI.
	assert.Equal(t, "cargo test", profile.TestRunner.Command)
	assert.Equal(t, "ci", profile.TestRunner.Source)

	// Lint tools — rustfmt.
	require.Len(t, profile.LintTools, 1)
	assert.Equal(t, "rustfmt", profile.LintTools[0].Name)
	assert.Equal(t, "rustfmt.toml", profile.LintTools[0].ConfigFile)

	// Monorepo — cargo workspace.
	require.NotNil(t, profile.MonorepoLayout)
	assert.Equal(t, "cargo-workspace", profile.MonorepoLayout.Type)
	assert.Equal(t, "Cargo.toml", profile.MonorepoLayout.ConfigFile)
	assert.Contains(t, profile.MonorepoLayout.Packages, "crate-a")
	assert.Contains(t, profile.MonorepoLayout.Packages, "crate-b")
	assert.Contains(t, profile.MonorepoLayout.Packages, "crate-c")
}

func TestScanProject_PythonProject(t *testing.T) {
	dir := t.TempDir()

	// pyproject.toml with ruff config.
	writeFile(t, dir, "pyproject.toml", `[build-system]
requires = ["setuptools"]

[project]
name = "myapp"
dependencies = [
    "fastapi>=0.100.0",
    "uvicorn",
]

[tool.ruff]
line-length = 88
select = ["E", "F"]
`)

	// Python source files.
	writeFile(t, dir, "app/main.py", `from fastapi import FastAPI`)
	writeFile(t, dir, "app/models.py", `class User: pass`)
	writeFile(t, dir, "tests/test_main.py", `def test_main(): pass`)

	// GitLab CI.
	writeFile(t, dir, ".gitlab-ci.yml", `stages:
  - test
test:
  script:
    - pytest
`)

	profile, err := ScanProject(dir, WithRunCmd(func(name string, args ...string) ([]byte, error) {
		return nil, fmt.Errorf("not available in test")
	}))
	require.NoError(t, err)

	// Languages.
	require.NotEmpty(t, profile.Languages)
	assert.Equal(t, "Python", profile.Languages[0].Name)
	assert.Equal(t, 3, profile.Languages[0].FileCount)
	assert.Equal(t, 100.0, profile.Languages[0].Percentage)

	// Build system — none (pyproject.toml is not a build system in our detection).
	assert.Equal(t, "none", profile.BuildSystem.Name)

	// Test runner — heuristic fallback to language default (GitLab CI parsing is not implemented for run steps).
	assert.Equal(t, "pytest", profile.TestRunner.Command)
	assert.Equal(t, "heuristic", profile.TestRunner.Source)
	assert.Equal(t, "low", profile.TestRunner.Confidence)

	// Lint tools — ruff via pyproject.toml.
	require.Len(t, profile.LintTools, 1)
	assert.Equal(t, "ruff", profile.LintTools[0].Name)
	assert.Equal(t, "pyproject.toml", profile.LintTools[0].ConfigFile)

	// CI platform — GitLab CI.
	assert.Equal(t, "gitlab-ci", profile.CIPlatform.Name)
	assert.Equal(t, ".gitlab-ci.yml", profile.CIPlatform.ConfigPath)

	// Frameworks — FastAPI from pyproject.toml.
	var fastapiFound bool
	for _, fw := range profile.Frameworks {
		if fw.Name == "FastAPI" {
			fastapiFound = true
			assert.Equal(t, "pyproject.toml", fw.Source)
		}
	}
	assert.True(t, fastapiFound, "expected FastAPI to be detected")
}

func TestScanProject_Monorepo(t *testing.T) {
	dir := t.TempDir()

	// pnpm workspace.
	writeFile(t, dir, "pnpm-workspace.yaml", `packages:
  - "packages/*"
  - "apps/*"
`)

	// turbo.json.
	writeFile(t, dir, "turbo.json", `{
  "$schema": "https://turbo.build/schema.json",
  "pipeline": {
    "build": { "dependsOn": ["^build"] },
    "test": {}
  }
}
`)

	// Root package.json.
	writeFile(t, dir, "package.json", `{
  "name": "monorepo",
  "private": true,
  "scripts": {
    "build": "turbo run build",
    "test": "turbo run test"
  }
}
`)

	// Package A.
	writeFile(t, dir, "packages/ui/package.json", `{"name": "@mono/ui"}`)
	writeFile(t, dir, "packages/ui/src/index.ts", `export {}`)

	// Package B.
	writeFile(t, dir, "packages/utils/package.json", `{"name": "@mono/utils"}`)
	writeFile(t, dir, "packages/utils/src/index.ts", `export {}`)

	// App.
	writeFile(t, dir, "apps/web/package.json", `{"name": "@mono/web"}`)
	writeFile(t, dir, "apps/web/src/index.tsx", `export {}`)

	// Prettier.
	writeFile(t, dir, ".prettierrc", `{ "semi": false }`)

	// Biome.
	writeFile(t, dir, "biome.json", `{ "organizeImports": {} }`)

	profile, err := ScanProject(dir, WithRunCmd(func(name string, args ...string) ([]byte, error) {
		return nil, fmt.Errorf("not available in test")
	}))
	require.NoError(t, err)

	// Monorepo — pnpm-workspaces takes priority since it's checked before turborepo in detection order,
	// but we check turbo.json before pnpm-workspace.yaml, so turbo wins.
	require.NotNil(t, profile.MonorepoLayout)
	assert.Equal(t, "turborepo", profile.MonorepoLayout.Type)
	assert.Equal(t, "turbo.json", profile.MonorepoLayout.ConfigFile)

	// Languages — TypeScript.
	require.NotEmpty(t, profile.Languages)
	langNames := make(map[string]bool)
	for _, l := range profile.Languages {
		langNames[l.Name] = true
	}
	assert.True(t, langNames["TypeScript"], "expected TypeScript")

	// Build system — npm (from package.json).
	assert.Equal(t, "npm", profile.BuildSystem.Name)

	// Lint tools — should find prettier and biome.
	lintNames := make(map[string]bool)
	for _, lt := range profile.LintTools {
		lintNames[lt.Name] = true
	}
	assert.True(t, lintNames["prettier"], "expected prettier")
	assert.True(t, lintNames["biome"], "expected biome")
}

func TestScanProject_EmptyDirectory(t *testing.T) {
	dir := t.TempDir()

	profile, err := ScanProject(dir, WithRunCmd(func(name string, args ...string) ([]byte, error) {
		return nil, fmt.Errorf("not available in test")
	}))
	require.NoError(t, err)

	assert.Empty(t, profile.Languages)
	assert.Equal(t, "none", profile.BuildSystem.Name)
	assert.Empty(t, profile.TestRunner.Command)
	assert.Equal(t, "none", profile.TestRunner.Source)
	assert.Empty(t, profile.LintTools)
	assert.Equal(t, "none", profile.CIPlatform.Name)
	assert.Nil(t, profile.MonorepoLayout)
	assert.Empty(t, profile.Frameworks)
	assert.False(t, profile.HasClaudeMD)
	assert.False(t, profile.HasDocker)
}

func TestScanProject_MixedLanguage(t *testing.T) {
	dir := t.TempDir()

	// Go backend.
	writeFile(t, dir, "go.mod", `module github.com/example/mixed

go 1.22
`)
	writeFile(t, dir, "main.go", `package main`)
	writeFile(t, dir, "internal/api/api.go", `package api`)
	writeFile(t, dir, "internal/db/db.go", `package db`)

	// JavaScript/Node frontend in scripts/.
	writeFile(t, dir, "scripts/build.js", `console.log("build")`)
	writeFile(t, dir, "scripts/deploy.js", `console.log("deploy")`)

	// Dockerfile.
	writeFile(t, dir, "Dockerfile", `FROM golang:1.22
COPY . .
RUN go build
`)

	profile, err := ScanProject(dir, WithRunCmd(func(name string, args ...string) ([]byte, error) {
		return nil, fmt.Errorf("not available in test")
	}))
	require.NoError(t, err)

	// Languages — Go should be primary, JavaScript secondary.
	require.Len(t, profile.Languages, 2)
	assert.Equal(t, "Go", profile.Languages[0].Name)
	assert.Equal(t, 3, profile.Languages[0].FileCount)
	assert.Equal(t, "JavaScript", profile.Languages[1].Name)
	assert.Equal(t, 2, profile.Languages[1].FileCount)

	// Percentages should add to 100%.
	totalPct := 0.0
	for _, l := range profile.Languages {
		totalPct += l.Percentage
	}
	assert.InDelta(t, 100.0, totalPct, 0.2)

	// HasDocker.
	assert.True(t, profile.HasDocker)
}

func TestScanProject_MakefileTestOverridesDefault(t *testing.T) {
	dir := t.TempDir()

	// Go project.
	writeFile(t, dir, "go.mod", `module github.com/example/proj

go 1.22
`)
	writeFile(t, dir, "main.go", `package main`)

	// Makefile with test target.
	writeFile(t, dir, "Makefile", `.PHONY: test build

build:
	go build

test:
	go test -race -coverprofile=coverage.out ./...
`)

	// No CI config — so Makefile test target should be used.
	profile, err := ScanProject(dir, WithRunCmd(func(name string, args ...string) ([]byte, error) {
		return nil, fmt.Errorf("not available in test")
	}))
	require.NoError(t, err)

	// Test runner should come from Makefile, not heuristic.
	assert.Equal(t, "make test", profile.TestRunner.Command)
	assert.Equal(t, "makefile", profile.TestRunner.Source)
	assert.Equal(t, "high", profile.TestRunner.Confidence)
}

func TestScanProject_ConventionDetection(t *testing.T) {
	dir := t.TempDir()

	// Some source file to avoid empty scan.
	writeFile(t, dir, "main.go", `package main`)

	// PR template.
	writeFile(t, dir, ".github/pull_request_template.md", `## Summary

## Test Plan
`)

	// EditorConfig.
	writeFile(t, dir, ".editorconfig", `root = true

[*]
indent_style = tab
`)

	// Mock git log output with conventional commits.
	gitLogOutput := `feat: add user authentication
fix(api): resolve timeout issue
docs: update README
refactor: extract validation logic
test: add integration tests
chore: update dependencies
feat(ui): add dark mode toggle
fix: handle nil pointer in parser
feat: implement caching layer
test: improve coverage for auth module
`

	// Mock git branch output.
	gitBranchOutput := `origin/main
origin/feature/auth
origin/feature/dark-mode
origin/fix/timeout-bug
origin/release/v1.0
origin/chore/deps
`

	profile, err := ScanProject(dir, WithRunCmd(func(name string, args ...string) ([]byte, error) {
		if name == "git" && len(args) > 0 {
			if args[0] == "log" {
				return []byte(gitLogOutput), nil
			}
			if args[0] == "branch" {
				return []byte(gitBranchOutput), nil
			}
		}
		return nil, fmt.Errorf("unknown command: %s %v", name, args)
	}))
	require.NoError(t, err)

	// Commit format.
	assert.Equal(t, "conventional", profile.Conventions.CommitFormat)

	// PR template.
	assert.True(t, profile.Conventions.HasPRTemplate)

	// EditorConfig.
	assert.True(t, profile.Conventions.HasEditorConfig)

	// Branch naming — should detect feature/* pattern.
	assert.Equal(t, "feature/*", profile.Conventions.BranchNaming)
}

func TestScanProject_SkipDirs(t *testing.T) {
	dir := t.TempDir()

	// Files in skipped directories should not be counted.
	writeFile(t, dir, "main.go", `package main`)
	writeFile(t, dir, "node_modules/lodash/index.js", `module.exports = {}`)
	writeFile(t, dir, "node_modules/react/index.js", `module.exports = {}`)
	writeFile(t, dir, "vendor/github.com/pkg/errors/errors.go", `package errors`)
	writeFile(t, dir, "__pycache__/cache.py", `pass`)
	writeFile(t, dir, ".git/config", `[core]`)

	profile, err := ScanProject(dir, WithRunCmd(func(name string, args ...string) ([]byte, error) {
		return nil, fmt.Errorf("not available in test")
	}))
	require.NoError(t, err)

	// Only main.go should be counted.
	require.Len(t, profile.Languages, 1)
	assert.Equal(t, "Go", profile.Languages[0].Name)
	assert.Equal(t, 1, profile.Languages[0].FileCount)
}

func TestScanProject_MaxFiles(t *testing.T) {
	dir := t.TempDir()

	// Create more files than the limit.
	for i := 0; i < 20; i++ {
		writeFile(t, dir, fmt.Sprintf("file%d.go", i), `package main`)
	}

	profile, err := ScanProject(dir,
		WithMaxFiles(5),
		WithRunCmd(func(name string, args ...string) ([]byte, error) {
			return nil, fmt.Errorf("not available in test")
		}),
	)
	require.NoError(t, err)

	// Should stop after maxFiles.
	require.NotEmpty(t, profile.Languages)
	assert.Equal(t, "Go", profile.Languages[0].Name)
	assert.LessOrEqual(t, profile.Languages[0].FileCount, 5)
}

func TestScanProject_Docker(t *testing.T) {
	tests := []struct {
		name  string
		files map[string]string
		want  bool
	}{
		{
			name:  "Dockerfile",
			files: map[string]string{"Dockerfile": "FROM alpine"},
			want:  true,
		},
		{
			name:  "docker-compose.yml",
			files: map[string]string{"docker-compose.yml": "version: '3'"},
			want:  true,
		},
		{
			name:  "compose.yml",
			files: map[string]string{"compose.yml": "version: '3'"},
			want:  true,
		},
		{
			name:  "devcontainer",
			files: map[string]string{".devcontainer/devcontainer.json": "{}"},
			want:  true,
		},
		{
			name:  "none",
			files: map[string]string{"main.go": "package main"},
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			for f, content := range tt.files {
				writeFile(t, dir, f, content)
			}

			profile, err := ScanProject(dir, WithRunCmd(func(name string, args ...string) ([]byte, error) {
				return nil, fmt.Errorf("not available in test")
			}))
			require.NoError(t, err)
			assert.Equal(t, tt.want, profile.HasDocker)
		})
	}
}

func TestScanProject_CIPlatforms(t *testing.T) {
	tests := []struct {
		name       string
		files      map[string]string
		wantName   string
		wantConfig string
	}{
		{
			name: "github-actions",
			files: map[string]string{
				".github/workflows/ci.yml": "name: CI",
			},
			wantName:   "github-actions",
			wantConfig: ".github/workflows",
		},
		{
			name: "gitlab-ci",
			files: map[string]string{
				".gitlab-ci.yml": "stages: [test]",
			},
			wantName:   "gitlab-ci",
			wantConfig: ".gitlab-ci.yml",
		},
		{
			name: "bitbucket",
			files: map[string]string{
				"bitbucket-pipelines.yml": "pipelines:",
			},
			wantName:   "bitbucket-pipelines",
			wantConfig: "bitbucket-pipelines.yml",
		},
		{
			name: "circleci",
			files: map[string]string{
				".circleci/config.yml": "version: 2.1",
			},
			wantName:   "circleci",
			wantConfig: ".circleci/config.yml",
		},
		{
			name:       "none",
			files:      map[string]string{},
			wantName:   "none",
			wantConfig: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			for f, content := range tt.files {
				writeFile(t, dir, f, content)
			}

			profile, err := ScanProject(dir, WithRunCmd(func(name string, args ...string) ([]byte, error) {
				return nil, fmt.Errorf("not available in test")
			}))
			require.NoError(t, err)
			assert.Equal(t, tt.wantName, profile.CIPlatform.Name)
			assert.Equal(t, tt.wantConfig, profile.CIPlatform.ConfigPath)
		})
	}
}

func TestScanProject_GoWorkspace(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, dir, "go.work", `go 1.22

use (
	./svc-a
	./svc-b
	./lib/shared
)
`)
	writeFile(t, dir, "svc-a/main.go", `package main`)
	writeFile(t, dir, "svc-b/main.go", `package main`)
	writeFile(t, dir, "lib/shared/shared.go", `package shared`)

	profile, err := ScanProject(dir, WithRunCmd(func(name string, args ...string) ([]byte, error) {
		return nil, fmt.Errorf("not available in test")
	}))
	require.NoError(t, err)

	require.NotNil(t, profile.MonorepoLayout)
	assert.Equal(t, "go-workspace", profile.MonorepoLayout.Type)
	assert.Equal(t, "go.work", profile.MonorepoLayout.ConfigFile)
	assert.Contains(t, profile.MonorepoLayout.Packages, "./svc-a")
	assert.Contains(t, profile.MonorepoLayout.Packages, "./svc-b")
	assert.Contains(t, profile.MonorepoLayout.Packages, "./lib/shared")
}

func TestScanProject_YarnWorkspaces(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, dir, "package.json", `{
  "name": "mono",
  "private": true,
  "workspaces": ["packages/*"],
  "scripts": {
    "test": "jest"
  }
}
`)

	writeFile(t, dir, "packages/core/package.json", `{"name": "@mono/core"}`)
	writeFile(t, dir, "packages/core/index.js", `module.exports = {}`)

	writeFile(t, dir, "packages/cli/package.json", `{"name": "@mono/cli"}`)
	writeFile(t, dir, "packages/cli/index.js", `module.exports = {}`)

	profile, err := ScanProject(dir, WithRunCmd(func(name string, args ...string) ([]byte, error) {
		return nil, fmt.Errorf("not available in test")
	}))
	require.NoError(t, err)

	require.NotNil(t, profile.MonorepoLayout)
	assert.Equal(t, "yarn-workspaces", profile.MonorepoLayout.Type)
	assert.Equal(t, "package.json", profile.MonorepoLayout.ConfigFile)
	assert.Contains(t, profile.MonorepoLayout.Packages, filepath.Join("packages", "cli"))
	assert.Contains(t, profile.MonorepoLayout.Packages, filepath.Join("packages", "core"))
}

func TestScanProject_PNPMWorkspaces(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, dir, "pnpm-workspace.yaml", `packages:
  - "packages/*"
`)

	writeFile(t, dir, "package.json", `{"name": "root", "private": true}`)

	writeFile(t, dir, "packages/alpha/package.json", `{"name": "alpha"}`)
	writeFile(t, dir, "packages/beta/package.json", `{"name": "beta"}`)

	profile, err := ScanProject(dir, WithRunCmd(func(name string, args ...string) ([]byte, error) {
		return nil, fmt.Errorf("not available in test")
	}))
	require.NoError(t, err)

	require.NotNil(t, profile.MonorepoLayout)
	assert.Equal(t, "pnpm-workspaces", profile.MonorepoLayout.Type)
	assert.Equal(t, "pnpm-workspace.yaml", profile.MonorepoLayout.ConfigFile)
	assert.Contains(t, profile.MonorepoLayout.Packages, filepath.Join("packages", "alpha"))
	assert.Contains(t, profile.MonorepoLayout.Packages, filepath.Join("packages", "beta"))
}

func TestScanProject_DockerCompose(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, dir, "compose.yml", `services:
  api:
    build: ./api
  web:
    build: ./web
  db:
    image: postgres:15
`)
	writeFile(t, dir, "api/main.go", `package main`)
	writeFile(t, dir, "web/index.js", `console.log("hello")`)

	profile, err := ScanProject(dir, WithRunCmd(func(name string, args ...string) ([]byte, error) {
		return nil, fmt.Errorf("not available in test")
	}))
	require.NoError(t, err)

	require.NotNil(t, profile.MonorepoLayout)
	assert.Equal(t, "docker-compose", profile.MonorepoLayout.Type)
	assert.Equal(t, "compose.yml", profile.MonorepoLayout.ConfigFile)
	assert.Contains(t, profile.MonorepoLayout.Packages, "api")
	assert.Contains(t, profile.MonorepoLayout.Packages, "web")
	assert.Contains(t, profile.MonorepoLayout.Packages, "db")
}

func TestScanProject_LintToolCombinations(t *testing.T) {
	dir := t.TempDir()

	// Create multiple lint configs.
	writeFile(t, dir, ".eslintrc.json", `{}`)
	writeFile(t, dir, ".prettierrc.json", `{}`)
	writeFile(t, dir, ".flake8", `[flake8]
max-line-length = 88
`)
	writeFile(t, dir, "ruff.toml", `line-length = 88`)

	writeFile(t, dir, "main.py", `pass`)

	profile, err := ScanProject(dir, WithRunCmd(func(name string, args ...string) ([]byte, error) {
		return nil, fmt.Errorf("not available in test")
	}))
	require.NoError(t, err)

	lintNames := make(map[string]bool)
	for _, lt := range profile.LintTools {
		lintNames[lt.Name] = true
	}
	assert.True(t, lintNames["eslint"])
	assert.True(t, lintNames["prettier"])
	assert.True(t, lintNames["flake8"])
	assert.True(t, lintNames["ruff"])
	assert.Len(t, profile.LintTools, 4)
}

func TestScanProject_FrameworkDetection(t *testing.T) {
	tests := []struct {
		name      string
		files     map[string]string
		wantFrame string
		wantSrc   string
	}{
		{
			name:      "vite",
			files:     map[string]string{"vite.config.ts": "export default {}"},
			wantFrame: "Vite",
			wantSrc:   "vite.config.ts",
		},
		{
			name:      "angular",
			files:     map[string]string{"angular.json": "{}"},
			wantFrame: "Angular",
			wantSrc:   "angular.json",
		},
		{
			name:      "svelte",
			files:     map[string]string{"svelte.config.js": "export default {}"},
			wantFrame: "Svelte",
			wantSrc:   "svelte.config.js",
		},
		{
			name:      "nuxt",
			files:     map[string]string{"nuxt.config.ts": "export default {}"},
			wantFrame: "Nuxt",
			wantSrc:   "nuxt.config.ts",
		},
		{
			name: "flask",
			files: map[string]string{
				"requirements.txt": "flask>=3.0\ngunicorn",
			},
			wantFrame: "Flask",
			wantSrc:   "requirements.txt",
		},
		{
			name: "django",
			files: map[string]string{
				"requirements.txt": "django>=4.2\ndjango-rest-framework",
			},
			wantFrame: "Django",
			wantSrc:   "requirements.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			for f, content := range tt.files {
				writeFile(t, dir, f, content)
			}

			profile, err := ScanProject(dir, WithRunCmd(func(name string, args ...string) ([]byte, error) {
				return nil, fmt.Errorf("not available in test")
			}))
			require.NoError(t, err)

			var found bool
			for _, fw := range profile.Frameworks {
				if fw.Name == tt.wantFrame {
					found = true
					assert.Equal(t, tt.wantSrc, fw.Source)
				}
			}
			assert.True(t, found, "expected framework %q to be detected", tt.wantFrame)
		})
	}
}

func TestScanProject_ConventionUnknownWithoutGit(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "main.go", `package main`)

	profile, err := ScanProject(dir, WithRunCmd(func(name string, args ...string) ([]byte, error) {
		return nil, fmt.Errorf("not a git repo")
	}))
	require.NoError(t, err)

	assert.Equal(t, "unknown", profile.Conventions.CommitFormat)
	assert.Equal(t, "unknown", profile.Conventions.BranchNaming)
	assert.False(t, profile.Conventions.HasPRTemplate)
	assert.False(t, profile.Conventions.HasEditorConfig)
}

func TestParseMakefileTargets(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "Makefile")
	os.WriteFile(path, []byte(`.PHONY: build test lint

build:
	go build ./...

test:
	go test ./...

lint:
	golangci-lint run

internal-target:
	echo "internal"
`), 0644)

	targets := parseMakefileTargets(path)
	assert.Contains(t, targets, "build")
	assert.Contains(t, targets, "test")
	assert.Contains(t, targets, "lint")
	assert.Contains(t, targets, "internal-target")
}

func TestParsePackageJSONScripts(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "package.json")
	os.WriteFile(path, []byte(`{
  "scripts": {
    "build": "tsc",
    "test": "jest",
    "lint": "eslint .",
    "dev": "next dev"
  }
}
`), 0644)

	scripts := parsePackageJSONScripts(path)
	assert.Contains(t, scripts, "build")
	assert.Contains(t, scripts, "test")
	assert.Contains(t, scripts, "lint")
	assert.Contains(t, scripts, "dev")
	assert.Len(t, scripts, 4)
}

func TestHasRuffInPyproject(t *testing.T) {
	dir := t.TempDir()

	// With ruff config.
	path := filepath.Join(dir, "pyproject.toml")
	os.WriteFile(path, []byte(`[tool.ruff]
line-length = 88
`), 0644)
	assert.True(t, hasRuffInPyproject(path))

	// Without ruff config.
	os.WriteFile(path, []byte(`[project]
name = "foo"
`), 0644)
	assert.False(t, hasRuffInPyproject(path))

	// Nonexistent file.
	assert.False(t, hasRuffInPyproject(filepath.Join(dir, "nonexistent.toml")))
}

func TestParseGoWorkModules(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "go.work", `go 1.22

use (
	./svc-a
	./svc-b
)

use ./standalone
`)

	modules := parseGoWorkModules(dir)
	assert.Contains(t, modules, "./svc-a")
	assert.Contains(t, modules, "./svc-b")
	assert.Contains(t, modules, "./standalone")
}

func TestParseCargoWorkspaceMembers(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "Cargo.toml")
	os.WriteFile(path, []byte(`[workspace]
members = [
    "crate-a",
    "crate-b",
    "crate-c",
]
`), 0644)

	members := parseCargoWorkspaceMembers(path)
	assert.Contains(t, members, "crate-a")
	assert.Contains(t, members, "crate-b")
	assert.Contains(t, members, "crate-c")
}

func TestParseCargoWorkspaceMembers_Inline(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "Cargo.toml")
	os.WriteFile(path, []byte(`[workspace]
members = ["alpha", "beta"]
`), 0644)

	members := parseCargoWorkspaceMembers(path)
	assert.Contains(t, members, "alpha")
	assert.Contains(t, members, "beta")
}

func TestDetectCommitFormat(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   string
	}{
		{
			name: "conventional",
			output: `feat: add auth
fix(api): timeout
docs: update readme
refactor: clean up
test: add tests`,
			want: "conventional",
		},
		{
			name: "mixed_below_threshold",
			output: `feat: add auth
Update README
Fix bug
Add feature
Clean up code`,
			want: "unknown",
		},
		{
			name:   "empty",
			output: "",
			want:   "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &scanConfig{
				runCmd: func(name string, args ...string) ([]byte, error) {
					return []byte(tt.output), nil
				},
			}
			got := detectCommitFormat(cfg)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDetectBranchNaming(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   string
	}{
		{
			name: "feature_branches",
			output: `origin/main
origin/feature/auth
origin/feature/ui
origin/fix/bug-123
origin/release/v1.0`,
			want: "feature/*",
		},
		{
			name: "no_pattern",
			output: `origin/main
origin/develop
origin/staging`,
			want: "unknown",
		},
		{
			name:   "empty",
			output: "",
			want:   "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &scanConfig{
				runCmd: func(name string, args ...string) ([]byte, error) {
					return []byte(tt.output), nil
				},
			}
			got := detectBranchNaming(cfg)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestExtractRunSteps(t *testing.T) {
	data := map[string]interface{}{
		"jobs": map[string]interface{}{
			"test": map[string]interface{}{
				"steps": []interface{}{
					map[string]interface{}{"run": "go test ./..."},
					map[string]interface{}{"run": "golangci-lint run"},
					map[string]interface{}{"uses": "actions/checkout@v4"},
				},
			},
		},
	}

	runs := extractRunSteps(data)
	assert.Contains(t, runs, "go test ./...")
	assert.Contains(t, runs, "golangci-lint run")
	assert.Len(t, runs, 2)
}

func TestScanProject_TestRunnerPriority(t *testing.T) {
	// Ensure CI > Makefile > package.json > heuristic ordering.
	t.Run("CI_overrides_makefile", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "main.go", `package main`)
		writeFile(t, dir, "go.mod", `module test
go 1.22
`)
		writeFile(t, dir, "Makefile", `.PHONY: test
test:
	go test -race ./...
`)
		writeFile(t, dir, ".github/workflows/ci.yml", `name: CI
on: [push]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - run: make test
`)
		profile, err := ScanProject(dir, WithRunCmd(func(name string, args ...string) ([]byte, error) {
			return nil, fmt.Errorf("not available in test")
		}))
		require.NoError(t, err)
		assert.Equal(t, "make test", profile.TestRunner.Command)
		assert.Equal(t, "ci", profile.TestRunner.Source)
	})

	t.Run("makefile_overrides_heuristic", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "main.go", `package main`)
		writeFile(t, dir, "go.mod", `module test
go 1.22
`)
		writeFile(t, dir, "Makefile", `.PHONY: test
test:
	go test -race ./...
`)
		profile, err := ScanProject(dir, WithRunCmd(func(name string, args ...string) ([]byte, error) {
			return nil, fmt.Errorf("not available in test")
		}))
		require.NoError(t, err)
		assert.Equal(t, "make test", profile.TestRunner.Command)
		assert.Equal(t, "makefile", profile.TestRunner.Source)
		assert.Equal(t, "high", profile.TestRunner.Confidence)
	})

	t.Run("package_json_overrides_heuristic", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "index.js", `console.log()`)
		writeFile(t, dir, "package.json", `{"scripts": {"test": "jest"}}`)

		profile, err := ScanProject(dir, WithRunCmd(func(name string, args ...string) ([]byte, error) {
			return nil, fmt.Errorf("not available in test")
		}))
		require.NoError(t, err)
		assert.Equal(t, "npm test", profile.TestRunner.Command)
		assert.Equal(t, "package.json", profile.TestRunner.Source)
		assert.Equal(t, "high", profile.TestRunner.Confidence)
	})

	t.Run("heuristic_fallback", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "main.go", `package main`)
		writeFile(t, dir, "go.mod", `module test
go 1.22
`)

		profile, err := ScanProject(dir, WithRunCmd(func(name string, args ...string) ([]byte, error) {
			return nil, fmt.Errorf("not available in test")
		}))
		require.NoError(t, err)
		assert.Equal(t, "go test ./...", profile.TestRunner.Command)
		assert.Equal(t, "heuristic", profile.TestRunner.Source)
		assert.Equal(t, "low", profile.TestRunner.Confidence)
	})
}
