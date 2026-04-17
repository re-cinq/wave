# Skill Ecosystem Integration Guide

This guide covers how to install skills from the supported ecosystem adapters: Tessl, BMAD, OpenSpec, SpecKit, GitHub, File, and URL.

## Tessl

The Tessl registry is the primary skill distribution platform.

### Prerequisites

```bash
npm i -g @tessl/cli
```

### Install a Skill

```bash
wave skills install tessl:github/golang
wave skills install tessl:github/spec-kit
```

### Search the Registry

```bash
wave skills search golang
```

### Sync Project Dependencies

Syncs all skill dependencies declared in your project configuration:

```bash
wave skills sync
```

## BMAD

The BMAD (Breakthrough Method for Agile AI-Driven Development) ecosystem provides method-specific skills.

### Prerequisites

```bash
npm i -g npx
```

`npx` is included with npm 5.2+ by default.

### Install

```bash
wave skills install bmad:install
```

This runs `npx bmad-method install --tools claude-code --yes` behind the scenes, discovering and installing all BMAD skills.

## OpenSpec

The OpenSpec ecosystem provides specification-driven development skills.

### Prerequisites

```bash
npm i -g @openspec/cli
```

### Install

```bash
wave skills install openspec:init
```

This runs `openspec init` to initialize and discover OpenSpec skills.

## SpecKit

The SpecKit ecosystem provides specification toolkit skills.

### Prerequisites

```bash
npm i -g @speckit/cli
```

### Install

```bash
wave skills install speckit:init
```

This runs `specify init` to initialize and discover SpecKit skills.

## GitHub

Install skills directly from GitHub repositories.

### Format

```
github:<owner>/<repo>[/<path>]
```

### Install

```bash
wave skills install github:your-org/your-skills-repo/golang
wave skills install github:user/repo
```

The adapter clones the repository (or fetches the specific path) and discovers SKILL.md files.

## File

Install skills from local filesystem directories.

### Format

```
file:<path>
```

The path can be absolute or relative to the project root.

### Install

```bash
wave skills install file:./my-skill
wave skills install file:/absolute/path/to/skill
```

The directory must contain a valid SKILL.md file. Path traversal outside the project root is rejected for security.

## URL

Install skills from remote archives via HTTPS.

### Format

```
https://<url-to-archive>
```

Supports `.tar.gz` and `.zip` archives.

### Install

```bash
wave skills install https://example.com/skills/golang.tar.gz
```

The archive is downloaded, extracted to a temporary directory, and SKILL.md files are discovered and installed. Only HTTPS URLs are accepted.

## Common Operations

### List Installed Skills

```bash
wave skills list
wave skills list --format json
```

### Remove a Skill

```bash
wave skills remove golang
wave skills remove golang --yes  # Skip confirmation
```

### Check Skill Status

```bash
wave skills list --format json | jq '.skills[] | .name'
```

## Error Handling

| Error | Meaning | Solution |
|-------|---------|----------|
| `skill_dependency_missing` | Required CLI tool not installed | Install the prerequisite (see adapter sections above) |
| `skill_not_found` | Skill name not in store | Check spelling, run `wave skills list` |
| `skill_source_error` | Invalid source prefix or format | Use a recognized prefix: `tessl:`, `bmad:`, `openspec:`, `speckit:`, `github:`, `file:`, `https://` |
