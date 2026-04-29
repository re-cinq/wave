package persona

import "testing"

// TestPersonaZeroValue ensures the zero value carries empty fields as expected.
func TestPersonaZeroValue(t *testing.T) {
	var p Persona
	if p.Model != "" {
		t.Errorf("zero Model = %q, want empty", p.Model)
	}
	if len(p.AllowedTools) != 0 {
		t.Errorf("zero AllowedTools len = %d, want 0", len(p.AllowedTools))
	}
	if len(p.DenyTools) != 0 {
		t.Errorf("zero DenyTools len = %d, want 0", len(p.DenyTools))
	}
}

// TestPersonaFieldRoundtrip pins the field shape consumed by the agent
// compiler. Adding fields here is fine; renaming or removing them is a
// behavioural change that callers in internal/adapter and cmd/wave/commands
// depend on.
func TestPersonaFieldRoundtrip(t *testing.T) {
	p := Persona{
		Model:        "claude-opus-4",
		AllowedTools: []string{"Read", "Glob"},
		DenyTools:    []string{"Bash(rm*)"},
	}
	if p.Model != "claude-opus-4" {
		t.Errorf("Model = %q", p.Model)
	}
	if len(p.AllowedTools) != 2 || p.AllowedTools[0] != "Read" {
		t.Errorf("AllowedTools = %v", p.AllowedTools)
	}
	if len(p.DenyTools) != 1 || p.DenyTools[0] != "Bash(rm*)" {
		t.Errorf("DenyTools = %v", p.DenyTools)
	}
}
