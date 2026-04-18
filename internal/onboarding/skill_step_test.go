package onboarding

import (
	"reflect"
	"testing"

	"github.com/recinq/wave/internal/manifest"
)

func TestSkillSelectionStep_Name(t *testing.T) {
	s := &SkillSelectionStep{}
	if got := s.Name(); got != "Skill Selection" {
		t.Errorf("Name() = %q, want %q", got, "Skill Selection")
	}
}

func TestSkillSelectionStep_NonInteractive_NoExisting(t *testing.T) {
	s := &SkillSelectionStep{}
	cfg := &WizardConfig{Interactive: false}

	result, err := s.Run(cfg)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	skills, ok := result.Data["skills"].([]string)
	if !ok {
		t.Fatalf("expected []string, got %T", result.Data["skills"])
	}
	if len(skills) != 0 {
		t.Errorf("expected empty skills, got %v", skills)
	}
}

func TestSkillSelectionStep_ReconfigurePreservesExisting(t *testing.T) {
	s := &SkillSelectionStep{}
	cfg := &WizardConfig{
		Interactive: false,
		Reconfigure: true,
		Existing:    &manifest.Manifest{Skills: []string{"golang", "gh-cli"}},
	}

	result, err := s.Run(cfg)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	got := result.Data["skills"].([]string)
	want := []string{"golang", "gh-cli"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("skills = %v, want %v", got, want)
	}
}
