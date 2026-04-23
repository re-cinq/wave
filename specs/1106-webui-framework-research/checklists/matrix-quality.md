# Checklist: Comparison Matrix Requirements Quality

**Feature**: WebUI Framework Research (`1106-webui-framework-research`)  
**Date**: 2026-04-14  
**Scope**: Requirements quality for the comparison matrix deliverable  
**Validates**: FR-001, FR-002, FR-007, FR-008, FR-010, SC-001, SC-003

---

## Completeness

- [ ] CHK101 - Is each of the 9 evaluation criteria in FR-002 paired with a measurement method (what to measure, how to rate it) in either the spec, research.md criterion weights table, or matrix contract — rather than requiring researcher inference? [Completeness]
- [ ] CHK102 - Does the spec require an elimination section in the matrix even when no candidate is eliminated, or does the elimination section only exist when needed? Is this distinction explicit? [Completeness]
- [ ] CHK103 - Is the categorization scheme for handler impact (No change / Template removal / Adapter needed) defined in a functional requirement or contract, not only in research.md's "Technology Assessment"? [Completeness]
- [ ] CHK104 - Does FR-007 or SC-003 require both raw and gzipped bundle size measurements, or only one? (Research.md U3 specifies both, but is this requirement binding?) [Completeness]

---

## Clarity

- [ ] CHK105 - Is the minimum evidence standard per matrix cell (at minimum 2 sentences, as specified in the matrix contract) captured as a requirement in the spec itself, or only in the contract document that implementers may not consult until Phase 6 validation? [Clarity]
- [ ] CHK106 - Does the spec define what "concrete examples" means for the SSE row (acceptance scenario US1-3 says "concrete examples or code snippets") — is a prose description sufficient, or is code required? [Clarity]
- [ ] CHK107 - Is the rating scale (Strong / Good / Adequate / Weak / Fail) defined with anchoring descriptions for each level, or is the scale defined only by its labels — leaving subjective interpretation open? [Clarity]

---

## Consistency

- [ ] CHK108 - Does the bundle size baseline in `contracts/matrix-contract.md` item 6 (JS ~127 KB, CSS ~156 KB, total ~283 KB) agree with the corrected values in spec C4 clarification (JS ~124 KB, CSS ~152 KB, total ~276 KB) and research.md U3? [Consistency]
- [ ] CHK109 - Is the "Strong / Good / Adequate / Weak / Fail" scale defined in exactly one authoritative place and referenced (not redefined) elsewhere — or is it defined in multiple places with potential for divergence? [Consistency]

---

## Coverage

- [ ] CHK110 - Does any matrix criterion or handler impact section explicitly require documenting each candidate's Node.js build-time dependency (not just runtime), so that CI environment impact is captured in the matrix? [Coverage]
- [ ] CHK111 - Does the component reuse criterion (FR-010) require documenting migration complexity for the 30 custom template functions in `embed.go`, or only for the 6 HTML partials listed in the requirement? [Coverage]
- [ ] CHK112 - Does the matrix cover all 5 edge cases from the spec — either as dedicated rows, subsections, or explicit callouts within relevant criteria rows — rather than leaving edge cases to be addressed only in the recommendation? [Coverage]
