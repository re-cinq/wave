# feat(pipeline): add ops-bootstrap pipeline for greenfield project scaffolding

**Issue**: [#405](https://github.com/re-cinq/wave/issues/405)
**Parent**: [#402](https://github.com/re-cinq/wave/issues/402)
**Labels**: enhancement, pipeline

## Summary

Add a new `ops-bootstrap` pipeline that scaffolds greenfield projects with language-appropriate project structure, CI config, and initial files.

## Problem

Wave has no support for starting a project from scratch. After `wave init`, users have an empty project with Wave config but no actual code structure. They need to manually create project files before any implementation pipeline can run.

## Solution

New `ops-bootstrap` pipeline with 3 steps:

### Step 1: `assess` (navigator)
- Read `wave.yaml` to determine `project.flavour`
- Read existing files to understand what already exists
- Read any README, ADR, or design docs for project intent
- Output: assessment artifact with project intent + flavour + existing structure

### Step 2: `scaffold` (craftsman)
- Based on flavour, create initial project structure:
  - **rust**: `Cargo.toml`, `src/main.rs` or `src/lib.rs`, `tests/`, `.github/workflows/ci.yml`
  - **go**: `go.mod`, `main.go` or `cmd/`, `internal/`, `.github/workflows/ci.yml`
  - **node/bun**: `package.json`, `src/index.ts`, `tsconfig.json`, `.github/workflows/ci.yml`
  - **python**: `pyproject.toml`, `src/`, `tests/`, `.github/workflows/ci.yml`
  - **csharp**: `*.csproj`, `Program.cs`, `*.sln`, `.github/workflows/ci.yml`
  - etc.
- Create `.gitignore` appropriate for language
- Create initial README with project description
- Verify project builds: run `{{ project.build_command }}`
- Verify tests pass: run `{{ project.test_command }}`

### Step 3: `commit` (craftsman)
- Stage all new files
- Create initial commit with conventional message: `feat: scaffold {flavour} project`
- Push to remote if configured

## CLI Usage
```bash
# After wave init on empty project
wave run ops-bootstrap "Rust CLI tool for processing CSV files"

# With explicit flavour override
wave run ops-bootstrap "Python data pipeline" --set project.flavour=python
```

## Files to Create
- `internal/defaults/pipelines/ops-bootstrap.yaml` — pipeline definition
- `internal/defaults/contracts/bootstrap-assessment.schema.json` — assessment contract

## Acceptance Criteria
- [ ] Pipeline creates working project skeleton for at least: go, rust, node, bun, python, csharp
- [ ] Generated project builds and passes initial test
- [ ] Appropriate .gitignore for language
- [ ] CI config for GitHub Actions
- [ ] Initial commit created with conventional message
- [ ] Works on completely empty directory (post-wave init)
