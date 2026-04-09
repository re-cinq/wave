package security

import (
	"strings"
	"testing"
)

func TestContainsUTF7Sequence(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"plain ASCII path", "/home/user/file.txt", false},
		{"UTF-7 encoded angle bracket", "+ADw-script+AD4-", true},
		{"UTF-7 encoded slash", "+AC8-etc+AC8-passwd", true},
		{"plus sign but not UTF-7", "a+b=c", false},
		{"plus followed by dash immediately", "+-", false},
		{"valid base64 in UTF-7 context", "+ACQ-HOME", true},
		{"empty string", "", false},
		{"just plus sign", "+", false},
		{"plus with non-base64", "+!@#-", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := containsUTF7Sequence(tt.input)
			if got != tt.want {
				t.Errorf("containsUTF7Sequence(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestDetectMixedScript(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"pure Latin", "/home/user/file.txt", false},
		{"pure ASCII", "simple-path-123", false},
		{"Cyrillic only", "\u0410\u0411\u0412", false},       // АБВ
		{"Latin + Cyrillic mixed", "hello\u0430world", true}, // а is Cyrillic
		{"Latin + Greek mixed", "hello\u03B1world", true},    // α is Greek
		{"CJK with Latin (not confusable)", "file\u4e2d.txt", false},
		{"empty string", "", false},
		{"numbers only", "123456", false},
		{"Latin + Arabic", "hello\u0639world", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := detectMixedScript(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("detectMixedScript(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if err != nil && !strings.Contains(err.Error(), "homograph") {
				t.Errorf("error should mention homograph, got: %v", err)
			}
		})
	}
}

func TestRuneScript(t *testing.T) {
	tests := []struct {
		name string
		r    rune
		want dominantScript
	}{
		{"ASCII letter", 'a', scriptLatin},
		{"ASCII digit", '0', scriptLatin},
		{"Latin extended", '\u00e9', scriptLatin}, // é
		{"Cyrillic а", '\u0430', scriptCyrillic},
		{"Greek alpha", '\u03B1', scriptGreek},
		{"Arabic ain", '\u0639', scriptArabic},
		{"Hebrew alef", '\u05D0', scriptHebrew},
		{"CJK character", '\u4e2d', scriptOther},
		{"ASCII slash", '/', scriptLatin},
		{"ASCII dot", '.', scriptLatin},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := runeScript(tt.r)
			if got != tt.want {
				t.Errorf("runeScript(%q) = %v, want %v", tt.r, got, tt.want)
			}
		})
	}
}

func TestValidateUnicode_Integration(t *testing.T) {
	config := DefaultSecurityConfig()
	logger := NewSecurityLogger(false)
	validator := NewPathValidator(*config, logger)

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{"clean path", "src/main.go", false},
		{"UTF-7 attack", "+ADw-script+AD4-alert(1)", true},
		{"Cyrillic homograph", "/home/\u0430dmin/file", true}, // Cyrillic а + Latin dmin
		{"pure CJK path (allowed)", "\u4e2d\u6587\u8def\u5f84", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validateUnicode(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateUnicode(%q) error = %v, wantErr %v", tt.path, err, tt.wantErr)
			}
		})
	}
}

func TestSchemaCache(t *testing.T) {
	// Test store and retrieve
	path := "/test/schema.json"
	content := `{"type": "object"}`

	SetCachedSchemaContent(path, content)

	got, ok := GetCachedSchemaContent(path)
	if !ok {
		t.Error("expected cache hit")
	}
	if got != content {
		t.Errorf("cached content = %q, want %q", got, content)
	}

	// Test cache miss
	_, ok = GetCachedSchemaContent("/nonexistent/path")
	if ok {
		t.Error("expected cache miss for non-existent path")
	}

	// Test overwrite
	newContent := `{"type": "array"}`
	SetCachedSchemaContent(path, newContent)

	got, ok = GetCachedSchemaContent(path)
	if !ok {
		t.Error("expected cache hit after overwrite")
	}
	if got != newContent {
		t.Errorf("cached content after overwrite = %q, want %q", got, newContent)
	}
}

func TestIsBase64Char(t *testing.T) {
	tests := []struct {
		c    byte
		want bool
	}{
		{'A', true},
		{'Z', true},
		{'a', true},
		{'z', true},
		{'0', true},
		{'9', true},
		{'+', true},
		{'/', true},
		{'-', false},
		{'!', false},
		{' ', false},
		{'.', false},
	}

	for _, tt := range tests {
		t.Run(string(tt.c), func(t *testing.T) {
			got := isBase64Char(tt.c)
			if got != tt.want {
				t.Errorf("isBase64Char(%q) = %v, want %v", tt.c, got, tt.want)
			}
		})
	}
}
