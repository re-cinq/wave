# Contributing to Wave

Thank you for your interest in contributing to Wave! This document provides guidelines and instructions for contributing.

## Prerequisites

- **Go 1.25+** — [Download](https://go.dev/dl/)
- **An LLM CLI adapter** — [Claude Code](https://docs.anthropic.com/en/docs/claude-code) (`claude`) or another supported adapter
- **Git** — for version control and worktree operations

## Getting Started

```bash
# Clone the repository
git clone https://github.com/re-cinq/wave.git
cd wave

# Build from source
make build

# Run the test suite
make test
```

## Development Workflow

1. **Fork** the repository and create a feature branch from `main`
2. **Write tests** for any new functionality
3. **Run the full test suite** before submitting:
   ```bash
   go test -race ./...
   ```
4. **Commit** using [conventional commit](https://www.conventionalcommits.org/) prefixes
5. **Open a pull request** against `main`

## Commit Conventions

Wave uses conventional commits for automated versioning. Every merge to `main` produces a release.

| Prefix | Version Bump | Example |
|--------|-------------|---------|
| `fix:` | patch (0.0.X) | `fix: resolve workspace cleanup race condition` |
| `feat:` | minor (0.X.0) | `feat: add Bitbucket pipeline support` |
| `feat!:` or `BREAKING CHANGE:` | major (X.0.0) | `feat!: redesign contract validation API` |
| `docs:`, `test:`, `chore:`, `refactor:` | patch (0.0.X) | `docs: update installation guide` |

## Code Standards

- Follow idiomatic Go practices (`gofmt`, `go vet`)
- Single responsibility per package
- Use interfaces for testability and dependency injection
- Table-driven tests with edge case coverage
- No `t.Skip()` without a linked issue

## Project Structure

See the [README](README.md) for an overview of the project structure and core concepts.

## Running Tests

```bash
# Run all tests
go test ./...

# Run with race detector (required for PRs)
go test -race ./...

# Run tests for a specific package
go test -race ./internal/pipeline/...
```

## Reporting Issues

- Use [GitHub Issues](https://github.com/re-cinq/wave/issues) to report bugs or request features
- Include reproduction steps, expected behavior, and actual behavior
- Check existing issues before creating a new one

## License

By contributing to Wave, you agree that your contributions will be licensed under the [MIT License](LICENSE).
