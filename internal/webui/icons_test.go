package webui

import (
	"strings"
	"testing"
)

func TestAdapterIcon(t *testing.T) {
	known := []string{"claude-code", "gemini", "codex", "opencode", "browser", "mock"}
	for _, name := range known {
		t.Run(name, func(t *testing.T) {
			got := adapterIcon(name)
			if got == "" {
				t.Errorf("adapterIcon(%q) returned empty, want non-empty SVG", name)
			}
			if !strings.Contains(string(got), "currentColor") {
				t.Errorf("adapterIcon(%q) SVG missing currentColor", name)
			}
			if !strings.Contains(string(got), "icon-inline") {
				t.Errorf("adapterIcon(%q) SVG missing icon-inline class", name)
			}
			if !strings.Contains(string(got), "aria-hidden") {
				t.Errorf("adapterIcon(%q) SVG missing aria-hidden attribute", name)
			}
		})
	}
}

func TestAdapterIconUnknown(t *testing.T) {
	got := adapterIcon("nonexistent-adapter")
	if got != "" {
		t.Errorf("adapterIcon(%q) = %q, want empty string", "nonexistent-adapter", got)
	}
}

func TestForgeIcon(t *testing.T) {
	known := []string{"github", "gitlab", "bitbucket", "gitea", "codeberg", "forgejo"}
	for _, name := range known {
		t.Run(name, func(t *testing.T) {
			got := forgeIcon(name)
			if got == "" {
				t.Errorf("forgeIcon(%q) returned empty, want non-empty SVG", name)
			}
			if !strings.Contains(string(got), "currentColor") {
				t.Errorf("forgeIcon(%q) SVG missing currentColor", name)
			}
			if !strings.Contains(string(got), "icon-inline") {
				t.Errorf("forgeIcon(%q) SVG missing icon-inline class", name)
			}
			if !strings.Contains(string(got), "aria-hidden") {
				t.Errorf("forgeIcon(%q) SVG missing aria-hidden attribute", name)
			}
		})
	}
}

func TestForgeIconUnknown(t *testing.T) {
	got := forgeIcon("nonexistent-forge")
	if got != "" {
		t.Errorf("forgeIcon(%q) = %q, want empty string", "nonexistent-forge", got)
	}
}
