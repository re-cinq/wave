# Frontend UX Quality Checklist: WebUI Diff Viewer

## Interaction Requirements

- [ ] CHK044 - Is the click/selection interaction for the file list defined — does clicking a file replace the previous diff, or can multiple diffs be expanded simultaneously? [Clarity]
- [ ] CHK045 - Is the visual feedback for the currently-selected file in the file list specified (highlight, border, background)? [Completeness]
- [ ] CHK046 - Is the loading state specified when a single-file diff is being fetched (spinner, skeleton, "Loading..." text)? [Completeness]
- [ ] CHK047 - Is keyboard navigation specified for the file list (arrow keys, Enter to select) or is mouse-only interaction acceptable? [Coverage]
- [ ] CHK048 - Is the behavior specified when the user resizes the browser window — does the diff panel reflow, and at what breakpoint? [Completeness]

## View Mode Switching

- [ ] CHK049 - Is the default state of the "Raw" sub-toggle (Before/After) specified when switching TO raw mode from another mode? [Clarity]
- [ ] CHK050 - Is scroll position behavior specified when switching view modes — does the viewer maintain scroll position or reset to top? [Completeness]
- [ ] CHK051 - Are the toggle button labels exactly specified ("Unified", "Side-by-side", "Raw"), or are implementations free to choose labels? [Clarity]

## Diff Rendering

- [ ] CHK052 - Is line wrapping behavior specified for long lines in the diff viewer — wrap, horizontal scroll, or truncate? [Completeness]
- [ ] CHK053 - Is the diff hunk header format (`@@ -a,b +c,d @@`) rendering specified — is it displayed verbatim, styled, or enriched with context? [Clarity]
- [ ] CHK054 - Are empty-state messages specified for: no files changed, diff unavailable, binary file selected, network error? [Completeness]
- [ ] CHK055 - Is the color scheme for diff markers specified with sufficient contrast ratios for accessibility (WCAG AA)? [Coverage]

## Syntax Highlighting

- [ ] CHK056 - Is fallback behavior specified when a file extension is not in the supported list of 10 languages — plain text, or no highlighting? [Completeness]
- [ ] CHK057 - Are the regex-based highlighting accuracy expectations defined — is "good enough" quantified, or are known limitations documented? [Clarity]
- [ ] CHK058 - Is highlighting behavior specified for diff lines that span language boundaries (e.g., embedded SQL in Go)? [Coverage]

## Performance UX

- [ ] CHK059 - Is the user-visible feedback specified when the file list API request exceeds the 3-second SLA (SC-001) — timeout, retry, or loading indicator? [Completeness]
- [ ] CHK060 - Is the behavior specified when virtualization is active — can the user Ctrl+F/search within the diff? [Coverage]
- [ ] CHK061 - Is the scroll behavior of the virtualized diff specified — smooth scrolling, or instant jump on fast scroll? [Clarity]
