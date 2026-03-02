package forge

import (
	"testing"
)

func TestEndToEndForgeDetectionWithFiltering(t *testing.T) {
	// Simulate a GitHub repository with git remote output
	mockGitHub := func() (string, error) {
		return "origin\tgit@github.com:re-cinq/wave.git (fetch)\norigin\tgit@github.com:re-cinq/wave.git (push)\n", nil
	}

	// Detect forge
	primary, all, err := DetectPrimary(nil, mockGitHub)
	if err != nil {
		t.Fatalf("DetectPrimary() error = %v", err)
	}
	if primary.Type != GitHub {
		t.Fatalf("expected GitHub, got %q", primary.Type)
	}
	if primary.CLITool != "gh" {
		t.Errorf("expected CLI tool gh, got %q", primary.CLITool)
	}
	if primary.Hostname != "github.com" {
		t.Errorf("expected hostname github.com, got %q", primary.Hostname)
	}
	if len(all) != 1 {
		t.Errorf("expected 1 detection, got %d", len(all))
	}

	// Filter pipelines for detected forge
	allPipelines := []string{
		"gh-implement", "gh-pr-review", "gh-issue-triage",
		"gl-merge-request", "gl-deploy",
		"bb-pull-request",
		"gt-issue-flow",
		"prototype", "hotfix", "speckit-flow",
	}

	filtered := FilterPipelines(primary.Type, allPipelines)

	// Should include gh-* pipelines and universal pipelines only
	expected := map[string]bool{
		"gh-implement":    true,
		"gh-pr-review":    true,
		"gh-issue-triage": true,
		"prototype":       true,
		"hotfix":          true,
		"speckit-flow":    true,
	}

	if len(filtered) != len(expected) {
		t.Fatalf("expected %d filtered pipelines, got %d: %v", len(expected), len(filtered), filtered)
	}
	for _, name := range filtered {
		if !expected[name] {
			t.Errorf("unexpected pipeline in filtered result: %q", name)
		}
	}
}

func TestEndToEndMultiForgeDetection(t *testing.T) {
	// Simulate a mirrored repository with GitHub and GitLab remotes
	mockMultiForge := func() (string, error) {
		return "origin\tgit@github.com:org/repo.git (fetch)\n" +
			"origin\tgit@github.com:org/repo.git (push)\n" +
			"mirror\tgit@gitlab.com:org/repo.git (fetch)\n" +
			"mirror\tgit@gitlab.com:org/repo.git (push)\n", nil
	}

	primary, all, err := DetectPrimary(nil, mockMultiForge)
	if err != nil {
		t.Fatalf("DetectPrimary() error = %v", err)
	}

	// Primary should be GitHub (first detected)
	if primary.Type != GitHub {
		t.Errorf("expected primary GitHub, got %q", primary.Type)
	}

	// All should contain both
	if len(all) != 2 {
		t.Fatalf("expected 2 detections, got %d", len(all))
	}

	// Should be ambiguous
	if !IsAmbiguous(all) {
		t.Error("expected ambiguous detection for multi-forge repos")
	}
}

func TestEndToEndCustomDomainConfig(t *testing.T) {
	// Simulate enterprise GitHub with custom domain
	mockEnterprise := func() (string, error) {
		return "origin\tgit@git.internal.corp:team/project.git (fetch)\norigin\tgit@git.internal.corp:team/project.git (push)\n", nil
	}

	// Without config, should be unknown
	primary, _, err := DetectPrimary(nil, mockEnterprise)
	if err != nil {
		t.Fatalf("DetectPrimary() error = %v", err)
	}
	if primary.Type != Unknown {
		t.Errorf("expected Unknown without config, got %q", primary.Type)
	}

	// With config, should detect as GitHub
	cfg := &ForgeConfig{
		Domains: map[string]string{
			"git.internal.corp": "github",
		},
	}
	primary, _, err = DetectPrimary(cfg, mockEnterprise)
	if err != nil {
		t.Fatalf("DetectPrimary() with config error = %v", err)
	}
	if primary.Type != GitHub {
		t.Errorf("expected GitHub with config, got %q", primary.Type)
	}
	if primary.CLITool != "gh" {
		t.Errorf("expected CLI tool gh, got %q", primary.CLITool)
	}

	// Filter should work with detected type
	pipelines := []string{"gh-implement", "gl-deploy", "hotfix"}
	filtered := FilterPipelines(primary.Type, pipelines)
	if len(filtered) != 2 { // gh-implement + hotfix
		t.Errorf("expected 2 filtered pipelines, got %d: %v", len(filtered), filtered)
	}
}

func TestEndToEndNoRemotes(t *testing.T) {
	// Simulate a repository with no remotes
	mockNoRemotes := func() (string, error) {
		return "", nil
	}

	primary, all, err := DetectPrimary(nil, mockNoRemotes)
	if err != nil {
		t.Fatalf("DetectPrimary() error = %v", err)
	}
	if primary.Type != Unknown {
		t.Errorf("expected Unknown for no remotes, got %q", primary.Type)
	}
	if len(all) != 0 {
		t.Errorf("expected 0 detections, got %d", len(all))
	}

	// With Unknown forge type, FilterPipelines returns all
	pipelines := []string{"gh-implement", "gl-deploy", "hotfix"}
	filtered := FilterPipelines(primary.Type, pipelines)
	if len(filtered) != 3 {
		t.Errorf("expected all 3 pipelines for Unknown forge, got %d", len(filtered))
	}
}
