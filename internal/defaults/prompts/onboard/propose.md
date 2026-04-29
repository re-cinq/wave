## Objective

Read the injected `detection` artifact and produce a manifest of `.agents/*` files that the next step (generate) will create. The proposal is the deterministic plan — once it is written, the generate step does not improvise additional files.

## Context

The previous detect step wrote `detection.json` (matching `detection.schema.json`) and it has been injected as the `detection` artifact. You can read it from `.agents/artifacts/detection`. You are still in the project root; later steps will write to `.agents/`. No `.agents/*` files exist yet beyond the injected artifact.

## Requirements

1. Load the detection artifact from `.agents/artifacts/detection` and parse it.
2. Plan a minimal, tailored overlay that includes at least:
   - One persona under `.agents/personas/` named after the flavour or project intent (e.g. `<project-slug>-implementer.md`). When `flavour` is `unknown`, name it after the project directory or a generic role.
   - One pipeline under `.agents/pipelines/` whose steps reference real commands for the detected flavour (e.g. the canonical `test_command` from detection). The pipeline should perform a small, useful loop — for example, `lint → test → review` for an established project, or `bootstrap → smoke` for a greenfield one.
   - At least one prompt file under `.agents/prompts/<pipeline-name>/` for the pipeline above.
3. For each planned file, record:
   - `path` — the full relative path under `.agents/`
   - `purpose` — a one-sentence description of what the file does
   - `references` — list of other planned paths it points to (used by generate to verify the graph)
4. Verify the manifest is internally consistent — every `references` entry must appear elsewhere in the manifest. Fix any dangling references before emitting.
5. Write the manifest to `.agents/output/onboard-proposal.json` as a JSON object of the shape:
   ```json
   {
     "flavour": "go",
     "files": [
       {"path": ".agents/personas/example.md", "purpose": "...", "references": []}
     ]
   }
   ```

## Constraints

- The manifest must list ONLY paths under `.agents/`.
- Every pipeline you propose must reference at least one persona and at least one prompt that you also propose.
- Do NOT propose files that already exist in `.agents/` (check `ls .agents` first).
- Do NOT inflate the manifest with files unrelated to the detected flavour or intent.

## Quality Bar

A good proposal lists 3–6 files, all under `.agents/`, with a coherent reference graph and obvious tailoring to the detected flavour. A bad proposal is generic, dangling-reference-laden, or stuffs unrelated files into the manifest.
