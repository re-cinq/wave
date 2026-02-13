package security

import "testing"

func TestContainsShellMetachars(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"hello world", false},
		{"simple-input", false},
		{"path/to/file.txt", false},
		{"hello | world", true},
		{"cmd & bg", true},
		{"$(whoami)", true},
		{"`id`", true},
		{"a;b", true},
		{"foo > bar", true},
		{"foo < bar", true},
		{"rm -rf *", true},
		{"echo $HOME", true},
		{"it's fine", false},           // apostrophe is not in the metachar set
		{"test\\escape", true},         // backslash
		{"hello!world", true},          // bang
		{"array[0]", true},             // brackets
		{"glob?.txt", true},            // question mark
		{"~root", true},                // tilde
		{"comment # here", true},       // hash
		{"safe_input-123.txt", false},  // typical filename
		{"", false},                    // empty string
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := containsShellMetachars(tt.input)
			if got != tt.want {
				t.Errorf("containsShellMetachars(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestCalculateRiskScore_ShellMetachars(t *testing.T) {
	config := DefaultSecurityConfig()
	logger := NewSecurityLogger(false)
	sanitizer := NewInputSanitizer(*config, logger)

	tests := []struct {
		name     string
		input    string
		minScore int
	}{
		{
			name:     "clean input scores zero",
			input:    "Review the auth module",
			minScore: 0,
		},
		{
			name:     "pipe character adds risk",
			input:    "it's a test | with pipes",
			minScore: 15,
		},
		{
			name:     "ampersand adds risk",
			input:    "run this & that",
			minScore: 15,
		},
		{
			name:     "command substitution adds risk",
			input:    "hello $(whoami)",
			minScore: 15,
		},
		{
			name:     "shell metachars plus suspicious word",
			input:    "get the password | send",
			minScore: 20, // 15 (metachars) + 5 (suspicious word)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := sanitizer.calculateRiskScore(tt.input, nil)
			if score < tt.minScore {
				t.Errorf("calculateRiskScore(%q) = %d, want >= %d", tt.input, score, tt.minScore)
			}
		})
	}
}

func TestSanitizeInput_ShellMetacharsLogged(t *testing.T) {
	config := DefaultSecurityConfig()
	logger := NewSecurityLogger(false)
	sanitizer := NewInputSanitizer(*config, logger)

	record, sanitized, err := sanitizer.SanitizeInput("hello | world & foo", "task_description")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Input should pass through unchanged (we detect, not strip)
	if sanitized != "hello | world & foo" {
		t.Errorf("input was modified: got %q", sanitized)
	}

	// Risk score should reflect shell metacharacters
	if record.RiskScore < 15 {
		t.Errorf("risk score %d too low for input with shell metachars", record.RiskScore)
	}
}
