package meta

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnalyzeCodebase_GoProject(t *testing.T) {
	dir := t.TempDir()

	err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/test\n\ngo 1.25\n"), 0644)
	require.NoError(t, err)

	// Create a few .go files.
	err = os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(dir, "main_test.go"), []byte("package main\n"), 0644)
	require.NoError(t, err)

	profile := AnalyzeCodebase(dir)

	assert.Equal(t, "go", profile.Language)
	assert.Equal(t, "go test ./...", profile.TestCommand)
	assert.Equal(t, "go build ./...", profile.BuildCommand)
	assert.Equal(t, "**/*.go", profile.SourceGlob)
	assert.Equal(t, "single", profile.Structure)
	assert.Equal(t, SizeSmall, profile.Size)
	assert.GreaterOrEqual(t, profile.PackageCount, 1)
}

func TestAnalyzeCodebase_PythonProject(t *testing.T) {
	dir := t.TempDir()

	err := os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte("[project]\nname = \"test\"\n"), 0644)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(dir, "app.py"), []byte("print('hello')\n"), 0644)
	require.NoError(t, err)

	profile := AnalyzeCodebase(dir)

	assert.Equal(t, "python", profile.Language)
	assert.Equal(t, "pytest", profile.TestCommand)
	assert.Equal(t, "", profile.BuildCommand)
	assert.Equal(t, "**/*.py", profile.SourceGlob)
	assert.Equal(t, SizeSmall, profile.Size)
}

func TestAnalyzeCodebase_JavaScriptWithReact(t *testing.T) {
	dir := t.TempDir()

	pkgJSON := `{
	"name": "test-app",
	"dependencies": {
		"react": "^18.0.0",
		"react-dom": "^18.0.0"
	}
}`
	err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(pkgJSON), 0644)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(dir, "index.js"), []byte("import React from 'react';\n"), 0644)
	require.NoError(t, err)

	profile := AnalyzeCodebase(dir)

	assert.Equal(t, "javascript", profile.Language)
	assert.Equal(t, "npm test", profile.TestCommand)
	assert.Equal(t, "npm run build", profile.BuildCommand)
	assert.Equal(t, "react", profile.Framework)
	assert.Equal(t, SizeSmall, profile.Size)
}

func TestAnalyzeCodebase_JavaScriptWithNext(t *testing.T) {
	dir := t.TempDir()

	// next depends on react, but next should be detected first due to priority ordering.
	pkgJSON := `{
	"name": "test-app",
	"dependencies": {
		"next": "^14.0.0",
		"react": "^18.0.0"
	}
}`
	err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(pkgJSON), 0644)
	require.NoError(t, err)

	profile := AnalyzeCodebase(dir)

	assert.Equal(t, "javascript", profile.Language)
	assert.Equal(t, "nextjs", profile.Framework)
}

func TestAnalyzeCodebase_JavaScriptWithVue(t *testing.T) {
	dir := t.TempDir()

	pkgJSON := `{
	"name": "test-app",
	"dependencies": {
		"vue": "^3.0.0"
	}
}`
	err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(pkgJSON), 0644)
	require.NoError(t, err)

	profile := AnalyzeCodebase(dir)

	assert.Equal(t, "vue", profile.Framework)
}

func TestAnalyzeCodebase_JavaScriptWithExpress(t *testing.T) {
	dir := t.TempDir()

	pkgJSON := `{
	"name": "test-app",
	"dependencies": {
		"express": "^4.18.0"
	}
}`
	err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(pkgJSON), 0644)
	require.NoError(t, err)

	profile := AnalyzeCodebase(dir)

	assert.Equal(t, "express", profile.Framework)
}

func TestAnalyzeCodebase_JavaScriptNoFramework(t *testing.T) {
	dir := t.TempDir()

	pkgJSON := `{
	"name": "test-app",
	"dependencies": {
		"lodash": "^4.17.0"
	}
}`
	err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(pkgJSON), 0644)
	require.NoError(t, err)

	profile := AnalyzeCodebase(dir)

	assert.Equal(t, "javascript", profile.Language)
	assert.Equal(t, "", profile.Framework)
}

func TestAnalyzeCodebase_JavaScriptDevDependencyFramework(t *testing.T) {
	dir := t.TempDir()

	pkgJSON := `{
	"name": "test-app",
	"devDependencies": {
		"react": "^18.0.0"
	}
}`
	err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(pkgJSON), 0644)
	require.NoError(t, err)

	profile := AnalyzeCodebase(dir)

	assert.Equal(t, "react", profile.Framework)
}

func TestAnalyzeCodebase_GoMonorepo(t *testing.T) {
	dir := t.TempDir()

	// Root go.mod.
	err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/monorepo\n\ngo 1.25\n"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n"), 0644)
	require.NoError(t, err)

	// Sub-module go.mod.
	subDir := filepath.Join(dir, "services", "api")
	err = os.MkdirAll(subDir, 0755)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(subDir, "go.mod"), []byte("module example.com/monorepo/services/api\n\ngo 1.25\n"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(subDir, "handler.go"), []byte("package api\n"), 0644)
	require.NoError(t, err)

	profile := AnalyzeCodebase(dir)

	assert.Equal(t, "go", profile.Language)
	assert.Equal(t, "monorepo", profile.Structure)
	assert.Equal(t, SizeMonorepo, profile.Size)
	assert.GreaterOrEqual(t, profile.PackageCount, 2)
}

func TestAnalyzeCodebase_UnknownProject(t *testing.T) {
	dir := t.TempDir()

	profile := AnalyzeCodebase(dir)

	assert.Equal(t, "unknown", profile.Language)
	assert.Equal(t, "", profile.TestCommand)
	assert.Equal(t, "", profile.BuildCommand)
	assert.Equal(t, "", profile.SourceGlob)
	assert.Equal(t, "single", profile.Structure)
	assert.Equal(t, SizeSmall, profile.Size)
	assert.Equal(t, 1, profile.PackageCount)
}

func TestAnalyzeCodebase_JavaMavenProject(t *testing.T) {
	dir := t.TempDir()

	pomXML := `<project>
  <modelVersion>4.0.0</modelVersion>
  <groupId>com.example</groupId>
  <artifactId>test-app</artifactId>
  <version>1.0</version>
</project>`
	err := os.WriteFile(filepath.Join(dir, "pom.xml"), []byte(pomXML), 0644)
	require.NoError(t, err)

	srcDir := filepath.Join(dir, "src", "main", "java")
	err = os.MkdirAll(srcDir, 0755)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(srcDir, "App.java"), []byte("public class App {}\n"), 0644)
	require.NoError(t, err)

	profile := AnalyzeCodebase(dir)

	assert.Equal(t, "java", profile.Language)
	assert.Equal(t, "mvn test", profile.TestCommand)
	assert.Equal(t, "mvn package", profile.BuildCommand)
	assert.Equal(t, "**/*.java", profile.SourceGlob)
}

func TestAnalyzeCodebase_JavaGradleProject(t *testing.T) {
	dir := t.TempDir()

	err := os.WriteFile(filepath.Join(dir, "build.gradle"), []byte("apply plugin: 'java'\n"), 0644)
	require.NoError(t, err)

	profile := AnalyzeCodebase(dir)

	assert.Equal(t, "java", profile.Language)
	assert.Equal(t, "gradle test", profile.TestCommand)
	assert.Equal(t, "gradle build", profile.BuildCommand)
}

func TestAnalyzeCodebase_RustProject(t *testing.T) {
	dir := t.TempDir()

	cargoToml := `[package]
name = "test-app"
version = "0.1.0"
edition = "2021"
`
	err := os.WriteFile(filepath.Join(dir, "Cargo.toml"), []byte(cargoToml), 0644)
	require.NoError(t, err)

	srcDir := filepath.Join(dir, "src")
	err = os.MkdirAll(srcDir, 0755)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(srcDir, "main.rs"), []byte("fn main() {}\n"), 0644)
	require.NoError(t, err)

	profile := AnalyzeCodebase(dir)

	assert.Equal(t, "rust", profile.Language)
	assert.Equal(t, "cargo test", profile.TestCommand)
	assert.Equal(t, "cargo build", profile.BuildCommand)
	assert.Equal(t, "**/*.rs", profile.SourceGlob)
}

func TestAnalyzeCodebase_RustWorkspace(t *testing.T) {
	dir := t.TempDir()

	cargoToml := `[workspace]
members = ["crate-a", "crate-b"]
`
	err := os.WriteFile(filepath.Join(dir, "Cargo.toml"), []byte(cargoToml), 0644)
	require.NoError(t, err)

	profile := AnalyzeCodebase(dir)

	assert.Equal(t, "rust", profile.Language)
	assert.Equal(t, "workspace", profile.Structure)
	assert.Equal(t, SizeMonorepo, profile.Size)
}

func TestAnalyzeCodebase_RubyProject(t *testing.T) {
	dir := t.TempDir()

	err := os.WriteFile(filepath.Join(dir, "Gemfile"), []byte("source 'https://rubygems.org'\n"), 0644)
	require.NoError(t, err)

	profile := AnalyzeCodebase(dir)

	assert.Equal(t, "ruby", profile.Language)
	assert.Equal(t, "bundle exec rspec", profile.TestCommand)
	assert.Equal(t, "", profile.BuildCommand)
	assert.Equal(t, "**/*.rb", profile.SourceGlob)
}

func TestAnalyzeCodebase_PythonSetupPy(t *testing.T) {
	dir := t.TempDir()

	err := os.WriteFile(filepath.Join(dir, "setup.py"), []byte("from setuptools import setup\n"), 0644)
	require.NoError(t, err)

	profile := AnalyzeCodebase(dir)

	assert.Equal(t, "python", profile.Language)
	assert.Equal(t, "pytest", profile.TestCommand)
}

func TestAnalyzeCodebase_JavaScriptWorkspace(t *testing.T) {
	dir := t.TempDir()

	pkgJSON := `{
	"name": "monorepo",
	"workspaces": ["packages/*"]
}`
	err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(pkgJSON), 0644)
	require.NoError(t, err)

	profile := AnalyzeCodebase(dir)

	assert.Equal(t, "javascript", profile.Language)
	assert.Equal(t, "workspace", profile.Structure)
	assert.Equal(t, SizeMonorepo, profile.Size)
}

func TestAnalyzeCodebase_SizeClassification(t *testing.T) {
	tests := []struct {
		name      string
		fileCount int
		expected  SizeClass
	}{
		{"zero files", 0, SizeSmall},
		{"few files", 10, SizeSmall},
		{"99 files", 99, SizeSmall},
		{"100 files", 100, SizeMedium},
		{"500 files", 500, SizeMedium},
		{"1000 files", 1000, SizeMedium},
		{"1001 files", 1001, SizeLarge},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := classifySize(tt.fileCount)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAnalyzeCodebase_MediumSizeGoProject(t *testing.T) {
	dir := t.TempDir()

	err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/test\n\ngo 1.25\n"), 0644)
	require.NoError(t, err)

	// Create 150 .go files across multiple packages to trigger SizeMedium.
	for i := 0; i < 150; i++ {
		pkgDir := filepath.Join(dir, "pkg", string(rune('a'+i%26)))
		err = os.MkdirAll(pkgDir, 0755)
		require.NoError(t, err)
		fileName := filepath.Join(pkgDir, "file"+string(rune('a'+i/26))+".go")
		err = os.WriteFile(fileName, []byte("package x\n"), 0644)
		require.NoError(t, err)
	}

	profile := AnalyzeCodebase(dir)

	assert.Equal(t, "go", profile.Language)
	assert.Equal(t, SizeMedium, profile.Size)
	assert.Greater(t, profile.PackageCount, 1)
}

func TestAnalyzeCodebase_LanguagePriority(t *testing.T) {
	// When both go.mod and package.json exist, go.mod should win (higher priority).
	dir := t.TempDir()

	err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/test\n\ngo 1.25\n"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name": "test"}`), 0644)
	require.NoError(t, err)

	profile := AnalyzeCodebase(dir)

	assert.Equal(t, "go", profile.Language)
}

func TestCountSourceFiles_SkipsHiddenDirs(t *testing.T) {
	dir := t.TempDir()

	// Create a .go file in a hidden directory — should be skipped.
	hiddenDir := filepath.Join(dir, ".hidden")
	err := os.MkdirAll(hiddenDir, 0755)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(hiddenDir, "hidden.go"), []byte("package hidden\n"), 0644)
	require.NoError(t, err)

	// Create a visible .go file.
	err = os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n"), 0644)
	require.NoError(t, err)

	count := countSourceFiles(dir, []string{".go"})
	assert.Equal(t, 1, count, "should only count the visible file")
}

func TestCountSourceFiles_SkipsNodeModules(t *testing.T) {
	dir := t.TempDir()

	nmDir := filepath.Join(dir, "node_modules", "react")
	err := os.MkdirAll(nmDir, 0755)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(nmDir, "index.js"), []byte("module.exports = {};\n"), 0644)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(dir, "index.js"), []byte("console.log('hello');\n"), 0644)
	require.NoError(t, err)

	count := countSourceFiles(dir, []string{".js"})
	assert.Equal(t, 1, count, "should not count files in node_modules")
}

func TestCountDirsContainingExt(t *testing.T) {
	dir := t.TempDir()

	// Create .go files in two different directories.
	pkg1 := filepath.Join(dir, "pkg1")
	pkg2 := filepath.Join(dir, "pkg2")
	err := os.MkdirAll(pkg1, 0755)
	require.NoError(t, err)
	err = os.MkdirAll(pkg2, 0755)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(pkg1, "a.go"), []byte("package pkg1\n"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(pkg1, "b.go"), []byte("package pkg1\n"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(pkg2, "c.go"), []byte("package pkg2\n"), 0644)
	require.NoError(t, err)

	count := countDirsContainingExt(dir, ".go")
	assert.Equal(t, 2, count)
}

func TestCountMarkerFiles(t *testing.T) {
	dir := t.TempDir()

	// Create go.mod at root and in a subdirectory.
	err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module root\n"), 0644)
	require.NoError(t, err)

	subDir := filepath.Join(dir, "sub")
	err = os.MkdirAll(subDir, 0755)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(subDir, "go.mod"), []byte("module sub\n"), 0644)
	require.NoError(t, err)

	count := countMarkerFiles(dir, "go.mod")
	assert.Equal(t, 2, count)
}

func TestFileExists(t *testing.T) {
	dir := t.TempDir()

	// Non-existent file.
	assert.False(t, fileExists(filepath.Join(dir, "nope.txt")))

	// Existing file.
	path := filepath.Join(dir, "exists.txt")
	err := os.WriteFile(path, []byte("hello"), 0644)
	require.NoError(t, err)
	assert.True(t, fileExists(path))

	// Directory should return false (not a file).
	assert.False(t, fileExists(dir))
}

func TestDetectFramework_InvalidJSON(t *testing.T) {
	dir := t.TempDir()

	err := os.WriteFile(filepath.Join(dir, "package.json"), []byte("{invalid json"), 0644)
	require.NoError(t, err)

	result := detectFramework(dir)
	assert.Equal(t, "", result)
}

func TestDetectFramework_MissingPackageJSON(t *testing.T) {
	dir := t.TempDir()

	result := detectFramework(dir)
	assert.Equal(t, "", result)
}
