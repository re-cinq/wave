# Skill Authoring Guide

This guide covers how to create custom skills for Wave, including the SKILL.md format, resource directories, naming conventions, and best practices.

## SKILL.md Format

Every skill is defined by a `SKILL.md` file inside a named directory. The file uses YAML frontmatter followed by a markdown body.

### Structure

```
my-skill/
├── SKILL.md           # Required: skill definition
├── scripts/           # Optional: executable scripts
├── references/        # Optional: reference documents
└── assets/            # Optional: static assets
```

### Frontmatter Fields

The frontmatter is enclosed between `---` delimiters and contains YAML key-value pairs.

| Field | Required | Max Length | Description |
|-------|----------|-----------|-------------|
| `name` | Yes | 64 chars | Unique skill identifier matching the directory name |
| `description` | Yes | 1024 chars | Human-readable summary of the skill |
| `license` | No | — | License identifier (e.g., `MIT`, `Apache-2.0`) |
| `compatibility` | No | 500 chars | Compatibility notes (e.g., `Claude 4.x`) |
| `metadata` | No | — | Arbitrary key-value pairs for custom metadata |
| `allowed-tools` | No | — | Space-separated list of tools the skill may use |

### Example SKILL.md

```markdown
---
name: golang
description: Expert Go language development including idiomatic patterns and concurrency
license: MIT
compatibility: Claude 4.x
metadata:
  author: re-cinq
  version: "1.0"
allowed-tools: "Read Write Edit Bash Grep Glob"
---
# Go Development

You are an expert Go developer. Follow idiomatic Go patterns including:

- Use `gofmt` and `go vet` for code formatting
- Prefer composition over inheritance
- Handle errors explicitly
- Use table-driven tests
```

### Field Details

**name**: Must match the regex `^[a-z0-9]([a-z0-9-]*[a-z0-9])?$` and the enclosing directory name. Only lowercase alphanumeric characters and hyphens are allowed. Cannot start or end with a hyphen. Maximum 64 characters.

**description**: A concise summary displayed in `wave skills list` output. Maximum 1024 characters.

**allowed-tools**: A space-separated string of tool names that the skill is allowed to use. When specified, these are included in the runtime CLAUDE.md for the pipeline step.

**metadata**: Arbitrary key-value pairs. Values must be strings. Useful for versioning, authorship, and categorization.

## Resource Directories

Skills can include supporting files in three predefined subdirectories:

### scripts/

Executable scripts that the skill may reference during execution. Common uses include setup scripts, validation helpers, and build tools.

```
my-skill/
├── SKILL.md
└── scripts/
    ├── setup.sh
    └── validate.py
```

### references/

Reference documents such as API schemas, configuration templates, and specification files.

```
my-skill/
├── SKILL.md
└── references/
    ├── api-schema.json
    └── config-template.yaml
```

### assets/

Static assets such as templates, images, and data files.

```
my-skill/
├── SKILL.md
└── assets/
    ├── template.txt
    └── sample-data.csv
```

When a skill is provisioned into a workspace, all resource files are copied to `.agents/skills/<name>/` preserving the directory structure.

## Naming Conventions

- Use lowercase alphanumeric characters and hyphens: `my-skill`, `golang`, `spec-kit`
- Maximum 64 characters
- Must match the pattern: `^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`
- Cannot start or end with a hyphen
- The directory name must match the `name` field in SKILL.md

**Valid names**: `golang`, `my-skill`, `spec-kit`, `a`, `skill1`

**Invalid names**: `MySkill`, `my_skill`, `my.skill`, `-foo`, `foo-`, `../etc`

## Best Practices

1. **Keep skills focused**: Each skill should cover a single domain or capability. Prefer multiple small skills over one large skill.

2. **Write clear descriptions**: The description appears in `wave skills list` and search results. Make it informative and concise.

3. **Use allowed-tools sparingly**: Only declare tools the skill actually needs. This helps enforce least-privilege access.

4. **Include metadata**: Add `author` and `version` metadata to help users understand provenance and compatibility.

5. **Test your skill**: Install locally with `wave skills install file:./my-skill` and verify it parses correctly.

6. **Document in the body**: The markdown body is injected into the agent's context. Write clear instructions, examples, and constraints.

## Installing a Custom Skill

To install a skill from a local directory:

```bash
wave skills install file:./path/to/my-skill
```

To verify it installed correctly:

```bash
wave skills list
```
