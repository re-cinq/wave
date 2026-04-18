package skill

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
)

// --- T001: Mock store and test helpers ---

type memoryStore struct {
	skills   map[string]Skill
	writes   int
	writeLog []string
}

func newMemoryStore() *memoryStore {
	return &memoryStore{skills: make(map[string]Skill)}
}

func (m *memoryStore) Read(name string) (Skill, error) {
	s, ok := m.skills[name]
	if !ok {
		return Skill{}, fmt.Errorf("%w: %s", ErrNotFound, name)
	}
	return s, nil
}

func (m *memoryStore) Write(skill Skill) error {
	m.skills[skill.Name] = skill
	m.writes++
	m.writeLog = append(m.writeLog, skill.Name)
	return nil
}

func (m *memoryStore) List() ([]Skill, error) {
	var result []Skill
	for _, s := range m.skills {
		result = append(result, s)
	}
	return result, nil
}

func (m *memoryStore) Delete(name string) error {
	if _, ok := m.skills[name]; !ok {
		return fmt.Errorf("%w: %s", ErrNotFound, name)
	}
	delete(m.skills, name)
	return nil
}

func makeTestSkillDir(t *testing.T, dir, name, description string) {
	t.Helper()
	createSkillDir(t, dir, name, description)
}

// --- stubAdapter for router testing ---

type stubAdapter struct {
	prefix string
	calls  []string
	result *InstallResult
	err    error
}

func (s *stubAdapter) Prefix() string { return s.prefix }

func (s *stubAdapter) Install(_ context.Context, ref string, _ Store) (*InstallResult, error) {
	s.calls = append(s.calls, ref)
	if s.err != nil {
		return nil, s.err
	}
	if s.result != nil {
		return s.result, nil
	}
	return &InstallResult{}, nil
}

// --- T004: Router unit tests ---

func TestSourceRouterParse(t *testing.T) {
	tesslAdapter := &stubAdapter{prefix: "tessl"}
	githubAdapter := &stubAdapter{prefix: "github"}
	fileAdapter := &stubAdapter{prefix: "file"}
	bmadAdapter := &stubAdapter{prefix: "bmad"}
	openspecAdapter := &stubAdapter{prefix: "openspec"}
	speckitAdapter := &stubAdapter{prefix: "speckit"}
	urlAdapter := &stubAdapter{prefix: "https://"}

	router := NewSourceRouter(tesslAdapter, githubAdapter, fileAdapter, bmadAdapter, openspecAdapter, speckitAdapter, urlAdapter)

	tests := []struct {
		name        string
		source      string
		wantPrefix  string
		wantRef     string
		wantErr     bool
		errContains string
	}{
		{
			name:       "tessl prefix",
			source:     "tessl:github/spec-kit",
			wantPrefix: "tessl",
			wantRef:    "github/spec-kit",
		},
		{
			name:       "github prefix",
			source:     "github:owner/repo",
			wantPrefix: "github",
			wantRef:    "owner/repo",
		},
		{
			name:       "file prefix",
			source:     "file:./local/path",
			wantPrefix: "file",
			wantRef:    "./local/path",
		},
		{
			name:       "bmad prefix",
			source:     "bmad:install",
			wantPrefix: "bmad",
			wantRef:    "install",
		},
		{
			name:       "openspec prefix",
			source:     "openspec:init",
			wantPrefix: "openspec",
			wantRef:    "init",
		},
		{
			name:       "speckit prefix",
			source:     "speckit:init",
			wantPrefix: "speckit",
			wantRef:    "init",
		},
		{
			name:       "https URL",
			source:     "https://example.com/skill.tar.gz",
			wantPrefix: "https://",
			wantRef:    "https://example.com/skill.tar.gz",
		},
		{
			name:        "http URL rejected",
			source:      "http://example.com/skill.tar.gz",
			wantErr:     true,
			errContains: "only HTTPS",
		},
		{
			name:        "unknown prefix",
			source:      "foobar:something",
			wantErr:     true,
			errContains: "unknown source prefix",
		},
		{
			name:        "bare name no colon",
			source:      "golang",
			wantErr:     true,
			errContains: "no source prefix",
		},
		{
			name:    "empty string",
			source:  "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter, ref, err := router.Parse(tt.source)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error %q should contain %q", err.Error(), tt.errContains)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if adapter.Prefix() != tt.wantPrefix {
				t.Errorf("prefix = %q, want %q", adapter.Prefix(), tt.wantPrefix)
			}
			if ref != tt.wantRef {
				t.Errorf("ref = %q, want %q", ref, tt.wantRef)
			}
		})
	}
}

func TestSourceRouterInstall(t *testing.T) {
	expectedResult := &InstallResult{
		Skills: []Skill{{Name: "test-skill", Description: "A test"}},
	}
	stub := &stubAdapter{
		prefix: "tessl",
		result: expectedResult,
	}
	router := NewSourceRouter(stub)

	result, err := router.Install(context.Background(), "tessl:my-ref", newMemoryStore())
	if err != nil {
		t.Fatalf("Install() error = %v", err)
	}
	if len(result.Skills) != 1 || result.Skills[0].Name != "test-skill" {
		t.Errorf("unexpected result: %+v", result)
	}
	if len(stub.calls) != 1 || stub.calls[0] != "my-ref" {
		t.Errorf("expected stub called with 'my-ref', got %v", stub.calls)
	}
}

func TestSourceRouterInstallError(t *testing.T) {
	stub := &stubAdapter{
		prefix: "tessl",
		err:    errors.New("CLI failed"),
	}
	router := NewSourceRouter(stub)

	_, err := router.Install(context.Background(), "tessl:ref", newMemoryStore())
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "CLI failed") {
		t.Errorf("error should contain 'CLI failed': %v", err)
	}
}

func TestSourceRouterPrefixes(t *testing.T) {
	router := NewSourceRouter(
		&stubAdapter{prefix: "github"},
		&stubAdapter{prefix: "bmad"},
		&stubAdapter{prefix: "tessl"},
	)

	prefixes := router.Prefixes()
	if len(prefixes) != 3 {
		t.Fatalf("expected 3 prefixes, got %d", len(prefixes))
	}
	// Should be sorted
	if prefixes[0] != "bmad" || prefixes[1] != "github" || prefixes[2] != "tessl" {
		t.Errorf("prefixes not sorted: %v", prefixes)
	}
}

// --- T017: NewDefaultRouter tests ---

func TestNewDefaultRouter(t *testing.T) {
	router := NewDefaultRouter("/tmp")
	prefixes := router.Prefixes()

	expected := []string{"file"}
	if len(prefixes) != len(expected) {
		t.Fatalf("expected %d prefixes, got %d: %v", len(expected), len(prefixes), prefixes)
	}
	for i, want := range expected {
		if prefixes[i] != want {
			t.Errorf("prefixes[%d] = %q, want %q", i, prefixes[i], want)
		}
	}
}

func TestNewDefaultRouterParsesFilePrefix(t *testing.T) {
	router := NewDefaultRouter("/tmp")
	adapter, _, err := router.Parse("file:./local")
	if err != nil {
		t.Fatalf("Parse(file:./local) error = %v", err)
	}
	if adapter.Prefix() != "file" {
		t.Errorf("prefix = %q, want file", adapter.Prefix())
	}
}

func TestDependencyErrorFormat(t *testing.T) {
	e := &DependencyError{
		Binary:       "tessl",
		Instructions: "npm i -g @tessl/cli",
	}
	msg := e.Error()
	if !strings.Contains(msg, "tessl") {
		t.Errorf("expected binary name in error, got: %s", msg)
	}
	if !strings.Contains(msg, "npm i -g @tessl/cli") {
		t.Errorf("expected install instructions in error, got: %s", msg)
	}
}

func TestSourceRouterInstallParseError(t *testing.T) {
	router := NewSourceRouter(&stubAdapter{prefix: "tessl"})

	// Invalid source string with no prefix
	_, err := router.Install(context.Background(), "no-prefix", newMemoryStore())
	if err == nil {
		t.Fatal("expected error for invalid source")
	}
}

func TestUnknownPrefixListsRecognized(t *testing.T) {
	router := NewSourceRouter(
		&stubAdapter{prefix: "tessl"},
		&stubAdapter{prefix: "github"},
	)

	_, _, err := router.Parse("foobar:something")
	if err == nil {
		t.Fatal("expected error for unknown prefix")
	}
	errStr := err.Error()
	if !strings.Contains(errStr, "github") || !strings.Contains(errStr, "tessl") {
		t.Errorf("error should list recognized prefixes, got: %s", errStr)
	}
}
