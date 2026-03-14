# UX & Interaction Flow Checklist: Wave Init Interactive Skill Selection

**Feature**: `384-init-skill-selection`
**Date**: 2026-03-14
**Focus**: User experience flow completeness and interaction design requirements

---

## Flow Continuity

- [ ] CHK033 - Is the transition from Step 5 (model selection) to Step 6 (skill selection) specified — does the user see a visual separator, header, or seamless continuation? [Completeness]
- [ ] CHK034 - Is the post-installation summary display defined — does the user see a list of what was installed before the wizard completes? [Completeness]
- [ ] CHK035 - Are the ecosystem selection option descriptions specified — does the user see just names ("tessl") or names with descriptions ("tessl — AI skill ecosystem")? [Completeness]
- [ ] CHK036 - Is the install-all confirmation prompt wording specified — does the user know exactly what "all skills" means for BMAD/OpenSpec/Spec-Kit? [Clarity]
- [ ] CHK037 - Is the order of ecosystem options in the selection form specified (alphabetical, popularity, fixed)? [Completeness]

## Error UX

- [ ] CHK038 - When a single skill fails in a batch, is the error message format for the failed skill defined separately from the success format? [Clarity]
- [ ] CHK039 - After showing install instructions for a missing CLI, is the "retry" flow defined — can the user install the CLI in another terminal and retry within the wizard? [Completeness]
- [ ] CHK040 - Is the network error message for `tessl search` failure distinguishable from a CLI-missing error? [Clarity]

## Reconfiguration UX

- [ ] CHK041 - During reconfigure, is it clear whether the user can remove previously installed skills, or only add new ones? [Clarity]
- [ ] CHK042 - When reconfiguring with a different ecosystem than originally used, are the implications stated — do old skills remain alongside new ecosystem skills? [Completeness]
