## Objective

Materialise the proposal manifest into real files under `.agents/`. Read the proposal, write each file, and stop. No improvisation, no extra files, no edits outside `.agents/`.

## Context

The propose step wrote `onboard-proposal.json` and it has been injected as the `proposal` artifact at `.agents/artifacts/proposal`. The detect step's output is also still available at `.agents/artifacts/detection`. You are running in a worktree; the working directory is the project root.

## Requirements

1. Read the proposal manifest from `.agents/artifacts/proposal` and parse the `files` array.
2. Read the detection artifact from `.agents/artifacts/detection` for context (flavour, test command, signals).
3. For each entry in the proposal `files` array:
   1. Verify the `path` starts with `.agents/`. If it does not, abort with a clear error — never write outside `.agents/`.
   2. Skip the file if it already exists on disk and record a note in `.agents/output/onboard-generation.json`.
   3. Create parent directories with `mkdir -p`.
   4. Write the file with content tailored to the detected flavour:
      - **Personas**: short markdown describing role, rules, responsibilities, constraints — no language-specific commands.
      - **Pipelines**: valid Wave pipeline YAML referencing the personas and prompts defined elsewhere in the proposal. Use the detection `test_command` for any test-running step. Include the standard `kind: WavePipeline`, `metadata.name`, `metadata.description`, `steps[]` shape.
      - **Prompts**: markdown with Objective / Context / Requirements / Output Format / Constraints / Quality Bar sections, mirroring the shape of the prompt you are reading now.
4. After writing all files, emit a generation report at `.agents/output/onboard-generation.json`:
   ```json
   {
     "written": [".agents/personas/example.md", ...],
     "skipped": [".agents/pipelines/already-exists.yaml"],
     "errors": []
   }
   ```
5. Do NOT run any tests, builds, or validation commands here — that's the finalize step's job.

## Constraints

- Every write target MUST start with `.agents/`. Hard fail otherwise.
- Do NOT modify `wave.yaml`, `go.mod`, `package.json`, source files, CI files, or anything outside `.agents/`.
- Do NOT delete files. Skipping is the only valid response to a pre-existing file.
- Do NOT include placeholder content like `TODO` or `// fill me in` — every file must be functional on first read.
- Do NOT add Co-Authored-By or AI attribution lines to any generated file.

## Quality Bar

A good generation step writes exactly the files in the manifest, each tailored to the detected flavour, with no leakage outside `.agents/`. A bad step touches `wave.yaml`, writes generic boilerplate, or leaves dangling references because it skipped a referenced file silently.
