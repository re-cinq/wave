## Objective

Validate the generated `.agents/*` overlay and write the onboarding sentinel. After this step exits, `wave init` and the webui driver must be able to detect the project as onboarded by checking `.agents/.onboarding-done`.

## Context

The generate step wrote files under `.agents/` according to the proposal manifest. Its report is available as the `generation` artifact at `.agents/artifacts/generation`, listing what was written, skipped, and any errors. The detection and proposal artifacts are also still available under `.agents/artifacts/`. You are in the project root in a worktree.

## Requirements

1. Read `.agents/artifacts/generation` and confirm `errors` is empty. If non-empty, abort and emit a clear failure note in `.agents/output/onboard-finalize.json`.
2. Verify the on-disk overlay:
   - At least one file exists under `.agents/personas/`
   - At least one file exists under `.agents/pipelines/`
   - At least one file exists under `.agents/prompts/`
   Use `ls .agents/personas .agents/pipelines .agents/prompts` to confirm.
3. Validate every generated YAML pipeline by running:
   ```bash
   wave validate .agents/pipelines/<file>.yaml
   ```
   for each `.yaml` under `.agents/pipelines/`. If `wave validate` is not on PATH, fall back to `go run ./cmd/wave validate <file>` from the project root if a `cmd/wave` source tree exists, otherwise skip with a recorded note.
4. Validate every generated markdown file is non-empty and has a level-1 heading.
5. Write the sentinel:
   ```bash
   mkdir -p .agents
   : > .agents/.onboarding-done
   ```
   This produces the same end-state as `onboarding.MarkDoneAt` (a zero-byte file at the documented sentinel path).
6. Emit a finalization report at `.agents/output/onboard-finalize.json`:
   ```json
   {
     "validated": [".agents/pipelines/example.yaml"],
     "warnings": [],
     "sentinel_written": true
   }
   ```

## Constraints

- Do NOT modify generated files in this step. Validation is read-only — fix issues by failing the step, not by patching.
- Do NOT write the sentinel before validation passes. A failed validation must leave the sentinel absent so the project re-runs onboarding.
- Do NOT write outside `.agents/`.
- Do NOT commit or push. The driver decides what to do with the worktree.

## Quality Bar

A good finalize step exits with the sentinel present, every generated YAML validated, and a clean report. A bad step writes the sentinel despite a validation failure, modifies generated files in place, or skips validation silently.
