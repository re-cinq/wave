package skill

import (
	"fmt"
	"testing"
)

func TestClassifySkill(t *testing.T) {
	tests := []struct {
		name         string
		skill        Skill
		wantTag      string
		wantRefCount int
	}{
		{
			name: "standalone skill with no wave refs",
			skill: Skill{
				Name:        "golang",
				Description: "Go development patterns",
				Body:        "Use Go best practices. Follow effective Go guidelines.",
			},
			wantTag:      TagStandalone,
			wantRefCount: 0,
		},
		{
			name: "wave-specific skill with many refs",
			skill: Skill{
				Name:        "wave",
				Description: "Wave development",
				Body:        "Use wave run to execute pipelines. The wave.yaml manifest configures personas. Each persona runs in an ephemeral worktree. Use wave init to onboard. The .wave/ directory contains pipelines, personas, and manifests. Wave manages pipeline execution.",
			},
			wantTag: TagWaveSpecific,
		},
		{
			name: "both classification with moderate refs",
			skill: Skill{
				Name:        "speckit",
				Description: "Spec driven development",
				Body:        "Integrate with wave for pipeline-based specs.",
			},
			wantTag: TagBoth,
		},
		{
			name: "empty body is standalone",
			skill: Skill{
				Name:        "empty",
				Description: "empty skill",
				Body:        "",
			},
			wantTag:      TagStandalone,
			wantRefCount: 0,
		},
		{
			name: "case insensitive detection",
			skill: Skill{
				Name:        "mixed",
				Description: "mixed case",
				Body:        "Use WAVE.YAML for configuration. PERSONA definitions.",
			},
			wantTag: TagBoth,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifySkill(tt.skill)
			if got.Tag != tt.wantTag {
				t.Errorf("Tag = %q, want %q (WaveRefCount=%d)", got.Tag, tt.wantTag, got.WaveRefCount)
			}
			if tt.wantRefCount > 0 && got.WaveRefCount != tt.wantRefCount {
				t.Errorf("WaveRefCount = %d, want %d", got.WaveRefCount, tt.wantRefCount)
			}
			if got.Name != tt.skill.Name {
				t.Errorf("Name = %q, want %q", got.Name, tt.skill.Name)
			}
		})
	}
}

func TestClassifySkillWarnings(t *testing.T) {
	s := Skill{
		Name: "test",
		Body: "no wave refs",
	}
	c := ClassifySkill(s)
	if len(c.Warnings) != 2 {
		t.Errorf("expected 2 warnings (description + license), got %d: %v", len(c.Warnings), c.Warnings)
	}
}

type classifyMockStore struct {
	skills map[string]Skill
}

func (m *classifyMockStore) Read(name string) (Skill, error) {
	s, ok := m.skills[name]
	if !ok {
		return Skill{}, fmt.Errorf("%w: %s", ErrNotFound, name)
	}
	return s, nil
}

func (m *classifyMockStore) ReadMetadata(name string) (Skill, error) {
	s, ok := m.skills[name]
	if !ok {
		return Skill{}, fmt.Errorf("%w: %s", ErrNotFound, name)
	}
	s.Body = ""
	return s, nil
}

func (m *classifyMockStore) Write(_ Skill) error { return nil }
func (m *classifyMockStore) List() ([]Skill, error) {
	var result []Skill
	for _, s := range m.skills {
		result = append(result, s)
	}
	return result, nil
}
func (m *classifyMockStore) Delete(_ string) error { return nil }

func TestClassifyAll(t *testing.T) {
	store := &classifyMockStore{
		skills: map[string]Skill{
			"golang": {Name: "golang", Description: "Go", Body: "pure go patterns"},
			"wave":   {Name: "wave", Description: "Wave", Body: "use wave run in a pipeline with persona and wave.yaml. Use wave init with .wave/ manifest and worktree and wave again and more pipeline and persona and wave"},
		},
	}

	results, err := store.List()
	if err != nil {
		t.Fatal(err)
	}
	_ = results

	classifications, err := ClassifyAll(store)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(classifications) != 2 {
		t.Fatalf("expected 2 classifications, got %d", len(classifications))
	}

	tagMap := make(map[string]string)
	for _, c := range classifications {
		tagMap[c.Name] = c.Tag
	}

	if tagMap["golang"] != TagStandalone {
		t.Errorf("golang should be standalone, got %q", tagMap["golang"])
	}
	if tagMap["wave"] != TagWaveSpecific {
		t.Errorf("wave should be wave-specific, got %q", tagMap["wave"])
	}
}

func TestClassifyAllEmptyStore(t *testing.T) {
	store := &classifyMockStore{skills: map[string]Skill{}}
	classifications, err := ClassifyAll(store)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(classifications) != 0 {
		t.Errorf("expected 0 classifications, got %d", len(classifications))
	}
}
