package doctor

import (
	"testing"

	"github.com/recinq/wave/internal/forge"
	"github.com/recinq/wave/internal/manifest"
)

func TestOptimize_GoProjectWithMakefileOverride(t *testing.T) {
	profile := &ProjectProfile{
		Languages: []LanguageInfo{
			{Name: "Go", Extensions: []string{".go"}, FileCount: 150, Percentage: 85},
		},
		BuildSystem: BuildSystemInfo{
			Name:    "make",
			File:    "Makefile",
			Targets: []string{"build", "test", "lint"},
		},
		TestRunner: TestRunnerInfo{
			Command:    "make test",
			Source:     "makefile",
			Confidence: "high",
		},
	}

	current := &manifest.Project{
		Language:    "go",
		TestCommand: "go test ./...",
	}

	result := Optimize(profile, current, nil, nil)

	// Should propose upgrading test_command from simple default to makefile target
	found := false
	for _, c := range result.ProjectChanges {
		if c.Field == "project.test_command" {
			found = true
			if c.Current != "go test ./..." {
				t.Errorf("test_command Current = %q, want %q", c.Current, "go test ./...")
			}
			if c.Proposed != "make test" {
				t.Errorf("test_command Proposed = %q, want %q", c.Proposed, "make test")
			}
			if c.Source != "makefile" {
				t.Errorf("test_command Source = %q, want %q", c.Source, "makefile")
			}
		}
	}
	if !found {
		t.Error("expected test_command change, got none")
	}
}

func TestOptimize_NodeJSProjectLintFromESLint(t *testing.T) {
	profile := &ProjectProfile{
		Languages: []LanguageInfo{
			{Name: "JavaScript", Extensions: []string{".js", ".jsx"}, FileCount: 200, Percentage: 70},
		},
		BuildSystem: BuildSystemInfo{
			Name: "npm",
			File: "package.json",
		},
		LintTools: []LintToolInfo{
			{Name: "eslint", ConfigFile: ".eslintrc.json", Command: "npx eslint ."},
		},
		TestRunner: TestRunnerInfo{
			Command:    "npm test",
			Source:     "package.json",
			Confidence: "medium",
		},
	}

	current := &manifest.Project{
		Language: "javascript",
	}

	result := Optimize(profile, current, nil, nil)

	// Should propose lint_command from eslint detection
	found := false
	for _, c := range result.ProjectChanges {
		if c.Field == "project.lint_command" {
			found = true
			if c.Current != "" {
				t.Errorf("lint_command Current = %q, want empty", c.Current)
			}
			if c.Proposed != "npx eslint ." {
				t.Errorf("lint_command Proposed = %q, want %q", c.Proposed, "npx eslint .")
			}
		}
	}
	if !found {
		t.Error("expected lint_command change, got none")
	}
}

func TestOptimize_EmptyCurrentConfig(t *testing.T) {
	profile := &ProjectProfile{
		Languages: []LanguageInfo{
			{Name: "Go", Extensions: []string{".go"}, FileCount: 100, Percentage: 90},
		},
		BuildSystem: BuildSystemInfo{
			Name:    "make",
			File:    "Makefile",
			Targets: []string{"build", "test", "lint"},
		},
		TestRunner: TestRunnerInfo{
			Command:    "make test",
			Source:     "makefile",
			Confidence: "high",
		},
		LintTools: []LintToolInfo{
			{Name: "golangci-lint", ConfigFile: ".golangci.yml", Command: "golangci-lint run"},
		},
		Conventions: ConventionInfo{
			CommitFormat:    "conventional",
			HasEditorConfig: true,
		},
	}

	current := &manifest.Project{} // all empty

	result := Optimize(profile, current, nil, nil)

	// Should propose all fields
	fields := make(map[string]string)
	for _, c := range result.ProjectChanges {
		fields[c.Field] = c.Proposed
	}

	if fields["project.language"] != "go" {
		t.Errorf("language = %q, want %q", fields["project.language"], "go")
	}
	if fields["project.test_command"] != "make test" {
		t.Errorf("test_command = %q, want %q", fields["project.test_command"], "make test")
	}
	// Makefile has lint target, so "make lint" takes precedence
	if fields["project.lint_command"] != "make lint" {
		t.Errorf("lint_command = %q, want %q", fields["project.lint_command"], "make lint")
	}
	if fields["project.build_command"] != "make build" {
		t.Errorf("build_command = %q, want %q", fields["project.build_command"], "make build")
	}
	if fields["project.source_glob"] != "**/*.go" {
		t.Errorf("source_glob = %q, want %q", fields["project.source_glob"], "**/*.go")
	}

	if !result.HasChanges() {
		t.Error("expected HasChanges() to return true for empty config")
	}
}

func TestOptimize_MatchingConfig_NoChanges(t *testing.T) {
	profile := &ProjectProfile{
		Languages: []LanguageInfo{
			{Name: "Go", Extensions: []string{".go"}, FileCount: 100, Percentage: 90},
		},
		BuildSystem: BuildSystemInfo{
			Name:    "make",
			File:    "Makefile",
			Targets: []string{"build", "test"},
		},
		TestRunner: TestRunnerInfo{
			Command:    "make test",
			Source:     "makefile",
			Confidence: "high",
		},
		LintTools: []LintToolInfo{
			{Name: "golangci-lint", ConfigFile: ".golangci.yml", Command: "golangci-lint run"},
		},
	}

	current := &manifest.Project{
		Language:     "go",
		TestCommand:  "make test",
		LintCommand:  "golangci-lint run",
		BuildCommand: "make build",
		SourceGlob:   "**/*.go",
	}

	result := Optimize(profile, current, nil, nil)

	if result.HasChanges() {
		t.Errorf("expected no changes, but HasChanges() returned true")
		for _, c := range result.ProjectChanges {
			t.Logf("  change: %s: %q -> %q", c.Field, c.Current, c.Proposed)
		}
	}
}

func TestOptimize_PipelineRecommendations_GitHubForge(t *testing.T) {
	fi := &forge.ForgeInfo{
		Type:           forge.ForgeGitHub,
		PipelinePrefix: "gh",
	}

	pipelines := []string{
		"speckit-flow",
		"gh-implement",
		"gh-research",
		"bb-implement",
		"gl-scope",
		"wave-evolve",
		"doc-audit",
	}

	result := Optimize(&ProjectProfile{}, &manifest.Project{}, fi, pipelines)

	recs := make(map[string]PipelineRecommendation)
	for _, r := range result.PipelineRecs {
		recs[r.Name] = r
	}

	// Universal pipelines should be recommended
	for _, name := range []string{"speckit-flow", "wave-evolve", "doc-audit"} {
		rec, ok := recs[name]
		if !ok {
			t.Errorf("missing recommendation for %q", name)
			continue
		}
		if !rec.Recommended {
			t.Errorf("%q should be recommended (universal pipeline), got not recommended: %s", name, rec.Reason)
		}
	}

	// GitHub pipelines should be recommended
	for _, name := range []string{"gh-implement", "gh-research"} {
		rec, ok := recs[name]
		if !ok {
			t.Errorf("missing recommendation for %q", name)
			continue
		}
		if !rec.Recommended {
			t.Errorf("%q should be recommended (matches forge), got not recommended: %s", name, rec.Reason)
		}
	}

	// Wrong-forge pipelines should NOT be recommended
	for _, name := range []string{"bb-implement", "gl-scope"} {
		rec, ok := recs[name]
		if !ok {
			t.Errorf("missing recommendation for %q", name)
			continue
		}
		if rec.Recommended {
			t.Errorf("%q should NOT be recommended (wrong forge), got recommended: %s", name, rec.Reason)
		}
	}
}

func TestOptimize_PipelineRecommendations_UnknownForge(t *testing.T) {
	fi := &forge.ForgeInfo{
		Type: forge.ForgeUnknown,
	}

	pipelines := []string{
		"speckit-flow",
		"gh-implement",
		"bb-implement",
		"gl-scope",
	}

	result := Optimize(&ProjectProfile{}, &manifest.Project{}, fi, pipelines)

	for _, rec := range result.PipelineRecs {
		if !rec.Recommended {
			t.Errorf("%q should be recommended when forge is unknown, got not recommended: %s", rec.Name, rec.Reason)
		}
	}
}

func TestOptimize_ApplyTo(t *testing.T) {
	profile := &ProjectProfile{
		Languages: []LanguageInfo{
			{Name: "Go", Extensions: []string{".go"}, FileCount: 100, Percentage: 90},
		},
		BuildSystem: BuildSystemInfo{
			Name:    "make",
			File:    "Makefile",
			Targets: []string{"build", "test"},
		},
		TestRunner: TestRunnerInfo{
			Command:    "make test",
			Source:     "makefile",
			Confidence: "high",
		},
		LintTools: []LintToolInfo{
			{Name: "golangci-lint", ConfigFile: ".golangci.yml", Command: "golangci-lint run"},
		},
	}

	current := &manifest.Project{
		Language:    "go",
		TestCommand: "go test ./...",
		// LintCommand, BuildCommand, SourceGlob are empty
	}

	result := Optimize(profile, current, nil, nil)
	applied := result.ApplyTo(current)

	// Language should remain unchanged (matches)
	if applied.Language != "go" {
		t.Errorf("Language = %q, want %q", applied.Language, "go")
	}

	// TestCommand should be upgraded
	if applied.TestCommand != "make test" {
		t.Errorf("TestCommand = %q, want %q", applied.TestCommand, "make test")
	}

	// LintCommand should be set
	if applied.LintCommand != "golangci-lint run" {
		t.Errorf("LintCommand = %q, want %q", applied.LintCommand, "golangci-lint run")
	}

	// BuildCommand should be set
	if applied.BuildCommand != "make build" {
		t.Errorf("BuildCommand = %q, want %q", applied.BuildCommand, "make build")
	}

	// SourceGlob should be set
	if applied.SourceGlob != "**/*.go" {
		t.Errorf("SourceGlob = %q, want %q", applied.SourceGlob, "**/*.go")
	}

	// Original should be unmodified
	if current.TestCommand != "go test ./..." {
		t.Error("ApplyTo mutated the original Project")
	}
}

func TestOptimize_MultipleLintTools_PicksPrimary(t *testing.T) {
	profile := &ProjectProfile{
		Languages: []LanguageInfo{
			{Name: "Go", Extensions: []string{".go"}, FileCount: 100, Percentage: 90},
		},
		LintTools: []LintToolInfo{
			{Name: "golangci-lint", ConfigFile: ".golangci.yml", Command: "golangci-lint run"},
			{Name: "staticcheck", ConfigFile: "", Command: "staticcheck ./..."},
			{Name: "gosec", ConfigFile: "", Command: "gosec ./..."},
		},
	}

	current := &manifest.Project{}

	result := Optimize(profile, current, nil, nil)

	found := false
	for _, c := range result.ProjectChanges {
		if c.Field == "project.lint_command" {
			found = true
			if c.Proposed != "golangci-lint run" {
				t.Errorf("lint_command = %q, want %q (should pick first/primary tool)", c.Proposed, "golangci-lint run")
			}
		}
	}
	if !found {
		t.Error("expected lint_command change, got none")
	}
}

func TestOptimize_CISourcedTestCommandPrecedence(t *testing.T) {
	profile := &ProjectProfile{
		Languages: []LanguageInfo{
			{Name: "Go", Extensions: []string{".go"}, FileCount: 100, Percentage: 90},
		},
		TestRunner: TestRunnerInfo{
			Command:    "go test -race -count=1 -coverprofile=coverage.out ./...",
			Source:     "ci",
			Confidence: "high",
		},
	}

	// Current has a simple default
	current := &manifest.Project{
		TestCommand: "go test ./...",
	}

	result := Optimize(profile, current, nil, nil)

	found := false
	for _, c := range result.ProjectChanges {
		if c.Field == "project.test_command" {
			found = true
			if c.Proposed != "go test -race -count=1 -coverprofile=coverage.out ./..." {
				t.Errorf("test_command Proposed = %q, want CI-sourced command", c.Proposed)
			}
			if c.Source != "ci" {
				t.Errorf("test_command Source = %q, want %q", c.Source, "ci")
			}
		}
	}
	if !found {
		t.Error("expected test_command change for CI-sourced override")
	}
}

func TestOptimize_NilProfile(t *testing.T) {
	result := Optimize(nil, &manifest.Project{Language: "go"}, nil, nil)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.HasChanges() {
		t.Error("expected no changes with nil profile")
	}
}

func TestOptimize_NilCurrent(t *testing.T) {
	profile := &ProjectProfile{
		Languages: []LanguageInfo{
			{Name: "Go", Extensions: []string{".go"}, FileCount: 50, Percentage: 80},
		},
	}

	result := Optimize(profile, nil, nil, nil)
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	found := false
	for _, c := range result.ProjectChanges {
		if c.Field == "project.language" {
			found = true
			if c.Proposed != "go" {
				t.Errorf("language Proposed = %q, want %q", c.Proposed, "go")
			}
		}
	}
	if !found {
		t.Error("expected language change when current is nil")
	}
}

func TestOptimize_ConventionsDetected(t *testing.T) {
	profile := &ProjectProfile{
		Conventions: ConventionInfo{
			CommitFormat:    "conventional",
			HasPRTemplate:   true,
			BranchNaming:    "feature/*",
			HasEditorConfig: true,
		},
		HasClaudeMD: true,
		HasDocker:   true,
	}

	result := Optimize(profile, &manifest.Project{}, nil, nil)

	expected := []string{
		"commit format: conventional",
		"pull request template configured",
		"branch naming: feature/*",
		"editorconfig configured",
		"AGENTS.md project instructions present",
		"Docker configuration present",
	}

	if len(result.Conventions) != len(expected) {
		t.Errorf("got %d conventions, want %d", len(result.Conventions), len(expected))
		for _, c := range result.Conventions {
			t.Logf("  got: %s", c)
		}
		return
	}

	for i, want := range expected {
		if result.Conventions[i] != want {
			t.Errorf("Conventions[%d] = %q, want %q", i, result.Conventions[i], want)
		}
	}
}

func TestOptimize_SourceGlob_TypeScript(t *testing.T) {
	profile := &ProjectProfile{
		Languages: []LanguageInfo{
			{Name: "TypeScript", Extensions: []string{".ts", ".tsx"}, FileCount: 80, Percentage: 60},
		},
	}

	result := Optimize(profile, &manifest.Project{}, nil, nil)

	found := false
	for _, c := range result.ProjectChanges {
		if c.Field == "project.source_glob" {
			found = true
			if c.Proposed != "**/*.{ts,tsx}" {
				t.Errorf("source_glob = %q, want %q", c.Proposed, "**/*.{ts,tsx}")
			}
		}
	}
	if !found {
		t.Error("expected source_glob change for TypeScript")
	}
}

func TestOptimize_BuildCommand_GoDefault(t *testing.T) {
	profile := &ProjectProfile{
		Languages: []LanguageInfo{
			{Name: "Go", Extensions: []string{".go"}, FileCount: 100, Percentage: 90},
		},
		BuildSystem: BuildSystemInfo{
			Name: "go",
			File: "go.mod",
		},
	}

	result := Optimize(profile, &manifest.Project{}, nil, nil)

	found := false
	for _, c := range result.ProjectChanges {
		if c.Field == "project.build_command" {
			found = true
			if c.Proposed != "go build ./..." {
				t.Errorf("build_command = %q, want %q", c.Proposed, "go build ./...")
			}
		}
	}
	if !found {
		t.Error("expected build_command change for Go project")
	}
}

func TestOptimize_MakefileLintTarget_OverridesToolCommand(t *testing.T) {
	profile := &ProjectProfile{
		Languages: []LanguageInfo{
			{Name: "Go", Extensions: []string{".go"}, FileCount: 100, Percentage: 90},
		},
		BuildSystem: BuildSystemInfo{
			Name:    "make",
			File:    "Makefile",
			Targets: []string{"build", "test", "lint"},
		},
		LintTools: []LintToolInfo{
			{Name: "golangci-lint", ConfigFile: ".golangci.yml", Command: "golangci-lint run"},
		},
	}

	current := &manifest.Project{}

	result := Optimize(profile, current, nil, nil)

	for _, c := range result.ProjectChanges {
		if c.Field == "project.lint_command" {
			if c.Proposed != "make lint" {
				t.Errorf("lint_command = %q, want %q (Makefile lint target should take precedence)", c.Proposed, "make lint")
			}
			return
		}
	}
	t.Error("expected lint_command change")
}

func TestOptimize_BasicLintDefault_Upgrade(t *testing.T) {
	profile := &ProjectProfile{
		Languages: []LanguageInfo{
			{Name: "Go", Extensions: []string{".go"}, FileCount: 100, Percentage: 90},
		},
		LintTools: []LintToolInfo{
			{Name: "golangci-lint", ConfigFile: ".golangci.yml", Command: "golangci-lint run"},
		},
	}

	current := &manifest.Project{
		LintCommand: "go vet ./...",
	}

	result := Optimize(profile, current, nil, nil)

	found := false
	for _, c := range result.ProjectChanges {
		if c.Field == "project.lint_command" {
			found = true
			if c.Proposed != "golangci-lint run" {
				t.Errorf("lint_command = %q, want %q", c.Proposed, "golangci-lint run")
			}
		}
	}
	if !found {
		t.Error("expected lint_command upgrade from basic default")
	}
}

func TestHasChanges_NoChanges(t *testing.T) {
	result := &OptimizeResult{
		ProjectChanges: []ConfigChange{
			{Field: "project.language", Current: "go", Proposed: "go"},
		},
	}
	if result.HasChanges() {
		t.Error("HasChanges() should return false when Current == Proposed")
	}
}

func TestHasChanges_WithChanges(t *testing.T) {
	result := &OptimizeResult{
		ProjectChanges: []ConfigChange{
			{Field: "project.language", Current: "", Proposed: "go"},
		},
	}
	if !result.HasChanges() {
		t.Error("HasChanges() should return true when Current != Proposed")
	}
}

func TestApplyTo_NilOriginal(t *testing.T) {
	result := &OptimizeResult{
		ProjectChanges: []ConfigChange{
			{Field: "project.language", Current: "", Proposed: "go"},
			{Field: "project.test_command", Current: "", Proposed: "make test"},
		},
	}

	applied := result.ApplyTo(nil)
	if applied.Language != "go" {
		t.Errorf("Language = %q, want %q", applied.Language, "go")
	}
	if applied.TestCommand != "make test" {
		t.Errorf("TestCommand = %q, want %q", applied.TestCommand, "make test")
	}
}

func TestApplyTo_SkipsMatchingValues(t *testing.T) {
	result := &OptimizeResult{
		ProjectChanges: []ConfigChange{
			{Field: "project.language", Current: "go", Proposed: "go"}, // same = skip
			{Field: "project.test_command", Current: "", Proposed: "make test"},
		},
	}

	original := &manifest.Project{Language: "go"}
	applied := result.ApplyTo(original)

	if applied.Language != "go" {
		t.Errorf("Language = %q, want %q", applied.Language, "go")
	}
	if applied.TestCommand != "make test" {
		t.Errorf("TestCommand = %q, want %q", applied.TestCommand, "make test")
	}
}

func TestClassifyPipeline_UniversalPipeline(t *testing.T) {
	rec := classifyPipeline("speckit-flow", forge.ForgeGitHub, "gh")
	if !rec.Recommended {
		t.Error("universal pipeline should be recommended")
	}
}

func TestClassifyPipeline_MatchingForge(t *testing.T) {
	rec := classifyPipeline("gh-implement", forge.ForgeGitHub, "gh")
	if !rec.Recommended {
		t.Error("matching forge pipeline should be recommended")
	}
}

func TestClassifyPipeline_WrongForge(t *testing.T) {
	rec := classifyPipeline("bb-implement", forge.ForgeGitHub, "gh")
	if rec.Recommended {
		t.Error("wrong forge pipeline should NOT be recommended")
	}
}

func TestExtractForgePrefix(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"gh-implement", "gh"},
		{"gl-scope", "gl"},
		{"bb-refresh", "bb"},
		{"gt-implement", "gt"},
		{"speckit-flow", ""},
		{"wave-evolve", ""},
		{"doc-audit", ""},
	}
	for _, tt := range tests {
		got := extractForgePrefix(tt.name)
		if got != tt.want {
			t.Errorf("extractForgePrefix(%q) = %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestSourceGlobForLanguage(t *testing.T) {
	tests := []struct {
		lang LanguageInfo
		want string
	}{
		{LanguageInfo{Name: "Go", Extensions: []string{".go"}}, "**/*.go"},
		{LanguageInfo{Name: "TypeScript", Extensions: []string{".ts", ".tsx"}}, "**/*.{ts,tsx}"},
		{LanguageInfo{Name: "Python", Extensions: []string{".py"}}, "**/*.py"},
		{LanguageInfo{Name: "Rust", Extensions: []string{".rs"}}, "**/*.rs"},
		{LanguageInfo{Name: "JavaScript", Extensions: []string{".js", ".jsx"}}, "**/*.{js,jsx}"},
		{LanguageInfo{Name: "UnknownLang", Extensions: []string{".xyz"}}, "**/*.xyz"},
		{LanguageInfo{Name: "UnknownMulti", Extensions: []string{".abc", ".def"}}, "**/*.{abc,def}"},
		{LanguageInfo{Name: "Empty", Extensions: []string{}}, ""},
	}
	for _, tt := range tests {
		got := sourceGlobForLanguage(tt.lang)
		if got != tt.want {
			t.Errorf("sourceGlobForLanguage(%q) = %q, want %q", tt.lang.Name, got, tt.want)
		}
	}
}
