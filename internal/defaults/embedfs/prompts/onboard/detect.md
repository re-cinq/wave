## Objective

Survey the project mounted at `/project` and emit a structured detection JSON describing its flavour, build system, test command, and the concrete signals that informed the detection. The output is consumed by the next pipeline step (propose) — accuracy and completeness here decide whether the generated `.agents/*` overlay is useful or generic.

## Context

This is the first step of the `onboard-project` pipeline. You have read-only access to the project directory at `/project`. The project may be greenfield (almost empty), partially scaffolded, or a mature codebase with existing tooling. Your job is to look at what is actually on disk — not to guess from the directory name or a README alone.

## Requirements

Execute these steps in order:

1. List the top two levels of the project directory: `ls /project` and `find /project -maxdepth 2 -type f`. Record every file path you'll cite as a signal.
2. Read manifest and lockfile heads (first 50–100 lines each) for any of the following that exist:
   - `go.mod`, `go.sum`
   - `Cargo.toml`, `Cargo.lock`
   - `package.json`, `package-lock.json`, `yarn.lock`, `pnpm-lock.yaml`, `bun.lockb`, `deno.json`, `deno.lock`
   - `pyproject.toml`, `setup.py`, `requirements.txt`, `Pipfile`
   - `pom.xml`, `build.gradle`, `build.gradle.kts`
   - `*.csproj`, `*.sln`
   - `Gemfile`
3. Read `wave.yaml` if present and extract `project.language`, `project.build_command`, `project.test_command`. These take precedence over inferred values.
4. Read the README (any case) for project intent — capture a one-sentence summary in `project_intent` if available.
5. Determine the flavour using this precedence:
   1. `wave.yaml` `project.language` if explicitly set.
   2. Manifest signal: `go.mod` → `go`; `Cargo.toml` → `rust`; `deno.json` → `deno`; `bun.lockb` → `bun`; `package.json` → `node`; `pyproject.toml` or `setup.py` → `python`; `*.csproj` / `*.sln` → `csharp`; `pom.xml` / `build.gradle*` → `java`; `Gemfile` → `ruby`.
   3. If still ambiguous, emit `flavour: "unknown"` and capture the unusual evidence under `additional_signals`.
6. Determine the canonical test command: prefer `wave.yaml` `project.test_command`; otherwise the flavour default (`go test ./...`, `cargo test`, `npm test`, `bun test`, `pytest`, `dotnet test`, `mvn test`, `gradle test`, `bundle exec rake test`).
7. Detect frameworks only when there's a strong signal (e.g. a top-level dependency entry referencing the framework name).

## Output Format

Produce JSON matching `detection.schema.json`. Every entry in `signals[]` must be a real path you observed via `ls` / `find`. When a field cannot be determined honestly, use `null` (for nullable fields) or `unknown` (for `flavour`).

Write the output to `.agents/output/detection.json`.

## Constraints

- Do NOT write any files outside `.agents/output/`.
- Do NOT invent signals you didn't observe.
- Do NOT pick a flavour to satisfy the schema — `unknown` is preferred over a wrong guess.
- Do NOT read or modify files outside the `/project` mount.

## Quality Bar

Good detection cites at least one concrete signal per non-null field, picks the right flavour on the first try when manifest evidence exists, and degrades to `unknown` cleanly when it doesn't. Bad detection guesses, omits signals, or hallucinates a `test_command` the project doesn't actually support.
