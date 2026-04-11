# Contract: runs.html Template Rendering

**Type**: Behavioral (template output contract)  
**Feature**: `772-webui-running-pipelines`

## Description

The `runs.html` Go template, when rendered with the extended template data,
must produce HTML output conforming to these structural requirements for the
running-pipelines section.

## Required HTML Structure

```html
<!-- Section container — always present in page output -->
<div class="rp-section" id="rp-section">

  <!-- Header — always rendered, always keyboard-accessible -->
  <div class="rp-header"
       id="rp-section-header"
       role="button"
       tabindex="0"
       aria-expanded="true"
       aria-controls="rp-section-body"
       onclick="toggleRunningSection()"
       onkeydown="if(event.key==='Enter'||event.key===' '){toggleRunningSection();}">
    <span class="rp-label">Running</span>
    <span class="rp-badge" aria-label="N running pipelines">N</span>
    <span class="rp-chevron" aria-hidden="true">▾</span>
  </div>

  <!-- Body — hidden attribute removed/added by toggleRunningSection() -->
  <div class="rp-body" id="rp-section-body">

    <!-- CASE A: No running pipelines -->
    <div class="rp-empty">
      <!-- Message text present -->
      <a href="/pipelines" class="rp-cta"><!-- CTA text present --></a>
    </div>

    <!-- CASE B: Running pipelines present (one per RunSummary) -->
    <a href="/runs/{RunID}" class="wr-run">
      <!-- Same card structure as main list -->
    </a>

  </div>
</div>
```

## Invariants

1. `.rp-section` is always present in the rendered output (SC-001).
2. On initial render, `aria-expanded="true"` and body does NOT have `hidden` attribute (SC-002).
3. When `RunningCount > 0`: each `RunSummary` in `RunningRuns` produces exactly one
   `<a href="/runs/{RunID}" class="wr-run">` element in `.rp-body` (SC-003).
4. When `RunningCount == 0`: `.rp-empty` is rendered and contains exactly one `<a class="rp-cta">` (SC-004).
5. `aria-expanded` value matches the actual expanded/collapsed state of `.rp-body` (SC-005).
6. The `.rp-section` block appears AFTER `.wr-toolbar` and BEFORE `.wr-list` in document order (FR-001).

## Verification

- Visual inspection of rendered `/runs` page in browser
- Accessibility check: `aria-expanded` reflects state, Enter/Space operable
- Template test: parse rendered HTML and assert structural invariants
