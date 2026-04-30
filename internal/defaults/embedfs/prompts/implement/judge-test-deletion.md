You are capturing the diff of `_test.go` files for a downstream LLM-as-judge gate.

## Working Directory

You are running in an **isolated git worktree** shared with the implement step.
Your working directory IS the project root. The feature branch is already checked
out and contains the implementation diff.

## Objective

Produce `.agents/output/test-diff.md` containing the diff of every `_test.go` file
touched on this branch versus the base branch. The downstream `llm_judge` contract
reads this file and rejects the step if any `func Test*` declarations are net-removed
without a demonstrable replacement.

## Instructions

### Step 1: Capture the diff

Run the following from the workspace root:

```bash
git diff --no-color main...HEAD -- '*_test.go'
```

If the command fails (e.g. unusual workspace state), capture stderr and continue —
do NOT abort. The judge tolerates a sentinel header.

### Step 2: Write the artifact

Create `.agents/output/test-diff.md` using one of these shapes.

**When the diff is non-empty**, write a header followed by the raw diff inside a
fenced block:

```markdown
# Test file diff (main...HEAD)

```diff
<paste exact `git diff` output here>
```
```

**When the diff is empty** (no `_test.go` files touched), write only:

```markdown
# Test file diff (main...HEAD)

No `_test.go` files were modified on this branch.
```

**When the `git diff` invocation failed**, write only:

```markdown
# Test file diff (main...HEAD)

Diff capture failed: <one-line stderr summary>. No `_test.go` deletions can be
asserted from this run.
```

### Step 3: Stop

Do not modify any other file. Do not commit. Do not run tests. The artifact is
the entire deliverable.

## Constraints

- Touch only `.agents/output/test-diff.md`.
- Do NOT delete or edit existing test files — that is the failure mode this gate
  exists to catch.
- Do NOT spawn sub-agents.
- Keep the diff verbatim — do not summarise, paraphrase, or strip context lines.
