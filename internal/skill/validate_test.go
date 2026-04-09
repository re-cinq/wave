package skill

import (
	"fmt"
	"strings"
	"testing"
)

// mockStore implements Store for testing ValidateSkillRefs.
type mockStore struct {
	skills map[string]Skill
}

func (m *mockStore) Read(name string) (Skill, error) {
	s, ok := m.skills[name]
	if !ok {
		return Skill{}, fmt.Errorf("%w: %s", ErrNotFound, name)
	}
	return s, nil
}

func (m *mockStore) Write(_ Skill) error    { return nil }
func (m *mockStore) List() ([]Skill, error) { return nil, nil }
func (m *mockStore) Delete(_ string) error  { return nil }

func TestValidateSkillRefs(t *testing.T) {
	store := &mockStore{
		skills: map[string]Skill{
			"golang":  {Name: "golang", Description: "Go skill"},
			"speckit": {Name: "speckit", Description: "Speckit skill"},
		},
	}

	tests := []struct {
		name       string
		names      []string
		scope      string
		store      Store
		wantCount  int
		wantSubstr []string // substrings expected in error messages
	}{
		{
			name:      "empty names no errors",
			names:     []string{},
			scope:     "global",
			store:     store,
			wantCount: 0,
		},
		{
			name:      "nil names no errors",
			names:     nil,
			scope:     "global",
			store:     store,
			wantCount: 0,
		},
		{
			name:      "valid names with nil store",
			names:     []string{"golang", "speckit"},
			scope:     "global",
			store:     nil,
			wantCount: 0,
		},
		{
			name:       "invalid format name reports scope",
			names:      []string{"INVALID"},
			scope:      "persona:navigator",
			store:      nil,
			wantCount:  1,
			wantSubstr: []string{"persona:navigator", "invalid skill name", "INVALID"},
		},
		{
			name:       "valid format but nonexistent in store",
			names:      []string{"nonexistent"},
			scope:      "global",
			store:      store,
			wantCount:  1,
			wantSubstr: []string{"global", "nonexistent", "not found in store"},
		},
		{
			name:      "multiple invalid names all reported",
			names:     []string{"BAD_ONE", "../traversal", "also bad!"},
			scope:     "global",
			store:     store,
			wantCount: 3,
		},
		{
			name:       "mixed valid and invalid only invalid reported",
			names:      []string{"golang", "INVALID", "speckit"},
			scope:      "global",
			store:      store,
			wantCount:  1,
			wantSubstr: []string{"INVALID"},
		},
		{
			name:      "store is nil only format validation runs",
			names:     []string{"nonexistent-but-valid"},
			scope:     "global",
			store:     nil,
			wantCount: 0,
		},
		{
			name:       "T020 invalid characters at scope",
			names:      []string{"my_skill", "my.skill", "MY-SKILL"},
			scope:      "persona:craftsman",
			store:      nil,
			wantCount:  3,
			wantSubstr: []string{"persona:craftsman"},
		},
		{
			name:       "T021 missing skills directory all reads fail",
			names:      []string{"golang", "speckit"},
			scope:      "global",
			store:      &mockStore{skills: map[string]Skill{}}, // empty store
			wantCount:  2,
			wantSubstr: []string{"not found in store"},
		},
		{
			name:       "T024 skill dir exists but no SKILL.md",
			names:      []string{"no-skillmd"},
			scope:      "global",
			store:      &mockStore{skills: map[string]Skill{}}, // Read returns error
			wantCount:  1,
			wantSubstr: []string{"no-skillmd", "not found in store"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := ValidateSkillRefs(tt.names, tt.scope, tt.store)
			if len(errs) != tt.wantCount {
				t.Fatalf("got %d errors, want %d: %v", len(errs), tt.wantCount, errs)
			}
			for _, substr := range tt.wantSubstr {
				found := false
				for _, err := range errs {
					if strings.Contains(err.Error(), substr) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected substring %q in errors, got: %v", substr, errs)
				}
			}
		})
	}
}

func TestValidateForPublish(t *testing.T) {
	tests := []struct {
		name       string
		skill      Skill
		wantValid  bool
		wantErrors int
		wantWarns  int
		errField   string
		warnField  string
	}{
		{
			name: "fully valid skill",
			skill: Skill{
				Name:          "golang",
				Description:   "Go development",
				License:       "MIT",
				Compatibility: "Claude Code",
				AllowedTools:  []string{"Read", "Write"},
			},
			wantValid:  true,
			wantErrors: 0,
			wantWarns:  0,
		},
		{
			name: "missing name",
			skill: Skill{
				Description: "test",
			},
			wantValid:  false,
			wantErrors: 1,
			errField:   "name",
		},
		{
			name: "missing description",
			skill: Skill{
				Name: "test",
			},
			wantValid:  false,
			wantErrors: 1,
			errField:   "description",
		},
		{
			name: "invalid name format",
			skill: Skill{
				Name:        "INVALID_NAME",
				Description: "test",
			},
			wantValid:  false,
			wantErrors: 1,
			errField:   "name",
		},
		{
			name: "missing license is warning",
			skill: Skill{
				Name:          "test",
				Description:   "test",
				Compatibility: "all",
				AllowedTools:  []string{"Read"},
			},
			wantValid: true,
			wantWarns: 1,
			warnField: "license",
		},
		{
			name: "missing compatibility is warning",
			skill: Skill{
				Name:         "test",
				Description:  "test",
				License:      "MIT",
				AllowedTools: []string{"Read"},
			},
			wantValid: true,
			wantWarns: 1,
			warnField: "compatibility",
		},
		{
			name: "missing allowed-tools is warning",
			skill: Skill{
				Name:          "test",
				Description:   "test",
				License:       "MIT",
				Compatibility: "all",
			},
			wantValid: true,
			wantWarns: 1,
			warnField: "allowed-tools",
		},
		{
			name:       "multiple issues collected",
			skill:      Skill{},
			wantValid:  false,
			wantErrors: 2, // name + description
			wantWarns:  3, // license + compatibility + allowed-tools
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := ValidateForPublish(tt.skill)
			if report.Valid() != tt.wantValid {
				t.Errorf("Valid() = %v, want %v", report.Valid(), tt.wantValid)
			}
			if tt.wantErrors > 0 && len(report.Errors) != tt.wantErrors {
				t.Errorf("got %d errors, want %d: %v", len(report.Errors), tt.wantErrors, report.Errors)
			}
			if tt.wantWarns > 0 && len(report.Warnings) != tt.wantWarns {
				t.Errorf("got %d warnings, want %d: %v", len(report.Warnings), tt.wantWarns, report.Warnings)
			}
			if tt.errField != "" {
				found := false
				for _, e := range report.Errors {
					if e.Field == tt.errField {
						found = true
					}
				}
				if !found {
					t.Errorf("expected error for field %q, got: %v", tt.errField, report.Errors)
				}
			}
			if tt.warnField != "" {
				found := false
				for _, w := range report.Warnings {
					if w.Field == tt.warnField {
						found = true
					}
				}
				if !found {
					t.Errorf("expected warning for field %q, got: %v", tt.warnField, report.Warnings)
				}
			}
		})
	}
}

func TestValidateManifestSkills(t *testing.T) {
	store := &mockStore{
		skills: map[string]Skill{
			"golang":  {Name: "golang", Description: "Go skill"},
			"speckit": {Name: "speckit", Description: "Speckit skill"},
		},
	}

	tests := []struct {
		name         string
		globalSkills []string
		personas     []PersonaSkills
		store        Store
		wantCount    int
		wantSubstr   []string
	}{
		{
			name:         "all valid",
			globalSkills: []string{"golang"},
			personas: []PersonaSkills{
				{Name: "navigator", Skills: []string{"speckit"}},
			},
			store:     store,
			wantCount: 0,
		},
		{
			name:         "invalid global skill",
			globalSkills: []string{"INVALID"},
			personas:     nil,
			store:        nil,
			wantCount:    1,
			wantSubstr:   []string{"global", "INVALID"},
		},
		{
			name:         "invalid persona skill",
			globalSkills: nil,
			personas: []PersonaSkills{
				{Name: "craftsman", Skills: []string{"BAD"}},
			},
			store:      nil,
			wantCount:  1,
			wantSubstr: []string{"persona:craftsman", "BAD"},
		},
		{
			name:         "errors from multiple scopes aggregated",
			globalSkills: []string{"GLOBAL-BAD"},
			personas: []PersonaSkills{
				{Name: "navigator", Skills: []string{"NAV-BAD"}},
				{Name: "craftsman", Skills: []string{"CRAFT-BAD"}},
			},
			store:      nil,
			wantCount:  3,
			wantSubstr: []string{"global", "persona:navigator", "persona:craftsman"},
		},
		{
			name:         "nonexistent in store across scopes",
			globalSkills: []string{"missing-global"},
			personas: []PersonaSkills{
				{Name: "navigator", Skills: []string{"missing-nav"}},
			},
			store:      store,
			wantCount:  2,
			wantSubstr: []string{"global", "persona:navigator", "not found in store"},
		},
		{
			name:         "empty everything no errors",
			globalSkills: nil,
			personas:     nil,
			store:        store,
			wantCount:    0,
		},
		{
			name:         "nil store skips existence checks",
			globalSkills: []string{"golang", "nonexistent-but-valid"},
			personas: []PersonaSkills{
				{Name: "navigator", Skills: []string{"speckit", "also-valid"}},
			},
			store:     nil,
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := ValidateManifestSkills(tt.globalSkills, tt.personas, tt.store)
			if len(errs) != tt.wantCount {
				t.Fatalf("got %d errors, want %d: %v", len(errs), tt.wantCount, errs)
			}
			for _, substr := range tt.wantSubstr {
				found := false
				for _, err := range errs {
					if strings.Contains(err.Error(), substr) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected substring %q in errors, got: %v", substr, errs)
				}
			}
		})
	}
}
