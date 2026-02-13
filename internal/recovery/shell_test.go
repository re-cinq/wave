package recovery

import "testing"

func TestShellEscape(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty string", "", "''"},
		{"simple string", "hello", "hello"},
		{"string with hyphen", "--from-step", "--from-step"},
		{"string with slash", "path/to/file", "path/to/file"},
		{"string with spaces", "add auth", "'add auth'"},
		{"single quotes", "it's here", "'it'\\''s here'"},
		{"double quotes", `say "hello"`, `'say "hello"'`},
		{"ampersand", "foo & bar", "'foo & bar'"},
		{"semicolon", "foo; bar", "'foo; bar'"},
		{"backticks", "foo `bar`", "'foo `bar`'"},
		{"dollar sign", "foo $bar", "'foo $bar'"},
		{"glob characters", "*.go", "'*.go'"},
		{"newline", "foo\nbar", "'foo\nbar'"},
		{"multiple single quotes", "it's it's", "'it'\\''s it'\\''s'"},
		{"pipeline name", "feature", "feature"},
		{"step id with hyphen", "implement-code", "implement-code"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ShellEscape(tt.input)
			if got != tt.expected {
				t.Errorf("ShellEscape(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}
