# Research: WebUI Changed-Files Browser with Diff Views

**Feature Branch**: `565-webui-diff-browser`
**Date**: 2026-03-25

## Phase 0 — Unknowns & Research

### R-001: Git Diff Computation Strategy

**Decision**: Use `os/exec` to invoke `git diff` subprocess from the Go backend.

**Rationale**: Wave already uses `os/exec` extensively for git operations (`internal/worktree/worktree.go`). The codebase has no git library dependency (e.g., go-git) and adding one would violate the minimal dependency principle. The `git diff --stat` and `git diff` commands are well-suited for this use case.

**Alternatives Rejected**:
- **go-git library**: Adds a large dependency (~200+ transitive packages), violates Principle 1 (Single Binary, Minimal Dependencies). Would require vendoring and increases binary size.
- **Parse .git objects directly**: Too complex for diff computation, would duplicate git's well-tested diffing logic.

**Implementation Details**:
- Summary endpoint: `git diff --stat --numstat <base>...<branch>` — returns file list with add/delete counts
- Single-file endpoint: `git diff <base>...<branch> -- <path>` — returns unified diff for one file
- Base branch resolution: `git symbolic-ref refs/remotes/origin/HEAD` → strip prefix → fallback `main` → `master`
- Three-dot syntax (`...`) compares merge-base, matching standard PR diff behavior

### R-002: Frontend Diff Rendering (No External Libraries)

**Decision**: Implement a vanilla JavaScript diff parser and renderer with CSS-based syntax highlighting.

**Rationale**: The webui uses a zero-dependency embedded asset model (no npm, no build step). All existing JS files (`app.js`, `log-viewer.js`, `sse.js`, `dag.js`) are plain vanilla JavaScript. Adding a library like diff2html or Monaco would require a build step.

**Alternatives Rejected**:
- **diff2html**: Excellent library but requires npm install + bundling. Incompatible with `embed.FS` approach.
- **Monaco editor**: Massive bundle (~2MB minified), complete overkill for read-only diff viewing.
- **Server-side HTML rendering**: Would increase response size and complicate caching. Better to send raw diff text and render client-side.

**Implementation Details**:
- Parse unified diff format client-side (split on `@@` hunks, identify `+`/`-`/` ` lines)
- Three view modes: unified (standard), side-by-side (dual column), raw (before/after file content)
- CSS handles coloring: green background for additions, red for deletions
- Syntax highlighting via regex token matching: keywords, strings, comments, numbers per language

### R-003: Virtualization for Large Diffs

**Decision**: Implement viewport-based rendering using a sliding window technique.

**Rationale**: The spec requires virtualization for diffs exceeding 500 lines (FR-010, SC-004). Without it, rendering 5000+ lines causes browser janking and memory issues.

**Alternatives Rejected**:
- **Full DOM rendering with `display:none`**: Still creates all DOM nodes, no memory savings.
- **Intersection Observer lazy loading**: Better for infinite scroll, not ideal for diff where users jump around.

**Implementation Details**:
- Calculate total height from line count × fixed line height (monospace font)
- Render only visible lines + buffer (e.g., ±50 lines)
- Update rendered lines on scroll via `requestAnimationFrame`
- Container has `overflow-y: auto` with spacer elements for correct scrollbar

### R-004: Path Sanitization

**Decision**: Validate file paths from git diff output server-side before returning to client.

**Rationale**: FR-013 requires path traversal prevention. The existing artifact handler (`handlers_artifacts.go:46`) uses `filepath.Clean` + `strings.Contains(cleanPath, "..")` pattern. The security package has robust `PathValidator` in `internal/security/path.go`.

**Implementation Details**:
- Reject paths containing `..` or starting with `/`
- Reject paths outside repository root
- For the single-file endpoint, validate the `{path...}` parameter matches an entry from `git diff --stat`
- HTML-escape all path strings before returning in JSON (prevent XSS via crafted filenames)

### R-005: Existing WebUI Patterns

**Key patterns observed from codebase analysis**:

| Pattern | Implementation |
|---------|---------------|
| Route registration | `routes.go` — `mux.HandleFunc("GET /api/runs/{id}/...", handler)` |
| JSON API responses | `writeJSON(w, status, data)` / `writeJSONError(w, status, msg)` |
| HTML page handlers | Template execution via `s.templates["name"].ExecuteTemplate(w, "templates/layout.html", data)` |
| Path parameters | `r.PathValue("id")` (Go 1.22+ ServeMux) |
| Wildcard paths | `{path...}` in route pattern (Go 1.22+) |
| Test setup | `testServer(t)` returns `*Server, state.StateStore` with temp DB |
| State access | `s.store.GetRun(runID)` for read-only, `RunRecord.BranchName` for branch |
| Embedded assets | `//go:embed static/*` / `//go:embed templates/*` |
| Template functions | `funcMap` in `embed.go` — `statusClass`, `formatTime`, etc. |
| Static JS files | Plain vanilla JS in `internal/webui/static/` — no build step |
| Error handling | `writeJSONError` for API, `http.Error` for HTML pages |
