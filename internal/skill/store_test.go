package skill

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// testSkillFixtures provides inline SKILL.md content for hermetic testing.
// These replace the old repoRoot()-based tests that read from the working tree.
var testSkillFixtures = map[string]string{
	"minimal": `---
name: minimal-skill
description: A minimal test skill
---

# Minimal Skill

Does nothing.
`,
	"full": `---
name: full-skill
description: A comprehensive test skill with all fields
license: MIT
version: "1.0"
compatibility: ">=2.0"
allowed_tools:
  - Bash(go test)
  - Read
  - Grep
---

# Full Skill

This skill has all metadata fields populated.

## Instructions

Do the thing.
`,
	"multi-tool": `---
name: multi-tool-skill
description: Skill with multiple allowed tools
allowed_tools:
  - Bash(npm test)
  - Bash(npm run build)
  - Write
  - Edit
  - Glob
---

# Multi-Tool Skill

Has several allowed tools for testing serialization round-trips.
`,
}

// --- T006: Parser Unit Tests ---

func TestValidateName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid simple", "golang", false},
		{"valid hyphenated", "my-skill", false},
		{"valid single char", "a", false},
		{"valid multi hyphen", "a-b-c", false},
		{"valid 64 chars", strings.Repeat("a", 64), false},
		{"valid alphanumeric", "skill1", false},
		{"valid starts with number", "1skill", false},
		{"invalid empty", "", true},
		{"invalid uppercase", "MySkill", true},
		{"invalid dots", "my.skill", true},
		{"invalid underscores", "my_skill", true},
		{"invalid path traversal", "../etc", true},
		{"invalid slash", "foo/bar", true},
		{"invalid 65 chars", strings.Repeat("a", 65), true},
		{"invalid ends with hyphen", "foo-", true},
		{"invalid starts with hyphen", "-foo", true},
		{"invalid space", "my skill", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateName(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if err != nil {
				var pe *ParseError
				if !errors.As(err, &pe) {
					t.Errorf("expected *ParseError, got %T", err)
				}
			}
		})
	}
}

func TestParse(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantErr    bool
		wantName   string
		wantDesc   string
		wantBody   string
		checkTools []string
	}{
		{
			name: "valid with all fields",
			input: `---
name: my-skill
description: A test skill
license: MIT
compatibility: Claude 4.x
metadata:
  author: test
  version: "1.0"
allowed-tools: "Read Write Edit"
---
# Hello

This is the body.
`,
			wantName:   "my-skill",
			wantDesc:   "A test skill",
			wantBody:   "# Hello\n\nThis is the body.\n",
			checkTools: []string{"Read", "Write", "Edit"},
		},
		{
			name: "valid with only required fields",
			input: `---
name: minimal
description: A minimal skill
---
Body here.
`,
			wantName: "minimal",
			wantDesc: "A minimal skill",
			wantBody: "Body here.\n",
		},
		{
			name: "valid with empty body",
			input: `---
name: no-body
description: No body content
---
`,
			wantName: "no-body",
			wantDesc: "No body content",
			wantBody: "",
		},
		{
			name: "missing name",
			input: `---
description: No name here
---
body
`,
			wantErr: true,
		},
		{
			name: "missing description",
			input: `---
name: missing-desc
---
body
`,
			wantErr: true,
		},
		{
			name:    "no frontmatter delimiters",
			input:   "Just some text without frontmatter",
			wantErr: true,
		},
		{
			name: "invalid name uppercase",
			input: `---
name: MySkill
description: Bad name
---
`,
			wantErr: true,
		},
		{
			name: "invalid name dots",
			input: `---
name: my.skill
description: Bad name
---
`,
			wantErr: true,
		},
		{
			name: "invalid name path traversal",
			input: `---
name: ../etc
description: Bad name
---
`,
			wantErr: true,
		},
		{
			name:    "empty file",
			input:   "",
			wantErr: true,
		},
		{
			name: "malformed yaml",
			input: `---
name: [invalid yaml
description: test
---
`,
			wantErr: true,
		},
		{
			name:  "compatibility too long",
			input: "---\nname: test\ndescription: test\ncompatibility: " + strings.Repeat("x", 501) + "\n---\n",

			wantErr: true,
		},
		{
			name: "single allowed tool",
			input: `---
name: one-tool
description: Single tool
allowed-tools: Read
---
`,
			wantName:   "one-tool",
			wantDesc:   "Single tool",
			checkTools: []string{"Read"},
		},
		{
			name: "empty allowed tools",
			input: `---
name: no-tools
description: No tools
---
`,
			wantName: "no-tools",
			wantDesc: "No tools",
		},
		{
			name: "metadata map",
			input: `---
name: with-meta
description: Has metadata
metadata:
  key1: value1
  key2: value2
---
`,
			wantName: "with-meta",
			wantDesc: "Has metadata",
		},
		{
			name: "unterminated frontmatter",
			input: `---
name: broken
description: missing closing delimiter
`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			skill, err := Parse([]byte(tt.input))
			if (err != nil) != tt.wantErr {
				t.Fatalf("Parse() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if skill.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", skill.Name, tt.wantName)
			}
			if skill.Description != tt.wantDesc {
				t.Errorf("Description = %q, want %q", skill.Description, tt.wantDesc)
			}
			if skill.Body != tt.wantBody {
				t.Errorf("Body = %q, want %q", skill.Body, tt.wantBody)
			}
			if tt.checkTools != nil {
				if len(skill.AllowedTools) != len(tt.checkTools) {
					t.Errorf("AllowedTools = %v, want %v", skill.AllowedTools, tt.checkTools)
				} else {
					for i, tool := range tt.checkTools {
						if skill.AllowedTools[i] != tool {
							t.Errorf("AllowedTools[%d] = %q, want %q", i, skill.AllowedTools[i], tool)
						}
					}
				}
			}
		})
	}
}

func TestParseMetadata(t *testing.T) {
	input := `---
name: meta-only
description: Metadata loading test
license: MIT
---
# Body content that should be discarded
This is a long body.
`
	skill, err := ParseMetadata([]byte(input))
	if err != nil {
		t.Fatalf("ParseMetadata() error = %v", err)
	}
	if skill.Name != "meta-only" {
		t.Errorf("Name = %q, want %q", skill.Name, "meta-only")
	}
	if skill.Description != "Metadata loading test" {
		t.Errorf("Description = %q, want %q", skill.Description, "Metadata loading test")
	}
	if skill.License != "MIT" {
		t.Errorf("License = %q, want %q", skill.License, "MIT")
	}
	if skill.Body != "" {
		t.Errorf("Body should be empty, got %q", skill.Body)
	}

	// Verify same validation as Parse
	_, err = ParseMetadata([]byte(`---
description: no name
---
`))
	if err == nil {
		t.Error("ParseMetadata should fail for missing name")
	}
}

func TestSerialize(t *testing.T) {
	t.Run("round-trip fidelity", func(t *testing.T) {
		original := Skill{
			Name:          "round-trip",
			Description:   "Test round-trip",
			License:       "MIT",
			Compatibility: "Claude 4.x",
			Metadata:      map[string]string{"author": "test"},
			AllowedTools:  []string{"Read", "Write"},
			Body:          "# Hello\n\nBody content here.\n",
		}

		data, err := Serialize(original)
		if err != nil {
			t.Fatalf("Serialize() error = %v", err)
		}

		parsed, err := Parse(data)
		if err != nil {
			t.Fatalf("Parse(Serialize()) error = %v", err)
		}

		if parsed.Name != original.Name {
			t.Errorf("Name = %q, want %q", parsed.Name, original.Name)
		}
		if parsed.Description != original.Description {
			t.Errorf("Description = %q, want %q", parsed.Description, original.Description)
		}
		if parsed.License != original.License {
			t.Errorf("License = %q, want %q", parsed.License, original.License)
		}
		if parsed.Compatibility != original.Compatibility {
			t.Errorf("Compatibility = %q, want %q", parsed.Compatibility, original.Compatibility)
		}
		if parsed.Body != original.Body {
			t.Errorf("Body = %q, want %q", parsed.Body, original.Body)
		}
		if len(parsed.AllowedTools) != len(original.AllowedTools) {
			t.Errorf("AllowedTools = %v, want %v", parsed.AllowedTools, original.AllowedTools)
		}
		if parsed.Metadata["author"] != "test" {
			t.Errorf("Metadata[author] = %q, want %q", parsed.Metadata["author"], "test")
		}
	})

	t.Run("validation before serialization", func(t *testing.T) {
		_, err := Serialize(Skill{Name: "", Description: "valid"})
		if err == nil {
			t.Error("expected error for empty name")
		}

		_, err = Serialize(Skill{Name: "valid", Description: ""})
		if err == nil {
			t.Error("expected error for empty description")
		}

		_, err = Serialize(Skill{Name: "INVALID", Description: "valid"})
		if err == nil {
			t.Error("expected error for invalid name")
		}
	})

	t.Run("empty body", func(t *testing.T) {
		data, err := Serialize(Skill{Name: "no-body", Description: "Test"})
		if err != nil {
			t.Fatalf("Serialize() error = %v", err)
		}

		parsed, err := Parse(data)
		if err != nil {
			t.Fatalf("Parse() error = %v", err)
		}
		if parsed.Body != "" {
			t.Errorf("Body = %q, want empty", parsed.Body)
		}
	})
}

// --- T012: Store CRUD and Multi-Source Tests ---

func createSkillDir(t *testing.T, root, name, description string) {
	t.Helper()
	dir := filepath.Join(root, name)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	content := "---\nname: " + name + "\ndescription: " + description + "\n---\n# " + name + "\n"
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestDirectoryStoreRead(t *testing.T) {
	t.Run("existing skill", func(t *testing.T) {
		root := t.TempDir()
		createSkillDir(t, root, "test-skill", "A test skill")

		store := NewDirectoryStore(SkillSource{Root: root, Precedence: 1})
		skill, err := store.Read("test-skill")
		if err != nil {
			t.Fatalf("Read() error = %v", err)
		}
		if skill.Name != "test-skill" {
			t.Errorf("Name = %q, want %q", skill.Name, "test-skill")
		}
		if skill.Description != "A test skill" {
			t.Errorf("Description = %q, want %q", skill.Description, "A test skill")
		}
		if skill.SourcePath != filepath.Join(root, "test-skill") {
			t.Errorf("SourcePath = %q, want %q", skill.SourcePath, filepath.Join(root, "test-skill"))
		}
	})

	t.Run("non-existent skill", func(t *testing.T) {
		root := t.TempDir()
		store := NewDirectoryStore(SkillSource{Root: root, Precedence: 1})
		_, err := store.Read("nonexistent")
		if err == nil {
			t.Fatal("expected error for non-existent skill")
		}
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("expected ErrNotFound, got %v", err)
		}
	})

	t.Run("with resource files", func(t *testing.T) {
		root := t.TempDir()
		createSkillDir(t, root, "resourced", "Has resources")

		skillDir := filepath.Join(root, "resourced")
		for _, sub := range []string{"scripts", "references", "assets"} {
			if err := os.MkdirAll(filepath.Join(skillDir, sub), 0755); err != nil {
				t.Fatal(err)
			}
		}
		for _, f := range []struct{ path, content string }{
			{filepath.Join(skillDir, "scripts", "setup.sh"), "#!/bin/bash"},
			{filepath.Join(skillDir, "references", "api.json"), "{}"},
			{filepath.Join(skillDir, "assets", "template.txt"), "template"},
		} {
			if err := os.WriteFile(f.path, []byte(f.content), 0644); err != nil {
				t.Fatal(err)
			}
		}

		store := NewDirectoryStore(SkillSource{Root: root, Precedence: 1})
		skill, err := store.Read("resourced")
		if err != nil {
			t.Fatalf("Read() error = %v", err)
		}
		if len(skill.ResourcePaths) != 3 {
			t.Errorf("ResourcePaths count = %d, want 3, got %v", len(skill.ResourcePaths), skill.ResourcePaths)
		}
	})

	t.Run("path traversal rejected", func(t *testing.T) {
		root := t.TempDir()
		store := NewDirectoryStore(SkillSource{Root: root, Precedence: 1})
		_, err := store.Read("../etc")
		if err == nil {
			t.Fatal("expected error for path traversal")
		}
	})

	t.Run("symlink rejected", func(t *testing.T) {
		root := t.TempDir()
		target := t.TempDir()
		createSkillDir(t, target, "real-skill", "Real skill")

		// Create a symlink: root/symlinked -> target/real-skill
		if err := os.Symlink(filepath.Join(target, "real-skill"), filepath.Join(root, "symlinked")); err != nil {
			t.Skip("symlinks not supported")
		}

		store := NewDirectoryStore(SkillSource{Root: root, Precedence: 1})
		_, err := store.Read("symlinked")
		if err == nil {
			t.Fatal("expected error for symlink skill directory")
		}
		if !strings.Contains(err.Error(), "symlink rejected") {
			t.Errorf("expected symlink rejection error, got: %v", err)
		}
	})

	t.Run("name directory mismatch", func(t *testing.T) {
		root := t.TempDir()
		dir := filepath.Join(root, "dir-name")
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		// Write a SKILL.md where name doesn't match directory
		content := "---\nname: different-name\ndescription: Mismatch test\n---\n"
		if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		store := NewDirectoryStore(SkillSource{Root: root, Precedence: 1})
		_, err := store.Read("dir-name")
		if err == nil {
			t.Fatal("expected error for name/directory mismatch")
		}
		var pe *ParseError
		if !errors.As(err, &pe) || pe.Constraint != "must match directory name" {
			t.Errorf("expected name mismatch error, got %v", err)
		}
	})
}

func TestDirectoryStoreList(t *testing.T) {
	t.Run("multiple valid skills", func(t *testing.T) {
		root := t.TempDir()
		createSkillDir(t, root, "skill-a", "Skill A")
		createSkillDir(t, root, "skill-b", "Skill B")
		createSkillDir(t, root, "skill-c", "Skill C")

		store := NewDirectoryStore(SkillSource{Root: root, Precedence: 1})
		skills, err := store.List()
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}
		if len(skills) != 3 {
			t.Errorf("got %d skills, want 3", len(skills))
		}
	})

	t.Run("mix valid and invalid returns DiscoveryError", func(t *testing.T) {
		root := t.TempDir()
		createSkillDir(t, root, "valid-skill", "Valid one")

		// Create invalid skill (malformed YAML)
		badDir := filepath.Join(root, "bad-skill")
		if err := os.MkdirAll(badDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(badDir, "SKILL.md"), []byte("---\n[invalid yaml\n---\n"), 0644); err != nil {
			t.Fatal(err)
		}

		store := NewDirectoryStore(SkillSource{Root: root, Precedence: 1})
		skills, err := store.List()
		if err == nil {
			t.Fatal("expected DiscoveryError for invalid skills")
		}
		var de *DiscoveryError
		if !errors.As(err, &de) {
			t.Fatalf("expected *DiscoveryError, got %T", err)
		}
		if len(de.Errors) != 1 {
			t.Errorf("expected 1 error, got %d", len(de.Errors))
		}
		if len(skills) != 1 {
			t.Errorf("expected 1 valid skill, got %d", len(skills))
		}
	})

	t.Run("empty directory returns empty list", func(t *testing.T) {
		root := t.TempDir()
		store := NewDirectoryStore(SkillSource{Root: root, Precedence: 1})
		skills, err := store.List()
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}
		if len(skills) != 0 {
			t.Errorf("expected 0 skills, got %d", len(skills))
		}
	})

	t.Run("non-existent source skipped silently", func(t *testing.T) {
		store := NewDirectoryStore(SkillSource{Root: "/nonexistent/path", Precedence: 1})
		skills, err := store.List()
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}
		if len(skills) != 0 {
			t.Errorf("expected 0 skills, got %d", len(skills))
		}
	})
}

func TestDirectoryStoreWrite(t *testing.T) {
	t.Run("valid skill creates directory and file", func(t *testing.T) {
		root := t.TempDir()
		store := NewDirectoryStore(SkillSource{Root: root, Precedence: 1})

		err := store.Write(Skill{
			Name:        "new-skill",
			Description: "A new skill",
			Body:        "# Instructions\n",
		})
		if err != nil {
			t.Fatalf("Write() error = %v", err)
		}

		// Verify file exists and is valid
		data, err := os.ReadFile(filepath.Join(root, "new-skill", "SKILL.md"))
		if err != nil {
			t.Fatalf("failed to read written file: %v", err)
		}

		skill, err := Parse(data)
		if err != nil {
			t.Fatalf("written file fails to parse: %v", err)
		}
		if skill.Name != "new-skill" {
			t.Errorf("Name = %q, want %q", skill.Name, "new-skill")
		}
	})

	t.Run("overwrite existing", func(t *testing.T) {
		root := t.TempDir()
		store := NewDirectoryStore(SkillSource{Root: root, Precedence: 1})

		if err := store.Write(Skill{Name: "overwrite", Description: "First version", Body: "v1"}); err != nil {
			t.Fatal(err)
		}
		if err := store.Write(Skill{Name: "overwrite", Description: "Second version", Body: "v2"}); err != nil {
			t.Fatal(err)
		}

		skill, err := store.Read("overwrite")
		if err != nil {
			t.Fatalf("Read() error = %v", err)
		}
		if skill.Description != "Second version" {
			t.Errorf("Description = %q, want %q", skill.Description, "Second version")
		}
	})

	t.Run("invalid name rejected", func(t *testing.T) {
		root := t.TempDir()
		store := NewDirectoryStore(SkillSource{Root: root, Precedence: 1})
		err := store.Write(Skill{Name: "INVALID", Description: "test"})
		if err == nil {
			t.Fatal("expected error for invalid name")
		}
	})

	t.Run("empty description rejected", func(t *testing.T) {
		root := t.TempDir()
		store := NewDirectoryStore(SkillSource{Root: root, Precedence: 1})
		err := store.Write(Skill{Name: "valid-name", Description: ""})
		if err == nil {
			t.Fatal("expected error for empty description")
		}
	})

	t.Run("path traversal rejected", func(t *testing.T) {
		root := t.TempDir()
		store := NewDirectoryStore(SkillSource{Root: root, Precedence: 1})
		err := store.Write(Skill{Name: "../malicious", Description: "test"})
		if err == nil {
			t.Fatal("expected error for path traversal")
		}
	})

	t.Run("empty sources returns error", func(t *testing.T) {
		store := NewDirectoryStore()
		err := store.Write(Skill{Name: "test", Description: "test"})
		if err == nil {
			t.Fatal("expected error for empty sources")
		}
	})
}

func TestDirectoryStoreDelete(t *testing.T) {
	t.Run("existing skill removed", func(t *testing.T) {
		root := t.TempDir()
		createSkillDir(t, root, "to-delete", "Will be deleted")

		store := NewDirectoryStore(SkillSource{Root: root, Precedence: 1})
		err := store.Delete("to-delete")
		if err != nil {
			t.Fatalf("Delete() error = %v", err)
		}

		// Verify directory is gone
		if _, err := os.Stat(filepath.Join(root, "to-delete")); !os.IsNotExist(err) {
			t.Error("skill directory should have been removed")
		}
	})

	t.Run("non-existent returns not found", func(t *testing.T) {
		root := t.TempDir()
		store := NewDirectoryStore(SkillSource{Root: root, Precedence: 1})
		err := store.Delete("nonexistent")
		if err == nil {
			t.Fatal("expected error for non-existent skill")
		}
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("expected ErrNotFound, got %v", err)
		}
	})

	t.Run("path traversal rejected", func(t *testing.T) {
		root := t.TempDir()
		store := NewDirectoryStore(SkillSource{Root: root, Precedence: 1})
		err := store.Delete("../etc")
		if err == nil {
			t.Fatal("expected error for path traversal")
		}
	})
}

func TestMultiSourceResolution(t *testing.T) {
	t.Run("higher precedence shadows lower for Read", func(t *testing.T) {
		projectRoot := t.TempDir()
		userRoot := t.TempDir()

		createSkillDir(t, projectRoot, "golang", "Project Go skill")
		createSkillDir(t, userRoot, "golang", "User Go skill")

		store := NewDirectoryStore(
			SkillSource{Root: projectRoot, Precedence: 10},
			SkillSource{Root: userRoot, Precedence: 1},
		)

		skill, err := store.Read("golang")
		if err != nil {
			t.Fatalf("Read() error = %v", err)
		}
		if skill.Description != "Project Go skill" {
			t.Errorf("expected project version, got %q", skill.Description)
		}
	})

	t.Run("skill only in lower source returned", func(t *testing.T) {
		projectRoot := t.TempDir()
		userRoot := t.TempDir()

		createSkillDir(t, userRoot, "speckit", "User speckit skill")

		store := NewDirectoryStore(
			SkillSource{Root: projectRoot, Precedence: 10},
			SkillSource{Root: userRoot, Precedence: 1},
		)

		skill, err := store.Read("speckit")
		if err != nil {
			t.Fatalf("Read() error = %v", err)
		}
		if skill.Description != "User speckit skill" {
			t.Errorf("expected user version, got %q", skill.Description)
		}
	})

	t.Run("List merges with dedup", func(t *testing.T) {
		projectRoot := t.TempDir()
		userRoot := t.TempDir()

		createSkillDir(t, projectRoot, "golang", "Project Go")
		createSkillDir(t, projectRoot, "project-only", "Project exclusive")
		createSkillDir(t, userRoot, "golang", "User Go")
		createSkillDir(t, userRoot, "user-only", "User exclusive")

		store := NewDirectoryStore(
			SkillSource{Root: projectRoot, Precedence: 10},
			SkillSource{Root: userRoot, Precedence: 1},
		)

		skills, err := store.List()
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}

		if len(skills) != 3 {
			names := make([]string, len(skills))
			for i, s := range skills {
				names[i] = s.Name
			}
			t.Fatalf("expected 3 skills, got %d: %v", len(skills), names)
		}

		// Find golang skill and verify it's the project version
		for _, s := range skills {
			if s.Name == "golang" {
				if s.Description != "Project Go" {
					t.Errorf("golang should be from project source, got %q", s.Description)
				}
			}
		}
	})

	t.Run("Write goes to first source", func(t *testing.T) {
		projectRoot := t.TempDir()
		userRoot := t.TempDir()

		store := NewDirectoryStore(
			SkillSource{Root: projectRoot, Precedence: 10},
			SkillSource{Root: userRoot, Precedence: 1},
		)

		if err := store.Write(Skill{Name: "new-skill", Description: "New skill"}); err != nil {
			t.Fatal(err)
		}

		// Should be written to projectRoot (highest precedence)
		if _, err := os.Stat(filepath.Join(projectRoot, "new-skill", "SKILL.md")); err != nil {
			t.Error("skill should be written to highest-precedence source")
		}
		if _, err := os.Stat(filepath.Join(userRoot, "new-skill", "SKILL.md")); !os.IsNotExist(err) {
			t.Error("skill should not be written to lower-precedence source")
		}
	})
}

// --- T013: Parse Fixture Skills ---

func TestParseSkillFixtures(t *testing.T) {
	for name, content := range testSkillFixtures {
		t.Run(name, func(t *testing.T) {
			skill, err := Parse([]byte(content))
			if err != nil {
				t.Fatalf("Parse failed for %s: %v", name, err)
			}

			if skill.Name == "" {
				t.Error("Name should not be empty")
			}
			if skill.Description == "" {
				t.Error("Description should not be empty")
			}
		})
	}
}

// --- T014: Round-Trip and Performance Tests ---

func TestSerializeRoundTrip(t *testing.T) {
	for name, content := range testSkillFixtures {
		t.Run(name, func(t *testing.T) {
			original, err := Parse([]byte(content))
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}

			serialized, err := Serialize(original)
			if err != nil {
				t.Fatalf("Serialize failed: %v", err)
			}

			reparsed, err := Parse(serialized)
			if err != nil {
				t.Fatalf("re-Parse failed: %v", err)
			}

			if reparsed.Name != original.Name {
				t.Errorf("Name: %q != %q", reparsed.Name, original.Name)
			}
			if reparsed.Description != original.Description {
				t.Errorf("Description: %q != %q", reparsed.Description, original.Description)
			}
			if reparsed.License != original.License {
				t.Errorf("License: %q != %q", reparsed.License, original.License)
			}
			if reparsed.Compatibility != original.Compatibility {
				t.Errorf("Compatibility: %q != %q", reparsed.Compatibility, original.Compatibility)
			}
			if len(reparsed.AllowedTools) != len(original.AllowedTools) {
				t.Errorf("AllowedTools: %v != %v", reparsed.AllowedTools, original.AllowedTools)
			}
			if len(reparsed.Metadata) != len(original.Metadata) {
				t.Errorf("Metadata: %v != %v", reparsed.Metadata, original.Metadata)
			}
		})
	}
}

func TestListPerformance(t *testing.T) {
	root := t.TempDir()

	// Create 55 skill directories
	for i := 0; i < 55; i++ {
		name := "skill-" + strings.Repeat("a", 3) + "-" + itoa(i)
		createSkillDir(t, root, name, "Skill number "+itoa(i))
	}

	store := NewDirectoryStore(SkillSource{Root: root, Precedence: 1})

	start := time.Now()
	skills, err := store.List()
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(skills) != 55 {
		t.Errorf("expected 55 skills, got %d", len(skills))
	}
	if elapsed > 500*time.Millisecond {
		t.Errorf("List took %v, want < 500ms", elapsed)
	}
}

// itoa is a simple int-to-string helper to avoid importing strconv.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}

// --- Error type coverage tests ---

func TestParseErrorFormat(t *testing.T) {
	t.Run("with value", func(t *testing.T) {
		e := &ParseError{Field: "name", Constraint: "required", Value: "bad"}
		s := e.Error()
		if !strings.Contains(s, "name") || !strings.Contains(s, "required") || !strings.Contains(s, "bad") {
			t.Errorf("unexpected error format: %s", s)
		}
		if e.Unwrap() != nil {
			t.Error("Unwrap should return nil")
		}
	})

	t.Run("without value", func(t *testing.T) {
		e := &ParseError{Field: "description", Constraint: "required"}
		s := e.Error()
		if !strings.Contains(s, "description") || !strings.Contains(s, "required") {
			t.Errorf("unexpected error format: %s", s)
		}
	})
}

func TestSkillErrorFormat(t *testing.T) {
	e := &SkillError{
		SkillName: "broken",
		Path:      "/tmp/broken/SKILL.md",
		Err:       &ParseError{Field: "name", Constraint: "required"},
	}
	s := e.Error()
	if !strings.Contains(s, "broken") || !strings.Contains(s, "/tmp/broken/SKILL.md") {
		t.Errorf("unexpected error format: %s", s)
	}
	if e.Unwrap() == nil {
		t.Error("Unwrap should return underlying error")
	}
}

func TestDiscoveryErrorFormat(t *testing.T) {
	e := &DiscoveryError{
		Errors: []SkillError{
			{SkillName: "a", Path: "/a", Err: &ParseError{Field: "name", Constraint: "required"}},
			{SkillName: "b", Path: "/b", Err: &ParseError{Field: "name", Constraint: "required"}},
		},
	}
	s := e.Error()
	if !strings.Contains(s, "discovery errors") {
		t.Errorf("unexpected error format: %s", s)
	}
}

func TestSplitFrontmatterEdgeCases(t *testing.T) {
	t.Run("frontmatter ending without trailing newline", func(t *testing.T) {
		input := "---\nname: test\n---"
		_, _, err := splitFrontmatter([]byte(input))
		if err != nil {
			t.Errorf("should handle frontmatter without trailing newline: %v", err)
		}
	})

	t.Run("CRLF line endings", func(t *testing.T) {
		input := "---\r\nname: test\r\n---\r\nBody\r\n"
		yaml, body, err := splitFrontmatter([]byte(input))
		if err != nil {
			t.Fatalf("should handle CRLF: %v", err)
		}
		if len(yaml) == 0 {
			t.Error("yaml should not be empty")
		}
		if body == "" {
			t.Error("body should not be empty")
		}
	})

	t.Run("empty after opening delimiter", func(t *testing.T) {
		input := "---"
		_, _, err := splitFrontmatter([]byte(input))
		if err == nil {
			t.Error("should fail for just opening delimiter")
		}
	})
}

func TestDescriptionMaxLength(t *testing.T) {
	long := strings.Repeat("x", 1025)
	input := "---\nname: test\ndescription: " + long + "\n---\n"
	_, err := Parse([]byte(input))
	if err == nil {
		t.Error("should fail for description > 1024 chars")
	}
	var pe *ParseError
	if !errors.As(err, &pe) || pe.Field != "description" {
		t.Errorf("expected description ParseError, got %v", err)
	}
}

func TestDirectoryStoreReadIOError(t *testing.T) {
	root := t.TempDir()
	// Create a skill directory but make the SKILL.md unreadable
	skillDir := filepath.Join(root, "no-read")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(skillDir, "SKILL.md")
	if err := os.WriteFile(path, []byte("---\nname: no-read\ndescription: test\n---\n"), 0000); err != nil {
		t.Fatal(err)
	}

	store := NewDirectoryStore(SkillSource{Root: root, Precedence: 1})
	_, err := store.Read("no-read")
	if err == nil {
		// Only fail if we're not root (root can read everything)
		if os.Getuid() != 0 {
			t.Fatal("expected error for unreadable SKILL.md")
		}
	}

	// Restore permissions for cleanup
	_ = os.Chmod(path, 0644)
}

func TestParseMetadataValidation(t *testing.T) {
	// Cover the YAML unmarshal error path in ParseMetadata
	input := "---\n[bad yaml\n---\nbody\n"
	_, err := ParseMetadata([]byte(input))
	if err == nil {
		t.Error("expected error for bad YAML in ParseMetadata")
	}

	// Cover allowed-tools parsing in ParseMetadata
	input = "---\nname: tools-meta\ndescription: Tools test\nallowed-tools: Read Write\n---\nbody\n"
	skill, err := ParseMetadata([]byte(input))
	if err != nil {
		t.Fatalf("ParseMetadata error: %v", err)
	}
	if len(skill.AllowedTools) != 2 {
		t.Errorf("expected 2 tools, got %d", len(skill.AllowedTools))
	}
	if skill.Body != "" {
		t.Errorf("ParseMetadata should have empty body, got %q", skill.Body)
	}
}

func TestListWithDirReadError(t *testing.T) {
	root := t.TempDir()
	createSkillDir(t, root, "valid-skill", "Valid")

	// Create a subdirectory without SKILL.md (should be silently skipped)
	if err := os.MkdirAll(filepath.Join(root, "no-skillmd"), 0755); err != nil {
		t.Fatal(err)
	}

	store := NewDirectoryStore(SkillSource{Root: root, Precedence: 1})
	skills, err := store.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(skills) != 1 {
		t.Errorf("expected 1 skill, got %d", len(skills))
	}
}

func TestDirectoryStoreWriteCreatesPath(t *testing.T) {
	root := t.TempDir()
	store := NewDirectoryStore(SkillSource{Root: filepath.Join(root, "nested", "dir"), Precedence: 1})

	err := store.Write(Skill{
		Name:        "deep-skill",
		Description: "Deeply nested",
		Body:        "body",
	})
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	// Verify the deeply nested path was created
	data, err := os.ReadFile(filepath.Join(root, "nested", "dir", "deep-skill", "SKILL.md"))
	if err != nil {
		t.Fatalf("file not created: %v", err)
	}
	if len(data) == 0 {
		t.Error("written file should not be empty")
	}
}

// --- Coverage boost: multi-source List merge ---

func TestDirectoryStoreListMultiSourceMerge(t *testing.T) {
	root1 := t.TempDir()
	root2 := t.TempDir()

	createSkillDir(t, root1, "only-in-1", "First source only")
	createSkillDir(t, root2, "only-in-2", "Second source only")
	createSkillDir(t, root1, "shared", "Shared from root1")
	createSkillDir(t, root2, "shared", "Shared from root2")

	store := NewDirectoryStore(
		SkillSource{Root: root1, Precedence: 10},
		SkillSource{Root: root2, Precedence: 1},
	)

	skills, err := store.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(skills) != 3 {
		names := make([]string, len(skills))
		for i, s := range skills {
			names[i] = s.Name
		}
		t.Fatalf("expected 3 skills, got %d: %v", len(skills), names)
	}

	// "shared" should come from root1 (higher precedence)
	for _, s := range skills {
		if s.Name == "shared" && s.Description != "Shared from root1" {
			t.Errorf("expected 'shared' from higher precedence, got description %q", s.Description)
		}
	}
}

func TestDirectoryStoreListReadDirError(t *testing.T) {
	root := t.TempDir()
	createSkillDir(t, root, "valid-skill", "Valid")

	// Create a source root that exists but is unreadable
	badRoot := filepath.Join(t.TempDir(), "unreadable")
	if err := os.MkdirAll(badRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(badRoot, 0o000); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chmod(badRoot, 0o755) }()

	store := NewDirectoryStore(
		SkillSource{Root: root, Precedence: 10},
		SkillSource{Root: badRoot, Precedence: 1},
	)

	// Should still return valid skills from readable sources
	skills, err := store.List()
	if os.Getuid() != 0 {
		// non-root should get a DiscoveryError for the unreadable dir
		var de *DiscoveryError
		if errors.As(err, &de) {
			if len(de.Errors) == 0 {
				t.Error("expected at least one discovery error for unreadable dir")
			}
		}
	}
	if len(skills) < 1 {
		t.Error("expected at least 1 valid skill from readable source")
	}
}

func TestDirectoryStoreListSkipsNonDirsAndSymlinks(t *testing.T) {
	root := t.TempDir()
	createSkillDir(t, root, "real-skill", "Real")

	// Create a regular file (not a directory) in the root
	if err := os.WriteFile(filepath.Join(root, "not-a-dir"), []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}

	store := NewDirectoryStore(SkillSource{Root: root, Precedence: 1})
	skills, err := store.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(skills) != 1 {
		t.Errorf("expected 1 skill, got %d", len(skills))
	}
}

func TestDirectoryStoreDeleteFromMultiSource(t *testing.T) {
	root1 := t.TempDir()
	root2 := t.TempDir()

	createSkillDir(t, root2, "in-lower", "In lower precedence")

	store := NewDirectoryStore(
		SkillSource{Root: root1, Precedence: 10},
		SkillSource{Root: root2, Precedence: 1},
	)

	// Delete from lower source
	err := store.Delete("in-lower")
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify gone
	_, err = store.Read("in-lower")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound after delete, got: %v", err)
	}
}

// --- T002: TestDirectoryStoreConcurrency — US1-5: concurrent access race-free ---

func TestDirectoryStoreConcurrency(t *testing.T) {
	root := t.TempDir()
	store := NewDirectoryStore(SkillSource{Root: root, Precedence: 1})

	const goroutines = 10
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			name := "skill-" + itoa(idx)

			// Write
			err := store.Write(Skill{
				Name:        name,
				Description: "Concurrent skill " + itoa(idx),
				Body:        "Body " + itoa(idx),
			})
			if err != nil {
				t.Errorf("Write(%q) error = %v", name, err)
				return
			}

			// Read
			s, err := store.Read(name)
			if err != nil {
				t.Errorf("Read(%q) error = %v", name, err)
				return
			}
			if s.Name != name {
				t.Errorf("Read(%q) got name %q", name, s.Name)
			}

			// Delete
			err = store.Delete(name)
			if err != nil {
				t.Errorf("Delete(%q) error = %v", name, err)
			}
		}(i)
	}

	wg.Wait()
}

// --- T003: TestParseCRLF — US1-1 + edge case: CRLF line endings parsed correctly ---

func TestParseCRLF(t *testing.T) {
	input := "---\r\nname: crlf-skill\r\ndescription: CRLF test skill\r\nlicense: MIT\r\ncompatibility: Claude 4.x\r\nmetadata:\r\n  author: test\r\n  version: \"1.0\"\r\nallowed-tools: \"Read Write Edit\"\r\n---\r\n# Hello CRLF\r\n\r\nThis is the body.\r\n"

	skill, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if skill.Name != "crlf-skill" {
		t.Errorf("Name = %q, want %q", skill.Name, "crlf-skill")
	}
	if skill.Description != "CRLF test skill" {
		t.Errorf("Description = %q, want %q", skill.Description, "CRLF test skill")
	}
	if skill.License != "MIT" {
		t.Errorf("License = %q, want %q", skill.License, "MIT")
	}
	if skill.Compatibility != "Claude 4.x" {
		t.Errorf("Compatibility = %q, want %q", skill.Compatibility, "Claude 4.x")
	}
	if skill.Metadata["author"] != "test" {
		t.Errorf("Metadata[author] = %q, want %q", skill.Metadata["author"], "test")
	}
	if len(skill.AllowedTools) != 3 || skill.AllowedTools[0] != "Read" || skill.AllowedTools[1] != "Write" || skill.AllowedTools[2] != "Edit" {
		t.Errorf("AllowedTools = %v, want [Read Write Edit]", skill.AllowedTools)
	}
	wantBody := "# Hello CRLF\r\n\r\nThis is the body.\r\n"
	if skill.Body != wantBody {
		t.Errorf("Body = %q, want %q", skill.Body, wantBody)
	}
}

// --- T004: TestSerializeCRLFRoundTrip — CRLF body content preserved through round-trip ---

func TestSerializeCRLFRoundTrip(t *testing.T) {
	original := Skill{
		Name:        "crlf-round",
		Description: "CRLF round-trip test",
		Body:        "# Hello\r\n\r\nBody with CRLF.\r\n",
	}

	data, err := Serialize(original)
	if err != nil {
		t.Fatalf("Serialize() error = %v", err)
	}

	parsed, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse(Serialize()) error = %v", err)
	}

	if parsed.Name != original.Name {
		t.Errorf("Name = %q, want %q", parsed.Name, original.Name)
	}
	if parsed.Description != original.Description {
		t.Errorf("Description = %q, want %q", parsed.Description, original.Description)
	}
	if parsed.Body != original.Body {
		t.Errorf("Body = %q, want %q", parsed.Body, original.Body)
	}
}
