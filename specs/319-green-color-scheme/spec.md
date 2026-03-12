# audit: partial — adopt green color scheme (#301)

**Issue**: [#319](https://github.com/re-cinq/wave/issues/319)
**Labels**: audit
**Author**: nextlevelshit
**State**: OPEN

## Audit Finding: Partial

**Source**: [#301 — feat(tui,docs): adopt green color scheme](https://github.com/re-cinq/wave/issues/301)

### Category
**Partial** — Issue requirements not fully implemented.

### Evidence
- `internal/tui/theme.go:14` `cyan = lipgloss.Color("6")` — still uses cyan, NOT green
- No green color definitions found in TUI theme
- No linked PRs or commits

### Remediation
Update `internal/tui/theme.go` to replace cyan with green palette (e.g., `lipgloss.Color("2")` or hex green). Update docs CSS variables.

## Acceptance Criteria

1. `internal/tui/theme.go` uses green (`Color("2")`) instead of cyan (`Color("6")`) as the primary accent
2. The `WaveLogo()` function in `theme.go` renders in green
3. The `LogoAnimator` in `header_logo.go` uses green as its base color
4. All TUI files that hardcode `Color("6")` are updated to use a centralized green constant
5. The `display` package color schemes use green (`\033[32m`) instead of cyan (`\033[36m`) for the Primary color
6. Docs CSS `--wave-secondary` updated from cyan to green
7. All existing tests pass with `go test ./...`
